package event

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/nats-io/nats.go"
	"github.com/nats-io/nats.go/jetstream"

	"microservices/profile/internal/config"
	"microservices/profile/pkg/logger"
)

const (
	ProfileStreamName = "PROFILE"

	// Profile event subjects
	SubjectShopRegistered = "profile.shop.registered"
	SubjectShopApproved   = "profile.shop.approved"
	SubjectShopUpdated    = "profile.shop.updated"
	SubjectUserUpdated    = "profile.user.updated"
)

// ShopRegisteredEvent is published when a user registers a shop
type ShopRegisteredEvent struct {
	UserID       string `json:"userId"`
	ShopID       string `json:"shopId"`
	ShopName     string `json:"shopName"`
	BusinessType string `json:"businessType"`
	Status       string `json:"status"`
	Timestamp    string `json:"timestamp"`
}

// ShopApprovedEvent is published when admin approves a shop
type ShopApprovedEvent struct {
	UserID    string `json:"userId"`
	ShopID    string `json:"shopId"`
	Status    string `json:"status"`
	Timestamp string `json:"timestamp"`
}

// ShopUpdatedEvent is published when shop details are updated
type ShopUpdatedEvent struct {
	UserID    string `json:"userId"`
	ShopID    string `json:"shopId"`
	ShopName  string `json:"shopName,omitempty"`
	Status    string `json:"status,omitempty"`
	Timestamp string `json:"timestamp"`
}

// UserProfileUpdatedEvent is published when user profile is updated
type UserProfileUpdatedEvent struct {
	UserID      string `json:"userId"`
	DisplayName string `json:"displayName,omitempty"`
	AvatarURL   string `json:"avatarUrl,omitempty"`
	Email       string `json:"email,omitempty"`
	Version     int64  `json:"version"`
	Timestamp   string `json:"timestamp"`
}

// Publisher handles NATS event publishing for Profile service
type Publisher struct {
	conn   *nats.Conn
	js     jetstream.JetStream
	stream jetstream.Stream
}

// NewPublisher creates a new NATS publisher
func NewPublisher(cfg *config.Config) (*Publisher, error) {
	conn, err := nats.Connect(cfg.NATS.URL,
		nats.RetryOnFailedConnect(true),
		nats.MaxReconnects(10),
		nats.ReconnectWait(time.Second),
		nats.DisconnectErrHandler(func(nc *nats.Conn, err error) {
			if err != nil {
				logger.Warn().Err(err).Msg("NATS publisher disconnected")
			}
		}),
		nats.ReconnectHandler(func(nc *nats.Conn) {
			logger.Info().Msg("NATS publisher reconnected")
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

	pub := &Publisher{
		conn: conn,
		js:   js,
	}

	// Initialize stream
	if err := pub.initStream(); err != nil {
		conn.Close()
		return nil, err
	}

	logger.Info().Msg("Profile NATS publisher initialized")

	return pub, nil
}

func (p *Publisher) initStream() error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Try to get existing stream
	stream, err := p.js.Stream(ctx, ProfileStreamName)
	if err != nil {
		// Stream doesn't exist, create it
		stream, err = p.js.CreateStream(ctx, jetstream.StreamConfig{
			Name:        ProfileStreamName,
			Description: "Profile service events stream",
			Subjects:    []string{"profile.*"},
			Retention:   jetstream.LimitsPolicy,
			MaxAge:      7 * 24 * time.Hour,
			MaxMsgs:     -1,
			MaxBytes:    -1,
			Discard:     jetstream.DiscardOld,
			Storage:     jetstream.FileStorage,
			Replicas:    1,
		})
		if err != nil {
			return fmt.Errorf("failed to create stream: %w", err)
		}
		logger.Info().Msg("Created NATS stream: " + ProfileStreamName)
	}

	p.stream = stream
	return nil
}

// PublishShopRegistered publishes profile.shop.registered event
func (p *Publisher) PublishShopRegistered(ctx context.Context, event *ShopRegisteredEvent) error {
	event.Timestamp = time.Now().Format(time.RFC3339)

	data, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("failed to marshal event: %w", err)
	}

	_, err = p.js.Publish(ctx, SubjectShopRegistered, data)
	if err != nil {
		return fmt.Errorf("failed to publish event: %w", err)
	}

	logger.Info().
		Str("userId", event.UserID).
		Str("shopId", event.ShopID).
		Msg("Published profile.shop.registered event")

	return nil
}

// PublishShopApproved publishes profile.shop.approved event
func (p *Publisher) PublishShopApproved(ctx context.Context, event *ShopApprovedEvent) error {
	event.Timestamp = time.Now().Format(time.RFC3339)

	data, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("failed to marshal event: %w", err)
	}

	_, err = p.js.Publish(ctx, SubjectShopApproved, data)
	if err != nil {
		return fmt.Errorf("failed to publish event: %w", err)
	}

	logger.Info().
		Str("userId", event.UserID).
		Str("shopId", event.ShopID).
		Msg("Published profile.shop.approved event")

	return nil
}

// PublishShopUpdated publishes profile.shop.updated event
func (p *Publisher) PublishShopUpdated(ctx context.Context, event *ShopUpdatedEvent) error {
	event.Timestamp = time.Now().Format(time.RFC3339)

	data, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("failed to marshal event: %w", err)
	}

	_, err = p.js.Publish(ctx, SubjectShopUpdated, data)
	if err != nil {
		return fmt.Errorf("failed to publish event: %w", err)
	}

	logger.Info().
		Str("userId", event.UserID).
		Str("shopId", event.ShopID).
		Msg("Published profile.shop.updated event")

	return nil
}

// PublishUserUpdated publishes profile.user.updated event
func (p *Publisher) PublishUserUpdated(ctx context.Context, event *UserProfileUpdatedEvent) error {
	event.Timestamp = time.Now().Format(time.RFC3339)

	data, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("failed to marshal event: %w", err)
	}

	_, err = p.js.Publish(ctx, SubjectUserUpdated, data)
	if err != nil {
		return fmt.Errorf("failed to publish event: %w", err)
	}

	logger.Info().
		Str("userId", event.UserID).
		Msg("Published profile.user.updated event")

	return nil
}

// Close closes the NATS connection
func (p *Publisher) Close() error {
	if p.conn != nil {
		p.conn.Drain()
		p.conn.Close()
	}
	return nil
}

// HealthCheck checks if the NATS connection is healthy
func (p *Publisher) HealthCheck() error {
	if p.conn == nil {
		return fmt.Errorf("NATS connection is nil")
	}
	if !p.conn.IsConnected() {
		return fmt.Errorf("NATS is not connected")
	}
	return nil
}
