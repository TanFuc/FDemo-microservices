package nats

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"time"

	"github.com/google/uuid"
	"github.com/nats-io/nats.go"
	"github.com/nats-io/nats.go/jetstream"

	"tafu-logistic/logistics-service/internal/core/domain"
	"tafu-logistic/logistics-service/internal/core/ports"
	"tafu-logistic/logistics-service/internal/core/services"
)

// PaymentProcessedEvent represents the event from Payment Service
type PaymentProcessedEvent struct {
	OrderID       string `json:"order_id"`
	TransactionID string `json:"transaction_id"`
	Status        string `json:"status"` // SUCCESS, FAILED
	PaymentMethod string `json:"payment_method"`
	Amount        string `json:"amount"`
	ProcessedAt   string `json:"processed_at"`
}

// OrderInfoFetcher interface for fetching order details
type OrderInfoFetcher interface {
	GetOrderInfo(ctx context.Context, orderID string) (*OrderInfo, error)
}

// OrderInfo contains order details needed for shipment creation
type OrderInfo struct {
	OrderID  uuid.UUID
	UserID   uuid.UUID
	Provider domain.ProviderName
	Sender   domain.ContactInfo
	Receiver domain.ContactInfo
	Parcels  []domain.Parcel
	IsCOD    bool
	CODAmount float64
	Note     string
}

// PaymentConsumer handles payment event consumption
type PaymentConsumer struct {
	nc              *nats.Conn
	js              jetstream.JetStream
	shippingService *services.ShippingService
	orderFetcher    OrderInfoFetcher
	consumerCtx     jetstream.ConsumeContext
}

// PaymentConsumerConfig holds configuration for payment consumer
type PaymentConsumerConfig struct {
	URL          string
	ConsumerName string
}

// NewPaymentConsumer creates a new payment event consumer
func NewPaymentConsumer(cfg PaymentConsumerConfig, shippingService *services.ShippingService, orderFetcher OrderInfoFetcher) (*PaymentConsumer, error) {
	nc, err := nats.Connect(cfg.URL,
		nats.RetryOnFailedConnect(true),
		nats.MaxReconnects(10),
		nats.ReconnectWait(2*time.Second),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to NATS: %w", err)
	}

	js, err := jetstream.New(nc)
	if err != nil {
		nc.Close()
		return nil, fmt.Errorf("failed to create JetStream context: %w", err)
	}

	return &PaymentConsumer{
		nc:              nc,
		js:              js,
		shippingService: shippingService,
		orderFetcher:    orderFetcher,
	}, nil
}

// Start begins consuming payment.processed events
func (c *PaymentConsumer) Start(ctx context.Context, cfg PaymentConsumerConfig) error {
	// Get or create stream for payment events
	stream, err := c.js.CreateOrUpdateStream(ctx, jetstream.StreamConfig{
		Name:        "PAYMENTS",
		Description: "Payment service events",
		Subjects:    []string{"payment.>"},
		Retention:   jetstream.LimitsPolicy,
		MaxAge:      24 * time.Hour * 7,
		Storage:     jetstream.FileStorage,
		Replicas:    1,
	})
	if err != nil {
		return fmt.Errorf("failed to create/update stream: %w", err)
	}

	// Create durable consumer
	consumerName := cfg.ConsumerName
	if consumerName == "" {
		consumerName = "logistics-payment-consumer"
	}

	consumer, err := stream.CreateOrUpdateConsumer(ctx, jetstream.ConsumerConfig{
		Name:          consumerName,
		Durable:       consumerName,
		Description:   "Logistics service consumer for payment.processed events",
		FilterSubject: "payment.processed",
		AckPolicy:     jetstream.AckExplicitPolicy,
		MaxDeliver:    5,
		AckWait:       30 * time.Second,
	})
	if err != nil {
		return fmt.Errorf("failed to create consumer: %w", err)
	}

	// Start consuming
	consumerCtx, err := consumer.Consume(func(msg jetstream.Msg) {
		c.handlePaymentProcessed(ctx, msg)
	})
	if err != nil {
		return fmt.Errorf("failed to start consuming: %w", err)
	}

	c.consumerCtx = consumerCtx
	log.Printf("Started consuming payment.processed events for auto-shipment")

	return nil
}

// handlePaymentProcessed processes a payment.processed event
func (c *PaymentConsumer) handlePaymentProcessed(ctx context.Context, msg jetstream.Msg) {
	log.Printf("Received payment.processed event: %s", string(msg.Data()))

	var event PaymentProcessedEvent
	if err := json.Unmarshal(msg.Data(), &event); err != nil {
		log.Printf("Failed to unmarshal payment.processed event: %v", err)
		msg.NakWithDelay(5 * time.Second)
		return
	}

	// Only process successful payments
	if event.Status != "SUCCESS" {
		log.Printf("Payment not successful, skipping shipment creation: order=%s, status=%s", event.OrderID, event.Status)
		msg.Ack()
		return
	}

	// Skip COD orders - they get shipment created when order is packed
	if event.PaymentMethod == "COD" {
		log.Printf("COD order, skipping auto-shipment: order=%s", event.OrderID)
		msg.Ack()
		return
	}

	// Fetch order info
	if c.orderFetcher == nil {
		log.Printf("Order fetcher not configured, skipping shipment creation")
		msg.Ack()
		return
	}

	orderInfo, err := c.orderFetcher.GetOrderInfo(ctx, event.OrderID)
	if err != nil {
		log.Printf("Failed to fetch order info for %s: %v", event.OrderID, err)
		msg.NakWithDelay(10 * time.Second)
		return
	}

	// Create shipment request
	req := &services.CreateShipmentRequest{
		InternalOrderID: orderInfo.OrderID,
		Provider:        orderInfo.Provider,
		Sender:          orderInfo.Sender,
		Receiver:        orderInfo.Receiver,
		Parcels:         orderInfo.Parcels,
		IsCOD:           false,
		CODAmount:       0,
		Note:            orderInfo.Note,
	}

	// Create shipment
	result, err := c.shippingService.CreateShipment(ctx, req)
	if err != nil {
		log.Printf("Failed to create shipment for order %s: %v", event.OrderID, err)
		msg.NakWithDelay(10 * time.Second)
		return
	}

	log.Printf("Auto-created shipment for paid order %s: tracking=%s, fee=%.2f",
		event.OrderID, result.TrackingCode, result.ShippingFee)

	if err := msg.Ack(); err != nil {
		log.Printf("Failed to ack message: %v", err)
	}
}

// Stop stops the consumer
func (c *PaymentConsumer) Stop() {
	if c.consumerCtx != nil {
		c.consumerCtx.Stop()
	}
}

// Close closes the NATS connection
func (c *PaymentConsumer) Close() error {
	c.Stop()
	c.nc.Close()
	return nil
}

// HTTPOrderFetcher fetches order info via HTTP from Order Service
type HTTPOrderFetcher struct {
	baseURL    string
	serviceKey string
}

// NewHTTPOrderFetcher creates a new HTTP order fetcher
func NewHTTPOrderFetcher(baseURL, serviceKey string) *HTTPOrderFetcher {
	return &HTTPOrderFetcher{
		baseURL:    baseURL,
		serviceKey: serviceKey,
	}
}

// GetOrderInfo fetches order info from Order Service
func (f *HTTPOrderFetcher) GetOrderInfo(ctx context.Context, orderID string) (*OrderInfo, error) {
	// Implementation would call Order Service internal API
	// For now, return nil to indicate order fetcher is not fully implemented
	// This requires setting up the internal endpoint in Order Service
	return nil, fmt.Errorf("order fetcher not implemented - requires internal Order Service endpoint")
}

// Ensure HTTPOrderFetcher implements OrderInfoFetcher
var _ OrderInfoFetcher = (*HTTPOrderFetcher)(nil)

// MockOrderFetcher is a mock implementation for development
type MockOrderFetcher struct{}

// GetOrderInfo returns mock order info
func (f *MockOrderFetcher) GetOrderInfo(ctx context.Context, orderID string) (*OrderInfo, error) {
	id, err := uuid.Parse(orderID)
	if err != nil {
		return nil, fmt.Errorf("invalid order ID: %w", err)
	}

	return &OrderInfo{
		OrderID:  id,
		Provider: domain.ProviderMock,
		Sender: domain.ContactInfo{
			Name:    "Test Sender",
			Phone:   "0123456789",
			Address: "123 Test St",
		},
		Receiver: domain.ContactInfo{
			Name:    "Test Receiver",
			Phone:   "0987654321",
			Address: "456 Delivery Ave",
		},
		Parcels: []domain.Parcel{{
			Name:       "Package",
			WeightGram: 1000,
			Quantity:   1,
		}},
		IsCOD:     false,
		CODAmount: 0,
		Note:      "Auto-created from payment",
	}, nil
}

// Ensure MockOrderFetcher implements OrderInfoFetcher
var _ OrderInfoFetcher = (*MockOrderFetcher)(nil)

// ShipmentCreator interface for creating shipments from payment events
type ShipmentCreator interface {
	CreateShipmentFromPayment(ctx context.Context, orderID string, paymentMethod string) error
}

// Ensure ports compatibility
var _ ports.EventPublisher = (*Publisher)(nil)
