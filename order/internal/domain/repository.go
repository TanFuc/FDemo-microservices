package domain

import (
	"context"
	"time"

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

	// GetDraftsByUserID retrieves all draft orders for a user
	GetDraftsByUserID(ctx context.Context, userID uuid.UUID, limit, offset int) ([]*Order, error)

	// GetByIDAndUserID retrieves an order by ID while verifying ownership
	GetByIDAndUserID(ctx context.Context, orderID, userID uuid.UUID) (*Order, error)

	// DeleteExpiredDrafts soft-deletes DRAFT orders created before the cutoff time.
	// Processes in batches. Returns the number of records deleted in this batch.
	DeleteExpiredDrafts(ctx context.Context, cutoff time.Time, limit int) (int, error)

	// SoftDeleteDraft soft-deletes a single DRAFT order by ID.
	SoftDeleteDraft(ctx context.Context, orderID uuid.UUID) error

	// GetDraftCountByUserID returns the total count of active draft orders for a user.
	GetDraftCountByUserID(ctx context.Context, userID uuid.UUID) (int64, error)
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

// ReturnRepository defines the interface for return request persistence
type ReturnRepository interface {
	// Create creates a new return request
	Create(ctx context.Context, r *ReturnRequest) error

	// GetByID retrieves a return request by its ID
	GetByID(ctx context.Context, id uuid.UUID) (*ReturnRequest, error)

	// GetByOrderID retrieves all return requests for an order
	GetByOrderID(ctx context.Context, orderID uuid.UUID) ([]*ReturnRequest, error)

	// Update updates an existing return request
	Update(ctx context.Context, r *ReturnRequest) error

	// List retrieves return requests with filtering
	List(ctx context.Context, filter ReturnFilter) ([]*ReturnRequest, int64, error)
}

// ReturnFilter for querying return requests
type ReturnFilter struct {
	UserID  uuid.UUID
	OrderID uuid.UUID
	Status  string
	Page    int
	Limit   int
}
