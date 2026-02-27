package messaging

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/nats-io/nats.go"

	"microservices/wallet/internal/domain"
	"microservices/wallet/internal/port"
)

// NATSPublisher implements port.EventPublisher using NATS JetStream
type NATSPublisher struct {
	nc *nats.Conn
	js nats.JetStreamContext
}

// NewNATSPublisher creates a new NATS publisher
func NewNATSPublisher(url string) (*NATSPublisher, error) {
	nc, err := nats.Connect(url,
		nats.RetryOnFailedConnect(true),
		nats.MaxReconnects(10),
		nats.ReconnectWait(time.Second),
		nats.DisconnectErrHandler(func(nc *nats.Conn, err error) {
			if err != nil {
				fmt.Printf("NATS disconnected: %v\n", err)
			}
		}),
		nats.ReconnectHandler(func(nc *nats.Conn) {
			fmt.Println("NATS reconnected")
		}),
	)
	if err != nil {
		return nil, fmt.Errorf("connect to NATS: %w", err)
	}

	js, err := nc.JetStream()
	if err != nil {
		nc.Close()
		return nil, fmt.Errorf("create JetStream context: %w", err)
	}

	// Create WALLET_EVENTS stream (idempotent - ok if already exists)
	_, err = js.AddStream(&nats.StreamConfig{
		Name:      domain.WalletStreamName,
		Subjects:  []string{"wallet.>"},
		Storage:   nats.FileStorage,
		Retention: nats.WorkQueuePolicy,
		MaxAge:    7 * 24 * time.Hour,
	})
	if err != nil && err != nats.ErrStreamNameAlreadyInUse {
		// Stream might already exist, which is fine
		// Check if it's a different error
		if !isStreamExistsError(err) {
			nc.Close()
			return nil, fmt.Errorf("create stream: %w", err)
		}
	}

	return &NATSPublisher{nc: nc, js: js}, nil
}

// Publish publishes a wallet event to the specified subject
func (p *NATSPublisher) Publish(ctx context.Context, subject string, event *domain.WalletEvent) error {
	data, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("marshal event: %w", err)
	}
	_, err = p.js.Publish(subject, data)
	if err != nil {
		return fmt.Errorf("publish event: %w", err)
	}
	return nil
}

// PublishRaw publishes raw bytes to the specified subject
func (p *NATSPublisher) PublishRaw(ctx context.Context, subject string, payload []byte) error {
	_, err := p.js.Publish(subject, payload)
	if err != nil {
		return fmt.Errorf("publish raw: %w", err)
	}
	return nil
}

// Close closes the NATS connection
func (p *NATSPublisher) Close() error {
	if p.nc != nil {
		p.nc.Drain()
	}
	return nil
}

// Ensure NATSPublisher implements port.EventPublisher
var _ port.EventPublisher = (*NATSPublisher)(nil)

func isStreamExistsError(err error) bool {
	return err == nats.ErrStreamNameAlreadyInUse ||
		(err != nil && (err.Error() == "stream name already in use" ||
			err.Error() == "nats: stream name already in use"))
}
