package publisher

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/nats-io/nats.go"
	"github.com/nats-io/nats.go/jetstream"
	"microservices/payment/internal/domain"
	"microservices/payment/internal/port"
)

const (
	streamName = "PAYMENTS"
)

// NATSPublisher implements port.EventPublisher using NATS JetStream
type NATSPublisher struct {
	nc *nats.Conn
	js jetstream.JetStream
}

// NewNATSPublisher creates a new NATS JetStream publisher
func NewNATSPublisher(ctx context.Context, url string) (*NATSPublisher, error) {
	nc, err := nats.Connect(url,
		nats.RetryOnFailedConnect(true),
		nats.MaxReconnects(10),
		nats.ReconnectWait(time.Second),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to NATS: %w", err)
	}

	js, err := jetstream.New(nc)
	if err != nil {
		nc.Close()
		return nil, fmt.Errorf("failed to create JetStream context: %w", err)
	}

	// Create or update the stream
	_, err = js.CreateOrUpdateStream(ctx, jetstream.StreamConfig{
		Name:        streamName,
		Description: "Payment events stream",
		Subjects:    []string{"payment.>"},
		Retention:   jetstream.WorkQueuePolicy,
		MaxAge:      24 * time.Hour * 7, // 7 days retention
		Storage:     jetstream.FileStorage,
		Replicas:    1,
	})
	if err != nil {
		nc.Close()
		return nil, fmt.Errorf("failed to create stream: %w", err)
	}

	return &NATSPublisher{
		nc: nc,
		js: js,
	}, nil
}

// Ensure NATSPublisher implements port.EventPublisher
var _ port.EventPublisher = (*NATSPublisher)(nil)

// Publish publishes a payment event to the given subject
func (p *NATSPublisher) Publish(ctx context.Context, subject string, event *domain.PaymentEvent) error {
	data, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("failed to marshal event: %w", err)
	}

	_, err = p.js.Publish(ctx, subject, data)
	if err != nil {
		return fmt.Errorf("failed to publish event: %w", err)
	}

	return nil
}

// Close closes the NATS connection
func (p *NATSPublisher) Close() error {
	p.nc.Close()
	return nil
}
