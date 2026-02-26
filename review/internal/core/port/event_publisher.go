package port

import (
	"context"
	"microservices/review/internal/adapter/nats"
)

// EventPublisher defines the contract for publishing domain events
type EventPublisher interface {
	PublishReviewCreated(ctx context.Context, event *nats.ReviewCreatedEvent) error
	PublishReviewUpdated(ctx context.Context, event *nats.ReviewCreatedEvent) error
	PublishRatingUpdated(ctx context.Context, event *nats.RatingUpdatedEvent) error
}
