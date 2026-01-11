package nats

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/nats-io/nats.go"
	"github.com/nats-io/nats.go/jetstream"

	"tafu-logistic/logistics-service/internal/core/ports"
)

// Publisher implements ports.EventPublisher using NATS JetStream
type Publisher struct {
	nc     *nats.Conn
	js     jetstream.JetStream
	stream jetstream.Stream
}

// Config holds NATS configuration
type Config struct {
	URL        string
	StreamName string
}

// NewPublisher creates a new NATS JetStream publisher
func NewPublisher(cfg Config) (*Publisher, error) {
	// Connect to NATS
	nc, err := nats.Connect(cfg.URL,
		nats.RetryOnFailedConnect(true),
		nats.MaxReconnects(10),
		nats.ReconnectWait(2*time.Second),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to NATS: %w", err)
	}

	// Create JetStream context
	js, err := jetstream.New(nc)
	if err != nil {
		nc.Close()
		return nil, fmt.Errorf("failed to create JetStream context: %w", err)
	}

	streamName := cfg.StreamName
	if streamName == "" {
		streamName = "LOGISTICS"
	}

	// Create or get stream
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	stream, err := js.CreateOrUpdateStream(ctx, jetstream.StreamConfig{
		Name:        streamName,
		Description: "Logistics service events",
		Subjects:    []string{"logistics.>"},
		Retention:   jetstream.LimitsPolicy,
		MaxAge:      24 * time.Hour * 7, // 7 days
		Storage:     jetstream.FileStorage,
		Replicas:    1,
	})
	if err != nil {
		nc.Close()
		return nil, fmt.Errorf("failed to create/update stream: %w", err)
	}

	return &Publisher{
		nc:     nc,
		js:     js,
		stream: stream,
	}, nil
}

// Ensure interface implementation
var _ ports.EventPublisher = (*Publisher)(nil)

// PublishShipmentCreated publishes a shipment created event
func (p *Publisher) PublishShipmentCreated(ctx context.Context, event *ports.ShipmentCreatedEvent) error {
	return p.publish(ctx, ports.SubjectShipmentCreated, event)
}

// PublishStatusUpdated publishes a status updated event
func (p *Publisher) PublishStatusUpdated(ctx context.Context, event *ports.StatusUpdatedEvent) error {
	return p.publish(ctx, ports.SubjectStatusUpdated, event)
}

// publish is a helper to publish an event to a subject
func (p *Publisher) publish(ctx context.Context, subject string, event interface{}) error {
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
func (p *Publisher) Close() error {
	p.nc.Close()
	return nil
}
