package port

import (
	"context"

	"microservices/wallet/internal/domain"
)

// EventPublisher defines the interface for publishing wallet events
type EventPublisher interface {
	// Publish publishes a wallet event to the specified subject
	Publish(ctx context.Context, subject string, event *domain.WalletEvent) error

	// PublishRaw publishes raw bytes to the specified subject
	PublishRaw(ctx context.Context, subject string, payload []byte) error

	// Close closes the publisher connection
	Close() error
}
