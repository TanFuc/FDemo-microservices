package messaging

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"time"

	"github.com/google/uuid"
	"github.com/nats-io/nats.go"

	"microservices/order/internal/domain"
	inventorygrpc "microservices/order/internal/infrastructure/grpc"
)

// WalletPaymentCompletedEvent mirrors wallet service's WalletEvent
type WalletPaymentCompletedEvent struct {
	EventID       string `json:"event_id"`
	EventType     string `json:"event_type"`
	WalletID      string `json:"wallet_id"`
	UserID        string `json:"user_id"`
	TxID          string `json:"tx_id"`
	Amount        string `json:"amount"`
	BalanceBefore string `json:"balance_before"`
	BalanceAfter  string `json:"balance_after"`
	Currency      string `json:"currency"`
	ReferenceID   string `json:"reference_id"`   // = order_id
	ReferenceType string `json:"reference_type"` // = "order"
	Timestamp     string `json:"timestamp"`
}

// WalletEventListener listens for wallet payment events
type WalletEventListener struct {
	js        nats.JetStreamContext
	nc        *nats.Conn
	orderRepo domain.OrderRepository
	confirmer inventorygrpc.StockConfirmer
	publisher EventPublisher
	logger    *slog.Logger
	sub       *nats.Subscription
}

// NewWalletEventListener creates a new wallet event listener
func NewWalletEventListener(
	cfg *NATSConfig,
	orderRepo domain.OrderRepository,
	confirmer inventorygrpc.StockConfirmer,
	publisher EventPublisher,
	logger *slog.Logger,
) (*WalletEventListener, error) {
	nc, err := nats.Connect(cfg.URL,
		nats.RetryOnFailedConnect(true),
		nats.MaxReconnects(10),
		nats.ReconnectWait(time.Second),
	)
	if err != nil {
		return nil, fmt.Errorf("connect to NATS: %w", err)
	}

	js, err := nc.JetStream()
	if err != nil {
		nc.Close()
		return nil, fmt.Errorf("create JetStream context: %w", err)
	}

	return &WalletEventListener{
		js:        js,
		nc:        nc,
		orderRepo: orderRepo,
		confirmer: confirmer,
		publisher: publisher,
		logger:    logger,
	}, nil
}

// Start starts the wallet event listener
func (l *WalletEventListener) Start(ctx context.Context) error {
	// The "WALLET_EVENTS" stream is created by the wallet service on startup.
	sub, err := l.js.Subscribe(
		"wallet.payment.completed",
		l.handleWalletPayment,
		nats.Durable("order-wallet-payment-consumer"),
		nats.DeliverNew(),
		nats.ManualAck(),
		nats.AckWait(30*time.Second),
		nats.MaxDeliver(5),
	)
	if err != nil {
		return fmt.Errorf("subscribe wallet.payment.completed: %w", err)
	}
	l.sub = sub
	l.logger.Info("Wallet event listener started")

	<-ctx.Done()
	return nil
}

// Stop stops the wallet event listener
func (l *WalletEventListener) Stop() {
	if l.sub != nil {
		l.sub.Unsubscribe()
	}
	if l.nc != nil {
		l.nc.Drain()
	}
}

func (l *WalletEventListener) handleWalletPayment(msg *nats.Msg) {
	var event WalletPaymentCompletedEvent
	if err := json.Unmarshal(msg.Data, &event); err != nil {
		l.logger.Error("failed to unmarshal wallet.payment.completed",
			"error", err,
			"data", string(msg.Data),
		)
		msg.Term() // malformed - do not retry
		return
	}

	// Only handle order payments (reference_type = "order")
	if event.ReferenceType != "order" {
		l.logger.Debug("skipping non-order wallet payment", "reference_type", event.ReferenceType)
		msg.Ack()
		return
	}

	orderID, err := uuid.Parse(event.ReferenceID)
	if err != nil {
		l.logger.Error("invalid order_id in wallet event",
			"reference_id", event.ReferenceID,
			"error", err,
		)
		msg.Term()
		return
	}

	// Get the order
	order, err := l.orderRepo.GetByID(context.Background(), orderID)
	if err != nil {
		l.logger.Error("failed to get order for wallet payment",
			"order_id", orderID,
			"error", err,
		)
		msg.Nak() // retry
		return
	}

	// Idempotent: if already PAID, skip
	if order.PaymentStatus == domain.PaymentPaid {
		l.logger.Info("order already paid (idempotent)",
			"order_id", orderID,
			"payment_status", order.PaymentStatus,
		)
		msg.Ack()
		return
	}

	// Mark order as paid
	order.MarkAsPaid()
	order.PaymentMethod = "WALLET"
	order.PaymentProvider = "WALLET"

	if err := l.orderRepo.Update(context.Background(), order); err != nil {
		l.logger.Error("failed to update order payment status",
			"order_id", orderID,
			"error", err,
		)
		msg.Nak() // retry
		return
	}

	// Confirm stock reservation
	if l.confirmer != nil {
		if err := l.confirmer.ConfirmStock(context.Background(), order.ID.String()); err != nil {
			l.logger.Warn("failed to confirm stock reservation",
				"order_id", orderID,
				"error", err,
			)
			// Don't fail - stock confirmation is best effort
		}
	}

	// Publish order.paid event
	if l.publisher != nil {
		paidEvent := &OrderPaidEvent{
			OrderID: order.ID.String(),
			UserID:  order.UserID.String(),
			PaidAt:  time.Now().Format(time.RFC3339),
		}
		if err := l.publisher.PublishOrderPaid(context.Background(), paidEvent); err != nil {
			l.logger.Warn("failed to publish order.paid event",
				"order_id", orderID,
				"error", err,
			)
			// Don't fail - event publishing is best effort
		}
	}

	l.logger.Info("order marked as paid via wallet",
		"order_id", orderID,
		"wallet_tx_id", event.TxID,
		"amount", event.Amount,
	)

	msg.Ack()
}
