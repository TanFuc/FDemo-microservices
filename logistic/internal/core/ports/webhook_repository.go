package ports

import (
	"context"

	"github.com/google/uuid"

	"microservices/logistic/internal/core/domain"
)

// WebhookLogRepository defines the interface for webhook log persistence
type WebhookLogRepository interface {
	// Create persists a new webhook log entry
	Create(ctx context.Context, log *domain.WebhookLog) error

	// GetByID retrieves a webhook log by its ID
	GetByID(ctx context.Context, id uuid.UUID) (*domain.WebhookLog, error)

	// ListByTrackingCode retrieves webhook logs for a tracking code
	ListByTrackingCode(ctx context.Context, trackingCode string, limit, offset int) ([]*domain.WebhookLog, error)

	// ListByShippingOrderID retrieves webhook logs for a shipping order
	ListByShippingOrderID(ctx context.Context, shippingOrderID uuid.UUID, limit, offset int) ([]*domain.WebhookLog, error)
}
