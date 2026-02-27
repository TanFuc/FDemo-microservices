package port

import (
	"context"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"

	"microservices/wallet/internal/domain"
)

// WalletRepository defines the interface for wallet data persistence
type WalletRepository interface {
	// Wallet operations
	CreateWallet(ctx context.Context, w *domain.Wallet) error
	GetByUserID(ctx context.Context, userID uuid.UUID) (*domain.Wallet, error)
	GetByID(ctx context.Context, walletID uuid.UUID) (*domain.Wallet, error)
	ExistsByUserID(ctx context.Context, userID uuid.UUID) (bool, error)

	// Optimistic locking balance update.
	// Returns ErrVersionConflict if no rows updated (version mismatch).
	UpdateBalanceOptimistic(ctx context.Context, walletID uuid.UUID, newBalance decimal.Decimal, newVersion, expectedVersion int) error

	// Ledger operations
	CreateTransaction(ctx context.Context, tx *domain.WalletTransaction) error
	GetTransactionByID(ctx context.Context, txID uuid.UUID) (*domain.WalletTransaction, error)
	GetTransactionByIdempotencyKey(ctx context.Context, key string) (*domain.WalletTransaction, error)
	GetTransactionByPaymentTxID(ctx context.Context, paymentTxID uuid.UUID) (*domain.WalletTransaction, error)
	UpdateTransactionStatus(ctx context.Context, txID uuid.UUID, status domain.TxStatus) error
	ListTransactions(ctx context.Context, walletID uuid.UUID, limit, offset int) ([]*domain.WalletTransaction, int64, error)

	// Outbox operations
	CreateOutboxEvent(ctx context.Context, e *domain.OutboxEvent) error
	GetPendingOutboxEvents(ctx context.Context, limit int) ([]*domain.OutboxEvent, error)
	MarkOutboxPublished(ctx context.Context, eventID uuid.UUID) error
	MarkOutboxFailed(ctx context.Context, eventID uuid.UUID) error

	// RunInTransaction executes fn within a single PostgreSQL transaction.
	// Balance update + ledger insert + outbox insert must ALL be in the SAME pg tx.
	RunInTransaction(ctx context.Context, fn func(tx TxContext) error) error
}

// TxContext represents the operations available inside an active DB transaction.
type TxContext interface {
	CreateTransaction(ctx context.Context, tx *domain.WalletTransaction) error
	UpdateBalanceOptimistic(ctx context.Context, walletID uuid.UUID, newBalance decimal.Decimal, newVersion, expectedVersion int) error
	CreateOutboxEvent(ctx context.Context, e *domain.OutboxEvent) error
	UpdateTransactionStatus(ctx context.Context, txID uuid.UUID, status domain.TxStatus) error
}
