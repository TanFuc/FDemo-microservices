package domain

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

// PaymentTransaction represents a payment transaction in the ledger
type PaymentTransaction struct {
	ID            uuid.UUID       `json:"id"`
	OrderID       uuid.UUID       `json:"order_id"`
	UserID        uuid.UUID       `json:"user_id"`
	Amount        decimal.Decimal `json:"amount"`
	Currency      Currency        `json:"currency"`
	Provider      Provider        `json:"provider"`
	ProviderTxID  string          `json:"provider_tx_id,omitempty"`
	Status        Status          `json:"status"`
	IsWalletTopup bool            `json:"is_wallet_topup"`
	WalletTxID    *string         `json:"wallet_tx_id,omitempty"`
	Metadata      json.RawMessage `json:"metadata,omitempty"`
	CreatedAt     time.Time       `json:"created_at"`
	UpdatedAt     time.Time       `json:"updated_at"`
}

// NewPaymentTransaction creates a new payment transaction
func NewPaymentTransaction(orderID, userID uuid.UUID, amount decimal.Decimal, currency Currency, provider Provider) *PaymentTransaction {
	now := time.Now()
	return &PaymentTransaction{
		ID:        uuid.New(),
		OrderID:   orderID,
		UserID:    userID,
		Amount:    amount,
		Currency:  currency,
		Provider:  provider,
		Status:    StatusPending,
		Metadata:  json.RawMessage("{}"),
		CreatedAt: now,
		UpdatedAt: now,
	}
}

// SetProviderTxID sets the provider transaction ID
func (p *PaymentTransaction) SetProviderTxID(txID string) {
	p.ProviderTxID = txID
	p.UpdatedAt = time.Now()
}

// UpdateStatus updates the transaction status
func (p *PaymentTransaction) UpdateStatus(status Status) {
	p.Status = status
	p.UpdatedAt = time.Now()
}

// SetMetadata sets the metadata
func (p *PaymentTransaction) SetMetadata(data map[string]interface{}) error {
	bytes, err := json.Marshal(data)
	if err != nil {
		return err
	}
	p.Metadata = bytes
	p.UpdatedAt = time.Now()
	return nil
}

// GetMetadata returns the metadata as a map
func (p *PaymentTransaction) GetMetadata() (map[string]interface{}, error) {
	var result map[string]interface{}
	if err := json.Unmarshal(p.Metadata, &result); err != nil {
		return nil, err
	}
	return result, nil
}

// PaymentLog represents an audit log entry
type PaymentLog struct {
	ID            uuid.UUID       `json:"id"`
	TransactionID *uuid.UUID      `json:"transaction_id,omitempty"`
	Provider      Provider        `json:"provider"`
	EventType     string          `json:"event_type"`
	RawPayload    json.RawMessage `json:"raw_payload"`
	IPAddress     string          `json:"ip_address,omitempty"`
	CreatedAt     time.Time       `json:"created_at"`
}

// NewPaymentLog creates a new payment log entry
func NewPaymentLog(provider Provider, eventType string, payload json.RawMessage, ipAddress string) *PaymentLog {
	return &PaymentLog{
		ID:         uuid.New(),
		Provider:   provider,
		EventType:  eventType,
		RawPayload: payload,
		IPAddress:  ipAddress,
		CreatedAt:  time.Now(),
	}
}

// SetTransactionID sets the associated transaction ID
func (l *PaymentLog) SetTransactionID(txID uuid.UUID) {
	l.TransactionID = &txID
}

// PaymentEvent represents an event to be published
type PaymentEvent struct {
	OrderID       uuid.UUID `json:"order_id"`
	TransactionID uuid.UUID `json:"transaction_id"`
	Status        Status    `json:"status"`
	Provider      Provider  `json:"provider"`
	Amount        string    `json:"amount"`
	Currency      Currency  `json:"currency"`
	IsWalletTopup bool      `json:"is_wallet_topup"`
	WalletTxID    string    `json:"wallet_tx_id,omitempty"`
	Timestamp     time.Time `json:"timestamp"`
}

// NewPaymentEvent creates a new payment event from a transaction
func NewPaymentEvent(tx *PaymentTransaction) *PaymentEvent {
	event := &PaymentEvent{
		OrderID:       tx.OrderID,
		TransactionID: tx.ID,
		Status:        tx.Status,
		Provider:      tx.Provider,
		Amount:        tx.Amount.String(),
		Currency:      tx.Currency,
		IsWalletTopup: tx.IsWalletTopup,
		Timestamp:     time.Now(),
	}
	if tx.WalletTxID != nil {
		event.WalletTxID = *tx.WalletTxID
	}
	return event
}
