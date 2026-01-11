package port

import (
	"context"

	"microservices/payment/internal/domain"
)

// EventPublisher defines the interface for publishing payment events
type EventPublisher interface {
	// Publish publishes a payment event
	Publish(ctx context.Context, subject string, event *domain.PaymentEvent) error

	// Close closes the publisher connection
	Close() error
}

// Event subjects
const (
	SubjectPaymentProcessed = "payment.processed"
	SubjectPaymentFailed    = "payment.failed"
	SubjectPaymentRefunded  = "payment.refunded"
)
