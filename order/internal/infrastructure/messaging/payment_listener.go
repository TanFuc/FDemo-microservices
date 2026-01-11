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

// PaymentProcessedEvent represents the event from Payment Service
type PaymentProcessedEvent struct {
	OrderID       string `json:"order_id"`
	TransactionID string `json:"transaction_id"`
	Status        string `json:"status"` // SUCCESS, FAILED
	PaymentMethod string `json:"payment_method"`
	Amount        string `json:"amount"`
	ProcessedAt   string `json:"processed_at"`
}

// StockConfirmer interface for confirming/releasing stock
type StockConfirmer interface {
	ConfirmStock(ctx context.Context, orderID string) error
	ReleaseStockByOrderID(ctx context.Context, orderID string) error
}

// PaymentEventListener listens to payment events and updates order status
type PaymentEventListener struct {
	conn          *nats.Conn
	js            nats.JetStreamContext
	orderRepo     domain.OrderRepository
	stockClient   StockConfirmer
	publisher     EventPublisher
	subscription  *nats.Subscription
	logger        *slog.Logger
}

// NewPaymentEventListener creates a new payment event listener
func NewPaymentEventListener(
	cfg *config.NATSConfig,
	orderRepo domain.OrderRepository,
	stockClient StockConfirmer,
	publisher EventPublisher,
	logger *slog.Logger,
) (*PaymentEventListener, error) {
	conn, err := nats.Connect(cfg.URL)
	if err != nil {
		return nil, err
	}

	js, err := conn.JetStream()
	if err != nil {
		conn.Close()
		return nil, err
	}

	// Ensure PAYMENTS stream exists for consuming
	_, err = js.StreamInfo("PAYMENTS")
	if err != nil {
		// Create stream if it doesn't exist (for development)
		_, err = js.AddStream(&nats.StreamConfig{
			Name:     "PAYMENTS",
			Subjects: []string{"payment.*"},
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

	return &PaymentEventListener{
		conn:        conn,
		js:          js,
		orderRepo:   orderRepo,
		stockClient: stockClient,
		publisher:   publisher,
		logger:      logger,
	}, nil
}

// Start starts listening to payment events
func (l *PaymentEventListener) Start(ctx context.Context) error {
	// Create durable consumer for payment events
	sub, err := l.js.Subscribe(
		"payment.processed",
		l.handlePaymentProcessed,
		nats.Durable("order-service-payment-consumer"),
		nats.DeliverNew(),
		nats.ManualAck(),
		nats.AckWait(30*time.Second),
	)
	if err != nil {
		return err
	}

	l.subscription = sub
	l.logger.Info("Payment event listener started", "subject", "payment.processed")

	// Wait for context cancellation
	<-ctx.Done()
	return l.Stop()
}

// Stop stops the listener
func (l *PaymentEventListener) Stop() error {
	if l.subscription != nil {
		if err := l.subscription.Unsubscribe(); err != nil {
			l.logger.Error("Failed to unsubscribe", "error", err)
		}
	}
	if l.conn != nil {
		l.conn.Close()
	}
	l.logger.Info("Payment event listener stopped")
	return nil
}

// handlePaymentProcessed handles the payment.processed event
func (l *PaymentEventListener) handlePaymentProcessed(msg *nats.Msg) {
	ctx := context.Background()

	var event PaymentProcessedEvent
	if err := json.Unmarshal(msg.Data, &event); err != nil {
		l.logger.Error("Failed to unmarshal payment event", "error", err)
		msg.Nak()
		return
	}

	l.logger.Info("Received payment event",
		"order_id", event.OrderID,
		"status", event.Status,
		"transaction_id", event.TransactionID,
	)

	orderID, err := uuid.Parse(event.OrderID)
	if err != nil {
		l.logger.Error("Invalid order ID", "order_id", event.OrderID, "error", err)
		msg.Term() // Terminate - won't retry
		return
	}

	// Get current order
	order, err := l.orderRepo.GetByID(ctx, orderID)
	if err != nil {
		l.logger.Error("Failed to get order", "order_id", event.OrderID, "error", err)
		msg.Nak() // Retry later
		return
	}

	// Check if order is still pending
	if order.Status != domain.StatusPending {
		l.logger.Warn("Order is not in pending status, skipping",
			"order_id", event.OrderID,
			"current_status", order.Status,
		)
		msg.Ack() // Already processed
		return
	}

	if event.Status == "SUCCESS" {
		if err := l.handlePaymentSuccess(ctx, order); err != nil {
			l.logger.Error("Failed to handle payment success", "order_id", event.OrderID, "error", err)
			msg.Nak()
			return
		}
	} else {
		if err := l.handlePaymentFailed(ctx, order); err != nil {
			l.logger.Error("Failed to handle payment failure", "order_id", event.OrderID, "error", err)
			msg.Nak()
			return
		}
	}

	msg.Ack()
}

// handlePaymentSuccess handles successful payment
func (l *PaymentEventListener) handlePaymentSuccess(ctx context.Context, order *domain.Order) error {
	// Update order status to PAID
	if err := order.MarkAsPaid(); err != nil {
		return err
	}

	if err := l.orderRepo.Update(ctx, order); err != nil {
		return err
	}

	l.logger.Info("Order marked as paid", "order_id", order.ID)

	// Confirm stock reservation
	if l.stockClient != nil {
		if err := l.stockClient.ConfirmStock(ctx, order.ID.String()); err != nil {
			l.logger.Error("Failed to confirm stock", "order_id", order.ID, "error", err)
			// Continue - stock confirmation can be retried
		} else {
			l.logger.Info("Stock confirmed", "order_id", order.ID)
		}
	}

	// Publish order paid event
	if l.publisher != nil {
		if err := l.publisher.PublishOrderPaid(ctx, &OrderPaidEvent{
			OrderID: order.ID,
			UserID:  order.UserID,
			PaidAt:  time.Now(),
		}); err != nil {
			l.logger.Error("Failed to publish order.paid event", "order_id", order.ID, "error", err)
		}
	}

	return nil
}

// handlePaymentFailed handles failed payment
func (l *PaymentEventListener) handlePaymentFailed(ctx context.Context, order *domain.Order) error {
	// Cancel order
	if err := order.Cancel(); err != nil {
		return err
	}

	if err := l.orderRepo.Update(ctx, order); err != nil {
		return err
	}

	l.logger.Info("Order cancelled due to payment failure", "order_id", order.ID)

	// Release stock reservation
	if l.stockClient != nil {
		if err := l.stockClient.ReleaseStockByOrderID(ctx, order.ID.String()); err != nil {
			l.logger.Error("Failed to release stock", "order_id", order.ID, "error", err)
			// Continue - stock release can be retried
		} else {
			l.logger.Info("Stock released", "order_id", order.ID)
		}
	}

	// Publish order cancelled event
	if l.publisher != nil {
		if err := l.publisher.PublishOrderCancelled(ctx, &OrderCancelledEvent{
			OrderID:        order.ID,
			UserID:         order.UserID,
			ReservationIDs: order.ReservationIDs,
			CancelledAt:    time.Now(),
		}); err != nil {
			l.logger.Error("Failed to publish order.cancelled event", "order_id", order.ID, "error", err)
		}
	}

	return nil
}
