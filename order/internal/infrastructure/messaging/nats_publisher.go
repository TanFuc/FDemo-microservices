package messaging

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/nats-io/nats.go"
	"microservices/order/internal/config"
	"microservices/order/internal/domain"
)

// NATSPublisher handles event publishing to NATS JetStream
type NATSPublisher struct {
	conn       *nats.Conn
	js         nats.JetStreamContext
	streamName string
}

// NewNATSPublisher creates a new NATS publisher
func NewNATSPublisher(cfg *config.NATSConfig) (*NATSPublisher, error) {
	conn, err := nats.Connect(cfg.URL)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to NATS: %w", err)
	}

	js, err := conn.JetStream()
	if err != nil {
		conn.Close()
		return nil, fmt.Errorf("failed to get JetStream context: %w", err)
	}

	// Create stream if it doesn't exist
	streamCfg := &nats.StreamConfig{
		Name:     cfg.StreamName,
		Subjects: []string{"order.*"},
		Storage:  nats.FileStorage,
	}

	_, err = js.StreamInfo(cfg.StreamName)
	if err != nil {
		_, err = js.AddStream(streamCfg)
		if err != nil {
			conn.Close()
			return nil, fmt.Errorf("failed to create stream: %w", err)
		}
	}

	return &NATSPublisher{
		conn:       conn,
		js:         js,
		streamName: cfg.StreamName,
	}, nil
}

// Close closes the NATS connection
func (p *NATSPublisher) Close() error {
	if p.conn != nil {
		p.conn.Close()
	}
	return nil
}

// Publish publishes an event to the specified subject
func (p *NATSPublisher) Publish(ctx context.Context, subject string, event interface{}) error {
	data, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("failed to marshal event: %w", err)
	}

	_, err = p.js.Publish(subject, data)
	if err != nil {
		return fmt.Errorf("%w: %v", domain.ErrEventPublishFailed, err)
	}

	return nil
}

// PublishOrderCreated publishes an order created event
func (p *NATSPublisher) PublishOrderCreated(ctx context.Context, event *OrderCreatedEvent) error {
	return p.Publish(ctx, EventOrderCreated, event)
}

// PublishOrderCancelled publishes an order cancelled event
func (p *NATSPublisher) PublishOrderCancelled(ctx context.Context, event *OrderCancelledEvent) error {
	return p.Publish(ctx, EventOrderCancelled, event)
}

// PublishOrderPaid publishes an order paid event
func (p *NATSPublisher) PublishOrderPaid(ctx context.Context, event *OrderPaidEvent) error {
	return p.Publish(ctx, EventOrderPaid, event)
}

// PublishOrderShipped publishes an order shipped event
func (p *NATSPublisher) PublishOrderShipped(ctx context.Context, event *OrderShippedEvent) error {
	return p.Publish(ctx, EventOrderShipped, event)
}

// PublishOrderCompleted publishes an order completed event
func (p *NATSPublisher) PublishOrderCompleted(ctx context.Context, event *OrderCompletedEvent) error {
	return p.Publish(ctx, EventOrderCompleted, event)
}

// EventPublisher interface for dependency injection
type EventPublisher interface {
	PublishOrderCreated(ctx context.Context, event *OrderCreatedEvent) error
	PublishOrderCancelled(ctx context.Context, event *OrderCancelledEvent) error
	PublishOrderPaid(ctx context.Context, event *OrderPaidEvent) error
	PublishOrderShipped(ctx context.Context, event *OrderShippedEvent) error
	PublishOrderCompleted(ctx context.Context, event *OrderCompletedEvent) error
}

// Ensure NATSPublisher implements EventPublisher
var _ EventPublisher = (*NATSPublisher)(nil)
