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
	StreamName           = "USERS"
	SubjectPrefix        = "user."
	SubjectCreated       = "user.registered"
	SubjectIdentityCreated      = "identity.user.created"
	SubjectIdentityRoleChanged  = "identity.user.role_changed"
	SubjectIdentityStatusChanged = "identity.user.status_changed"
	SubjectIdentityDeleted      = "identity.user.deleted"
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

// UserCreatedEvent for federated user context propagation
type UserCreatedEvent struct {
	UserID      string `json:"userId"`
	Email       string `json:"email"`
	DisplayName string `json:"displayName"`
	Role        string `json:"role"`
	Timestamp   string `json:"timestamp"`
}

// UserRoleChangedEvent is published when user role changes
type UserRoleChangedEvent struct {
	UserID    string `json:"userId"`
	OldRole   string `json:"oldRole"`
	NewRole   string `json:"newRole"`
	Timestamp string `json:"timestamp"`
}

// UserStatusChangedEvent is published when user status changes
type UserStatusChangedEvent struct {
	UserID    string `json:"userId"`
	OldStatus string `json:"oldStatus"`
	NewStatus string `json:"newStatus"`
	Reason    string `json:"reason,omitempty"`
	Timestamp string `json:"timestamp"`
}

// UserDeletedEvent is published when user is soft-deleted
type UserDeletedEvent struct {
	UserID    string `json:"userId"`
	Reason    string `json:"reason,omitempty"`
	Timestamp string `json:"timestamp"`
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
			Subjects:    []string{SubjectPrefix + "*", "identity.user.*"},
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

// PublishUserCreated publishes identity.user.created event for federated user context
func (n *NATSClient) PublishUserCreated(ctx context.Context, event *UserCreatedEvent) error {
	data, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("failed to marshal event: %w", err)
	}

	_, err = n.js.Publish(ctx, SubjectIdentityCreated, data)
	if err != nil {
		return fmt.Errorf("failed to publish event: %w", err)
	}

	logger.Info().
		Str("userId", event.UserID).
		Str("role", event.Role).
		Msg("Published identity.user.created event")

	return nil
}

// PublishUserRoleChanged publishes identity.user.role_changed event
func (n *NATSClient) PublishUserRoleChanged(ctx context.Context, event *UserRoleChangedEvent) error {
	data, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("failed to marshal event: %w", err)
	}

	_, err = n.js.Publish(ctx, SubjectIdentityRoleChanged, data)
	if err != nil {
		return fmt.Errorf("failed to publish event: %w", err)
	}

	logger.Info().
		Str("userId", event.UserID).
		Str("newRole", event.NewRole).
		Msg("Published identity.user.role_changed event")

	return nil
}

// PublishUserStatusChanged publishes identity.user.status_changed event
func (n *NATSClient) PublishUserStatusChanged(ctx context.Context, event *UserStatusChangedEvent) error {
	data, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("failed to marshal event: %w", err)
	}

	_, err = n.js.Publish(ctx, SubjectIdentityStatusChanged, data)
	if err != nil {
		return fmt.Errorf("failed to publish event: %w", err)
	}

	logger.Info().
		Str("userId", event.UserID).
		Str("newStatus", event.NewStatus).
		Msg("Published identity.user.status_changed event")

	return nil
}

// PublishUserDeleted publishes identity.user.deleted event
func (n *NATSClient) PublishUserDeleted(ctx context.Context, event *UserDeletedEvent) error {
	data, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("failed to marshal event: %w", err)
	}

	_, err = n.js.Publish(ctx, SubjectIdentityDeleted, data)
	if err != nil {
		return fmt.Errorf("failed to publish event: %w", err)
	}

	logger.Info().
		Str("userId", event.UserID).
		Msg("Published identity.user.deleted event")

	return nil
}
