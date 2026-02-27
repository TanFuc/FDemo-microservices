package domain

import (
	"time"

	"github.com/google/uuid"
)

// NATS subjects
const (
	SubjectWalletToppedUp         = "wallet.topped_up"
	SubjectWalletPaymentCompleted = "wallet.payment.completed"
	SubjectWalletPaymentFailed    = "wallet.payment.failed"
	SubjectWalletRefunded         = "wallet.refunded"
	WalletStreamName              = "WALLET_EVENTS"
)

// WalletEvent is published to NATS after every wallet operation.
type WalletEvent struct {
	EventID       string    `json:"event_id"`
	EventType     string    `json:"event_type"`
	WalletID      uuid.UUID `json:"wallet_id"`
	UserID        uuid.UUID `json:"user_id"`
	TxID          uuid.UUID `json:"tx_id"`
	Amount        string    `json:"amount"`
	BalanceBefore string    `json:"balance_before"`
	BalanceAfter  string    `json:"balance_after"`
	Currency      string    `json:"currency"`
	ReferenceID   string    `json:"reference_id,omitempty"`
	ReferenceType string    `json:"reference_type,omitempty"`
	Timestamp     time.Time `json:"timestamp"`
}

// NewWalletEvent creates a new wallet event from a transaction
func NewWalletEvent(eventType string, tx *WalletTransaction) *WalletEvent {
	return &WalletEvent{
		EventID:       uuid.New().String(),
		EventType:     eventType,
		WalletID:      tx.WalletID,
		UserID:        tx.UserID,
		TxID:          tx.ID,
		Amount:        tx.Amount.String(),
		BalanceBefore: tx.BalanceBefore.String(),
		BalanceAfter:  tx.BalanceAfter.String(),
		Currency:      tx.Currency,
		ReferenceID:   tx.ReferenceID,
		ReferenceType: tx.ReferenceType,
		Timestamp:     time.Now(),
	}
}

// OutboxEvent is persisted in DB for guaranteed delivery via polling worker.
type OutboxEvent struct {
	ID          uuid.UUID
	AggregateID string
	EventType   string
	Payload     []byte
	Status      string // "PENDING", "PUBLISHED", "FAILED"
	RetryCount  int
	CreatedAt   time.Time
	PublishedAt *time.Time
}

// NewOutboxEvent creates a new outbox event
func NewOutboxEvent(aggregateID, eventType string, payload []byte) *OutboxEvent {
	return &OutboxEvent{
		ID:          uuid.New(),
		AggregateID: aggregateID,
		EventType:   eventType,
		Payload:     payload,
		Status:      "PENDING",
		RetryCount:  0,
		CreatedAt:   time.Now(),
	}
}
