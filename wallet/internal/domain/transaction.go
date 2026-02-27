package domain

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

type TxType string
type TxStatus string

const (
	TxTypeTopUp      TxType = "TOP_UP"
	TxTypePayment    TxType = "PAYMENT"
	TxTypeRefund     TxType = "REFUND"
	TxTypeAdjustment TxType = "ADJUSTMENT"

	TxStatusPending   TxStatus = "PENDING"
	TxStatusCompleted TxStatus = "COMPLETED"
	TxStatusFailed    TxStatus = "FAILED"
	TxStatusReversed  TxStatus = "REVERSED"
)

// WalletTransaction is an immutable ledger entry. Never UPDATE amount or type.
type WalletTransaction struct {
	ID             uuid.UUID
	WalletID       uuid.UUID
	UserID         uuid.UUID
	Type           TxType
	Status         TxStatus
	Amount         decimal.Decimal // always positive
	BalanceBefore  decimal.Decimal
	BalanceAfter   decimal.Decimal
	Currency       string
	ReferenceID    string     // order_id or payment_tx_id
	ReferenceType  string     // "order" or "payment"
	Description    string
	PaymentTxID    *uuid.UUID // for TOP_UP: links back to payment service tx
	IdempotencyKey string     // UNIQUE - prevents duplicate processing
	Metadata       json.RawMessage
	CreatedAt      time.Time
	UpdatedAt      time.Time
}

// NewWalletTransaction creates a new wallet transaction
func NewWalletTransaction(
	walletID, userID uuid.UUID,
	txType TxType,
	amount, balanceBefore, balanceAfter decimal.Decimal,
	currency string,
) *WalletTransaction {
	now := time.Now()
	return &WalletTransaction{
		ID:            uuid.New(),
		WalletID:      walletID,
		UserID:        userID,
		Type:          txType,
		Status:        TxStatusPending,
		Amount:        amount,
		BalanceBefore: balanceBefore,
		BalanceAfter:  balanceAfter,
		Currency:      currency,
		Metadata:      json.RawMessage("{}"),
		CreatedAt:     now,
		UpdatedAt:     now,
	}
}

// MarkCompleted marks the transaction as completed
func (t *WalletTransaction) MarkCompleted() {
	t.Status = TxStatusCompleted
	t.UpdatedAt = time.Now()
}

// MarkFailed marks the transaction as failed
func (t *WalletTransaction) MarkFailed() {
	t.Status = TxStatusFailed
	t.UpdatedAt = time.Now()
}

// SetReference sets the reference ID and type
func (t *WalletTransaction) SetReference(refID, refType string) {
	t.ReferenceID = refID
	t.ReferenceType = refType
}

// SetPaymentTxID sets the payment transaction ID for top-ups
func (t *WalletTransaction) SetPaymentTxID(paymentTxID uuid.UUID) {
	t.PaymentTxID = &paymentTxID
}

// SetIdempotencyKey sets the idempotency key
func (t *WalletTransaction) SetIdempotencyKey(key string) {
	t.IdempotencyKey = key
}

// SetDescription sets the transaction description
func (t *WalletTransaction) SetDescription(desc string) {
	t.Description = desc
}

// SetMetadata sets the transaction metadata
func (t *WalletTransaction) SetMetadata(data map[string]interface{}) error {
	bytes, err := json.Marshal(data)
	if err != nil {
		return err
	}
	t.Metadata = bytes
	return nil
}

// IsFinal returns true if the transaction is in a final state
func (t *WalletTransaction) IsFinal() bool {
	return t.Status == TxStatusCompleted || t.Status == TxStatusFailed || t.Status == TxStatusReversed
}
