package messaging

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"time"

	"github.com/google/uuid"
	"github.com/nats-io/nats.go"
	"github.com/shopspring/decimal"
)

// PaymentProcessedEvent mirrors payment service's PaymentEvent
type PaymentProcessedEvent struct {
	OrderID       string `json:"order_id"`
	TransactionID string `json:"transaction_id"`
	Status        string `json:"status"` // "SUCCESS" or "FAILED"
	Amount        string `json:"amount"`
	Currency      string `json:"currency"`
	IsWalletTopup bool   `json:"is_wallet_topup"`
	WalletTxID    string `json:"wallet_tx_id,omitempty"`
	Timestamp     string `json:"timestamp"`
}

// CompleteTopUpUseCase interface for completing top-ups
type CompleteTopUpUseCase interface {
	Execute(ctx context.Context, paymentTxID uuid.UUID, amount decimal.Decimal) error
}

// PaymentEventListener listens for payment.processed events
type PaymentEventListener struct {
	js              nats.JetStreamContext
	completeTopUpUC CompleteTopUpUseCase
	logger          *slog.Logger
	sub             *nats.Subscription
}

// NewPaymentEventListener creates a new payment event listener
func NewPaymentEventListener(url string, completeTopUpUC CompleteTopUpUseCase, logger *slog.Logger) (*PaymentEventListener, error) {
	nc, err := nats.Connect(url,
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

	return &PaymentEventListener{
		js:              js,
		completeTopUpUC: completeTopUpUC,
		logger:          logger,
	}, nil
}

// Start starts the listener
func (l *PaymentEventListener) Start(ctx context.Context) error {
	// The "PAYMENTS" stream is already created by the payment service.
	sub, err := l.js.Subscribe(
		"payment.processed",
		l.handle,
		nats.Durable("wallet-topup-completion-consumer"),
		nats.DeliverNew(),
		nats.ManualAck(),
		nats.AckWait(30*time.Second),
		nats.MaxDeliver(5),
	)
	if err != nil {
		return fmt.Errorf("subscribe payment.processed: %w", err)
	}
	l.sub = sub
	l.logger.Info("Payment event listener started")

	<-ctx.Done()
	return nil
}

// Stop stops the listener
func (l *PaymentEventListener) Stop() {
	if l.sub != nil {
		l.sub.Unsubscribe()
	}
}

func (l *PaymentEventListener) handle(msg *nats.Msg) {
	var event PaymentProcessedEvent
	if err := json.Unmarshal(msg.Data, &event); err != nil {
		l.logger.Error("failed to unmarshal payment.processed", "error", err)
		msg.Term() // malformed - do not retry
		return
	}

	// Only handle successful wallet top-up payments
	if event.Status != "SUCCESS" || !event.IsWalletTopup {
		msg.Ack()
		return
	}

	paymentTxID, err := uuid.Parse(event.TransactionID)
	if err != nil {
		l.logger.Error("invalid transaction_id in event", "transaction_id", event.TransactionID, "error", err)
		msg.Term()
		return
	}

	amount, err := decimal.NewFromString(event.Amount)
	if err != nil {
		l.logger.Error("invalid amount in event", "amount", event.Amount, "error", err)
		msg.Term()
		return
	}

	if err := l.completeTopUpUC.Execute(context.Background(), paymentTxID, amount); err != nil {
		l.logger.Error("failed to complete top-up", "payment_tx_id", event.TransactionID, "error", err)
		msg.Nak() // retry
		return
	}

	l.logger.Info("top-up completed", "payment_tx_id", event.TransactionID, "amount", event.Amount)
	msg.Ack()
}
