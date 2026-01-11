package queue

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/nats-io/nats.go"
	"github.com/nats-io/nats.go/jetstream"

	"microservices/auth/internal/config"
	"microservices/auth/pkg/logger"
)

const (
	StreamName     = "USERS"
	SubjectPrefix  = "user."
	SubjectCreated = "user.registered"
)

type NATSClient struct {
	conn   *nats.Conn
	js     jetstream.JetStream
	stream jetstream.Stream
}

type UserRegisteredEvent struct {
	UserID       string `json:"userId"`
	Email        string `json:"email"`
	FullName     string `json:"fullName"`
	RegisteredAt string `json:"registeredAt"`
}

func NewNATSClient(cfg *config.Config) (*NATSClient, error) {
	conn, err := nats.Connect(cfg.NATS.URL,
		nats.RetryOnFailedConnect(true),
		nats.MaxReconnects(10),
		nats.ReconnectWait(time.Second),
		nats.DisconnectErrHandler(func(nc *nats.Conn, err error) {
			if err != nil {
				logger.Warn().Err(err).Msg("NATS disconnected")
			}
		}),
		nats.ReconnectHandler(func(nc *nats.Conn) {
			logger.Info().Msg("NATS reconnected")
		}),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to NATS: %w", err)
	}

	js, err := jetstream.New(conn)
	if err != nil {
		conn.Close()
		return nil, fmt.Errorf("failed to create JetStream context: %w", err)
	}

	client := &NATSClient{
		conn: conn,
		js:   js,
	}

	// Initialize stream
	if err := client.initStream(); err != nil {
		conn.Close()
		return nil, err
	}

	logger.Info().Msg("Connected to NATS JetStream")

	return client, nil
}

func (n *NATSClient) initStream() error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Try to get existing stream
	stream, err := n.js.Stream(ctx, StreamName)
	if err != nil {
		// Stream doesn't exist, create it
		stream, err = n.js.CreateStream(ctx, jetstream.StreamConfig{
			Name:        StreamName,
			Description: "User events stream",
			Subjects:    []string{SubjectPrefix + "*"},
			Retention:   jetstream.LimitsPolicy,
			MaxAge:      7 * 24 * time.Hour, // 7 days retention
			MaxMsgs:     -1,
			MaxBytes:    -1,
			Discard:     jetstream.DiscardOld,
			Storage:     jetstream.FileStorage,
			Replicas:    1,
		})
		if err != nil {
			return fmt.Errorf("failed to create stream: %w", err)
		}
		logger.Info().Msg("Created NATS stream: " + StreamName)
	}

	n.stream = stream
	return nil
}

func (n *NATSClient) PublishUserRegistered(ctx context.Context, event *UserRegisteredEvent) error {
	data, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("failed to marshal event: %w", err)
	}

	_, err = n.js.Publish(ctx, SubjectCreated, data)
	if err != nil {
		return fmt.Errorf("failed to publish event: %w", err)
	}

	logger.Info().
		Str("userId", event.UserID).
		Str("email", event.Email).
		Msg("Published user.registered event")

	return nil
}

func (n *NATSClient) Close() error {
	if n.conn != nil {
		n.conn.Drain()
		n.conn.Close()
	}
	return nil
}

func (n *NATSClient) HealthCheck() error {
	if n.conn == nil {
		return fmt.Errorf("NATS connection is nil")
	}
	if !n.conn.IsConnected() {
		return fmt.Errorf("NATS is not connected")
	}
	return nil
}

func (n *NATSClient) JetStream() jetstream.JetStream {
	return n.js
}

func (n *NATSClient) Stream() jetstream.Stream {
	return n.stream
}
