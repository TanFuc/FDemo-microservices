package messaging

import (
	"context"
	"encoding/json"
	"log/slog"
	"time"

	"github.com/google/uuid"
	"github.com/nats-io/nats.go"
	"github.com/tafu/order-service/internal/config"
	"github.com/tafu/order-service/internal/domain"
)

// LogisticsStatusEvent represents the event from Logistics Service
type LogisticsStatusEvent struct {
	InternalOrderID string `json:"internal_order_id"`
	TrackingCode    string `json:"tracking_code"`
	Provider        string `json:"provider"`
	SystemStatus    string `json:"system_status"` // PENDING, SHIPPING, DELIVERED, RETURNED
	CarrierStatus   string `json:"carrier_status"`
	UpdatedAt       string `json:"updated_at"`
}

// OrderDeliveredEvent is published when an order is delivered
type OrderDeliveredEvent struct {
	OrderID    uuid.UUID `json:"order_id"`
	UserID     uuid.UUID `json:"user_id"`
	ProductIDs []string  `json:"product_ids"`
}

// LogisticsEventListener listens to logistics events and updates order status
type LogisticsEventListener struct {
	conn         *nats.Conn
	js           nats.JetStreamContext
	orderRepo    domain.OrderRepository
	publisher    EventPublisher
	subscription *nats.Subscription
	logger       *slog.Logger
}

// NewLogisticsEventListener creates a new logistics event listener
func NewLogisticsEventListener(
	cfg *config.NATSConfig,
	orderRepo domain.OrderRepository,
	publisher EventPublisher,
	logger *slog.Logger,
) (*LogisticsEventListener, error) {
	conn, err := nats.Connect(cfg.URL)
	if err != nil {
		return nil, err
	}

	js, err := conn.JetStream()
	if err != nil {
		conn.Close()
		return nil, err
	}

	// Ensure LOGISTICS stream exists for consuming
	_, err = js.StreamInfo("LOGISTICS")
	if err != nil {
		// Create stream if it doesn't exist (for development)
		_, err = js.AddStream(&nats.StreamConfig{
			Name:     "LOGISTICS",
			Subjects: []string{"logistics.*"},
			Storage:  nats.FileStorage,
		})
		if err != nil {
			conn.Close()
			return nil, err
		}
	}

	if logger == nil {
		logger = slog.Default()
	}

	return &LogisticsEventListener{
		conn:      conn,
		js:        js,
		orderRepo: orderRepo,
		publisher: publisher,
		logger:    logger,
	}, nil
}

// Start starts listening to logistics events
func (l *LogisticsEventListener) Start(ctx context.Context) error {
	// Create durable consumer for logistics status events
	sub, err := l.js.Subscribe(
		"logistics.status.updated",
		l.handleLogisticsStatusUpdated,
		nats.Durable("order-service-logistics-consumer"),
		nats.DeliverNew(),
		nats.ManualAck(),
		nats.AckWait(30*time.Second),
	)
	if err != nil {
		return err
	}

	l.subscription = sub
	l.logger.Info("Logistics event listener started", "subject", "logistics.status.updated")

	<-ctx.Done()
	return l.Stop()
}

// Stop stops the listener
func (l *LogisticsEventListener) Stop() error {
	if l.subscription != nil {
		if err := l.subscription.Unsubscribe(); err != nil {
			l.logger.Error("Failed to unsubscribe", "error", err)
		}
	}
	if l.conn != nil {
		l.conn.Close()
	}
	l.logger.Info("Logistics event listener stopped")
	return nil
}

// handleLogisticsStatusUpdated handles the logistics.status.updated event
func (l *LogisticsEventListener) handleLogisticsStatusUpdated(msg *nats.Msg) {
	ctx := context.Background()

	var event LogisticsStatusEvent
	if err := json.Unmarshal(msg.Data, &event); err != nil {
		l.logger.Error("Failed to unmarshal logistics event", "error", err)
		msg.Nak()
		return
	}

	l.logger.Info("Received logistics status event",
		"order_id", event.InternalOrderID,
		"status", event.SystemStatus,
		"tracking", event.TrackingCode,
	)

	orderID, err := uuid.Parse(event.InternalOrderID)
	if err != nil {
		l.logger.Error("Invalid order ID", "order_id", event.InternalOrderID, "error", err)
		msg.Term()
		return
	}

	// Get current order
	order, err := l.orderRepo.GetByID(ctx, orderID)
	if err != nil {
		l.logger.Error("Failed to get order", "order_id", event.InternalOrderID, "error", err)
		msg.Nak()
		return
	}

	// Handle status transitions
	switch event.SystemStatus {
	case "SHIPPING":
		if err := l.handleShippingStatus(ctx, order); err != nil {
			l.logger.Error("Failed to handle shipping status", "order_id", event.InternalOrderID, "error", err)
			msg.Nak()
			return
		}
	case "DELIVERED":
		if err := l.handleDeliveredStatus(ctx, order); err != nil {
			l.logger.Error("Failed to handle delivered status", "order_id", event.InternalOrderID, "error", err)
			msg.Nak()
			return
		}
	}

	msg.Ack()
}

// handleShippingStatus handles when order starts shipping
func (l *LogisticsEventListener) handleShippingStatus(ctx context.Context, order *domain.Order) error {
	// Only transition from PAID status
	if order.Status != domain.StatusPaid {
		l.logger.Warn("Order is not in PAID status, skipping shipping transition",
			"order_id", order.ID,
			"current_status", order.Status,
		)
		return nil
	}

	if err := order.MarkAsShipped(); err != nil {
		return err
	}

	if err := l.orderRepo.Update(ctx, order); err != nil {
		return err
	}

	l.logger.Info("Order marked as shipped", "order_id", order.ID)

	// Publish order shipped event
	if l.publisher != nil {
		if err := l.publisher.PublishOrderShipped(ctx, &OrderShippedEvent{
			OrderID:   order.ID,
			UserID:    order.UserID,
			ShippedAt: time.Now(),
		}); err != nil {
			l.logger.Error("Failed to publish order.shipped event", "order_id", order.ID, "error", err)
		}
	}

	return nil
}

// handleDeliveredStatus handles when order is delivered
func (l *LogisticsEventListener) handleDeliveredStatus(ctx context.Context, order *domain.Order) error {
	// Only transition from SHIPPED status
	if order.Status != domain.StatusShipped {
		l.logger.Warn("Order is not in SHIPPED status, skipping delivered transition",
			"order_id", order.ID,
			"current_status", order.Status,
		)
		return nil
	}

	if err := order.MarkAsCompleted(); err != nil {
		return err
	}

	if err := l.orderRepo.Update(ctx, order); err != nil {
		return err
	}

	l.logger.Info("Order marked as completed", "order_id", order.ID)

	// Publish order completed event
	if l.publisher != nil {
		if err := l.publisher.PublishOrderCompleted(ctx, &OrderCompletedEvent{
			OrderID:     order.ID,
			UserID:      order.UserID,
			CompletedAt: time.Now(),
		}); err != nil {
			l.logger.Error("Failed to publish order.completed event", "order_id", order.ID, "error", err)
		}
	}

	// Publish order.delivered event to enable reviews
	if l.publisher != nil {
		productIDs := make([]string, len(order.Items))
		for i, item := range order.Items {
			productIDs[i] = item.ProductID
		}

		deliveredEvent := &OrderDeliveredEvent{
			OrderID:    order.ID,
			UserID:     order.UserID,
			ProductIDs: productIDs,
		}

		data, _ := json.Marshal(deliveredEvent)
		if _, err := l.js.Publish("order.delivered", data); err != nil {
			l.logger.Error("Failed to publish order.delivered event", "order_id", order.ID, "error", err)
		} else {
			l.logger.Info("Published order.delivered event for review enablement", "order_id", order.ID)
		}
	}

	return nil
}
