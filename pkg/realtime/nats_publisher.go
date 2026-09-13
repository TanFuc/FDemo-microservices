package realtime

import (
	"context"
	"fmt"

	"github.com/nats-io/nats.go"
)

// NatsPublisher implements Publisher using NATS as the distributed message bus backplane.
type NatsPublisher struct {
	conn *nats.Conn
}

// NewNatsPublisher connects to NATS and initializes the publisher.
func NewNatsPublisher(natsURL string) (*NatsPublisher, error) {
	nc, err := nats.Connect(natsURL)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to nats at %s: %w", natsURL, err)
	}

	return &NatsPublisher{
		conn: nc,
	}, nil
}

// NewNatsPublisherWithConn wraps an existing NATS connection.
func NewNatsPublisherWithConn(nc *nats.Conn) *NatsPublisher {
	return &NatsPublisher{
		conn: nc,
	}
}

// Publish publishes the event to the appropriate NATS subject based on target scope.
func (p *NatsPublisher) Publish(ctx context.Context, event *EventEnvelope) error {
	if event == nil {
		return ErrNilEvent
	}

	data, err := SerializeEvent(event)
	if err != nil {
		return fmt.Errorf("failed to serialize event: %w", err)
	}

	subject := NatsSubjectForEvent(event)
	return p.conn.Publish(subject, data)
}

// Close closes the underlying NATS connection.
func (p *NatsPublisher) Close() error {
	if p.conn != nil {
		p.conn.Close()
	}
	return nil
}
