package ports

import (
	"context"

	"github.com/google/uuid"

	"tafu-logistic/logistics-service/internal/core/domain"
)

// ShippingOrderRepository defines the interface for shipping order persistence
type ShippingOrderRepository interface {
	// Create persists a new shipping order
	Create(ctx context.Context, order *domain.ShippingOrder) error

	// Update updates an existing shipping order
	Update(ctx context.Context, order *domain.ShippingOrder) error

	// GetByID retrieves a shipping order by its ID
	GetByID(ctx context.Context, id uuid.UUID) (*domain.ShippingOrder, error)

	// GetByInternalOrderID retrieves a shipping order by internal order ID
	GetByInternalOrderID(ctx context.Context, internalOrderID uuid.UUID) (*domain.ShippingOrder, error)

	// GetByTrackingCode retrieves a shipping order by tracking code
	GetByTrackingCode(ctx context.Context, trackingCode string) (*domain.ShippingOrder, error)

	// ListByStatus retrieves shipping orders by system status
	ListByStatus(ctx context.Context, status domain.SystemStatus, limit, offset int) ([]*domain.ShippingOrder, error)

	// UpdateStatus updates only the status fields of a shipping order
	UpdateStatus(ctx context.Context, id uuid.UUID, carrierStatus string, systemStatus domain.SystemStatus) error
}
