package domain

import (
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

type WalletStatus string

const (
	WalletStatusActive    WalletStatus = "ACTIVE"
	WalletStatusSuspended WalletStatus = "SUSPENDED"
	WalletStatusFrozen    WalletStatus = "FROZEN"
)

// Wallet represents a user's internal wallet
type Wallet struct {
	ID        uuid.UUID
	UserID    uuid.UUID
	Balance   decimal.Decimal
	Currency  string
	Status    WalletStatus
	CreatedAt time.Time
	UpdatedAt time.Time
	Version   int // used for optimistic locking
}

// NewWallet creates a new wallet for a user
func NewWallet(userID uuid.UUID) *Wallet {
	now := time.Now()
	return &Wallet{
		ID:        uuid.New(),
		UserID:    userID,
		Balance:   decimal.Zero,
		Currency:  "VND",
		Status:    WalletStatusActive,
		Version:   1,
		CreatedAt: now,
		UpdatedAt: now,
	}
}

// CanTransact checks if wallet can perform transactions
func (w *Wallet) CanTransact() error {
	switch w.Status {
	case WalletStatusSuspended:
		return ErrWalletSuspended
	case WalletStatusFrozen:
		return ErrWalletFrozen
	}
	return nil
}

// Credit adds funds to wallet. Returns (balanceBefore, balanceAfter, error).
func (w *Wallet) Credit(amount decimal.Decimal) (decimal.Decimal, decimal.Decimal, error) {
	if amount.LessThanOrEqual(decimal.Zero) {
		return decimal.Zero, decimal.Zero, ErrInvalidAmount
	}
	if err := w.CanTransact(); err != nil {
		return decimal.Zero, decimal.Zero, err
	}
	before := w.Balance
	w.Balance = w.Balance.Add(amount)
	w.UpdatedAt = time.Now()
	w.Version++
	return before, w.Balance, nil
}

// Debit subtracts funds from wallet. Returns (balanceBefore, balanceAfter, error).
func (w *Wallet) Debit(amount decimal.Decimal) (decimal.Decimal, decimal.Decimal, error) {
	if amount.LessThanOrEqual(decimal.Zero) {
		return decimal.Zero, decimal.Zero, ErrInvalidAmount
	}
	if err := w.CanTransact(); err != nil {
		return decimal.Zero, decimal.Zero, err
	}
	if w.Balance.LessThan(amount) {
		return decimal.Zero, decimal.Zero, ErrInsufficientBalance
	}
	before := w.Balance
	w.Balance = w.Balance.Sub(amount)
	w.UpdatedAt = time.Now()
	w.Version++
	return before, w.Balance, nil
}

// IsActive returns true if the wallet is in active status
func (w *Wallet) IsActive() bool {
	return w.Status == WalletStatusActive
}
