package domain

import "errors"

var (
	ErrWalletNotFound      = errors.New("wallet not found")
	ErrWalletAlreadyExists = errors.New("wallet already exists for this user")
	ErrWalletSuspended     = errors.New("wallet is suspended")
	ErrWalletFrozen        = errors.New("wallet is frozen")
	ErrInsufficientBalance = errors.New("insufficient wallet balance")
	ErrInvalidAmount       = errors.New("amount must be positive")
	ErrDuplicateTx         = errors.New("duplicate transaction (idempotency key already exists)")
	ErrVersionConflict     = errors.New("wallet version conflict - please retry")
	ErrPaymentTxNotFound   = errors.New("payment transaction not found")
	ErrTransactionNotFound = errors.New("transaction not found")
	ErrAmountTooSmall      = errors.New("amount is below minimum limit")
	ErrAmountTooLarge      = errors.New("amount exceeds maximum limit")
	ErrBalanceExceedsLimit = errors.New("balance would exceed maximum limit")
)
