package port

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/tafu/payment-service/internal/domain"
)

// PaymentRepository defines the interface for payment data persistence
type PaymentRepository interface {
	// Create creates a new payment transaction
	Create(ctx context.Context, tx *domain.PaymentTransaction) error

	// Update updates an existing payment transaction
	Update(ctx context.Context, tx *domain.PaymentTransaction) error

	// GetByID retrieves a payment transaction by ID
	GetByID(ctx context.Context, id uuid.UUID) (*domain.PaymentTransaction, error)

	// GetByOrderID retrieves payment transactions by order ID
	GetByOrderID(ctx context.Context, orderID uuid.UUID) ([]*domain.PaymentTransaction, error)

	// GetByProviderTxID retrieves a payment transaction by provider transaction ID
	GetByProviderTxID(ctx context.Context, provider domain.Provider, providerTxID string) (*domain.PaymentTransaction, error)

	// GetPendingTransactions retrieves pending transactions older than the given time
	GetPendingTransactions(ctx context.Context, olderThan time.Time) ([]*domain.PaymentTransaction, error)

	// CreateLog creates a payment log entry
	CreateLog(ctx context.Context, log *domain.PaymentLog) error

	// GetLogsByTransactionID retrieves logs for a transaction
	GetLogsByTransactionID(ctx context.Context, txID uuid.UUID) ([]*domain.PaymentLog, error)
}
