package repository

import (
	"context"

	"github.com/google/uuid"

	"microservices/template-service/internal/model"
)

// ItemRepository defines the interface for item data access
type ItemRepository interface {
	// Create creates a new item
	Create(ctx context.Context, item *model.Item) error

	// GetByID retrieves an item by ID
	GetByID(ctx context.Context, id uuid.UUID) (*model.Item, error)

	// Update updates an existing item
	Update(ctx context.Context, item *model.Item) error

	// Delete removes an item by ID
	Delete(ctx context.Context, id uuid.UUID) error

	// List retrieves items with pagination
	List(ctx context.Context, offset, limit int) ([]model.Item, int64, error)

	// ExistsByID checks if an item exists
	ExistsByID(ctx context.Context, id uuid.UUID) (bool, error)
}
