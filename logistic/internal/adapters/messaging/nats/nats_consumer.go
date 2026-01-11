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
	"tafu-logistic/logistics-service/internal/core/services"
)

// OrderPackedEvent represents the event payload from order service
type OrderPackedEvent struct {
	OrderID    uuid.UUID          `json:"order_id"`
	Provider   string             `json:"provider"`
	Sender     domain.ContactInfo `json:"sender"`
	Receiver   domain.ContactInfo `json:"receiver"`
	Parcels    []domain.Parcel    `json:"parcels"`
	IsCOD      bool               `json:"is_cod"`
	CODAmount  float64            `json:"cod_amount"`
	Note       string             `json:"note"`
}

// Consumer handles NATS message consumption
type Consumer struct {
	nc              *nats.Conn
	js              jetstream.JetStream
	shippingService *services.ShippingService
	consumerCtx     jetstream.ConsumeContext
}

// ConsumerConfig holds NATS consumer configuration
type ConsumerConfig struct {
	URL          string
	StreamName   string
	ConsumerName string
	Subject      string
}

// NewConsumer creates a new NATS consumer
func NewConsumer(cfg ConsumerConfig, shippingService *services.ShippingService) (*Consumer, error) {
	// Connect to NATS
	nc, err := nats.Connect(cfg.URL,
		nats.RetryOnFailedConnect(true),
		nats.MaxReconnects(10),
		nats.ReconnectWait(2*time.Second),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to NATS: %w", err)
	}

	// Create JetStream context
	js, err := jetstream.New(nc)
	if err != nil {
		nc.Close()
		return nil, fmt.Errorf("failed to create JetStream context: %w", err)
	}

	return &Consumer{
		nc:              nc,
		js:              js,
		shippingService: shippingService,
	}, nil
}

// Start begins consuming messages from the order.packed subject
func (c *Consumer) Start(ctx context.Context, cfg ConsumerConfig) error {
	// Get or create stream for order events
	stream, err := c.js.CreateOrUpdateStream(ctx, jetstream.StreamConfig{
		Name:        "ORDERS",
		Description: "Order service events",
		Subjects:    []string{"order.>"},
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
		consumerName = "logistics-service"
	}

	consumer, err := stream.CreateOrUpdateConsumer(ctx, jetstream.ConsumerConfig{
		Name:          consumerName,
		Durable:       consumerName,
		Description:   "Logistics service consumer for order.packed events",
		FilterSubject: "order.packed",
		AckPolicy:     jetstream.AckExplicitPolicy,
		MaxDeliver:    5,
		AckWait:       30 * time.Second,
	})
	if err != nil {
		return fmt.Errorf("failed to create consumer: %w", err)
	}

	// Start consuming
	consumerCtx, err := consumer.Consume(func(msg jetstream.Msg) {
		c.handleOrderPacked(ctx, msg)
	})
	if err != nil {
		return fmt.Errorf("failed to start consuming: %w", err)
	}

	c.consumerCtx = consumerCtx
	log.Printf("Started consuming order.packed events")

	return nil
}

// handleOrderPacked processes an order.packed event
func (c *Consumer) handleOrderPacked(ctx context.Context, msg jetstream.Msg) {
	log.Printf("Received order.packed event: %s", string(msg.Data()))

	var event OrderPackedEvent
	if err := json.Unmarshal(msg.Data(), &event); err != nil {
		log.Printf("Failed to unmarshal order.packed event: %v", err)
		// Nak with delay for retry
		msg.NakWithDelay(5 * time.Second)
		return
	}

	// Validate provider
	providerName := domain.ProviderName(event.Provider)
	if !providerName.IsValid() {
		log.Printf("Invalid provider in order.packed event: %s", event.Provider)
		// Use mock provider as fallback
		providerName = domain.ProviderMock
	}

	// Create shipment request
	req := &services.CreateShipmentRequest{
		InternalOrderID: event.OrderID,
		Provider:        providerName,
		Sender:          event.Sender,
		Receiver:        event.Receiver,
		Parcels:         event.Parcels,
		IsCOD:           event.IsCOD,
		CODAmount:       event.CODAmount,
		Note:            event.Note,
	}

	// Create shipment
	result, err := c.shippingService.CreateShipment(ctx, req)
	if err != nil {
		log.Printf("Failed to create shipment for order %s: %v", event.OrderID, err)
		// Nak with delay for retry
		msg.NakWithDelay(10 * time.Second)
		return
	}

	log.Printf("Created shipment for order %s: tracking=%s, fee=%.2f",
		event.OrderID, result.TrackingCode, result.ShippingFee)

	// Acknowledge the message
	if err := msg.Ack(); err != nil {
		log.Printf("Failed to ack message: %v", err)
	}
}

// Stop stops the consumer
func (c *Consumer) Stop() {
	if c.consumerCtx != nil {
		c.consumerCtx.Stop()
	}
}

// Close closes the NATS connection
func (c *Consumer) Close() error {
	c.Stop()
	c.nc.Close()
	return nil
}
