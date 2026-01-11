package domain

import (
	"context"

	"github.com/google/uuid"
)

// OrderRepository defines the interface for order persistence operations
type OrderRepository interface {
	// Create creates a new order with its items in a single transaction
	Create(ctx context.Context, order *Order) error

	// GetByID retrieves an order by its ID with items
	GetByID(ctx context.Context, id uuid.UUID) (*Order, error)

	// GetByUserID retrieves all orders for a user
	GetByUserID(ctx context.Context, userID uuid.UUID, limit, offset int) ([]*Order, error)

	// Update updates an existing order
	Update(ctx context.Context, order *Order) error

	// UpdateStatus updates only the order status
	UpdateStatus(ctx context.Context, id uuid.UUID, status OrderStatus) error
}

// UnitOfWork provides transaction support
type UnitOfWork interface {
	// Begin starts a new transaction
	Begin(ctx context.Context) (context.Context, error)

	// Commit commits the current transaction
	Commit(ctx context.Context) error

	// Rollback rolls back the current transaction
	Rollback(ctx context.Context) error
}
