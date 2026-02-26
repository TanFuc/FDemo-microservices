package nats

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/nats-io/nats.go"
)

// Event subjects
const (
	SubjectReviewCreated = "review.created"
	SubjectReviewUpdated = "review.updated"
	SubjectReviewDeleted = "review.deleted"
	SubjectRatingUpdated = "rating.updated"
)

// ReviewCreatedEvent is published after a review is successfully saved
type ReviewCreatedEvent struct {
	EventID    string    `json:"eventId"`
	ReviewID   string    `json:"reviewId"`
	ProductID  string    `json:"productId"`
	OrderID    string    `json:"orderId"`
	UserID     string    `json:"userId"`
	UserName   string    `json:"userName"`
	Rating     int       `json:"rating"`
	Content    string    `json:"content"`
	Images     []string  `json:"images"`
	IsEdited   bool      `json:"isEdited"`
	EventType  string    `json:"eventType"`
	OccurredAt time.Time `json:"occurredAt"`
}

// RatingUpdatedEvent is published after ProductRating is recalculated
type RatingUpdatedEvent struct {
	EventID       string    `json:"eventId"`
	ProductID     string    `json:"productId"`
	AverageRating float64   `json:"averageRating"`
	TotalReviews  int       `json:"totalReviews"`
	OccurredAt    time.Time `json:"occurredAt"`
}

// Publisher wraps NATS JetStream for reliable event publishing
type Publisher struct {
	conn *nats.Conn
	js   nats.JetStreamContext
}

// NewPublisher creates a new NATS JetStream publisher
func NewPublisher(url string) (*Publisher, error) {
	opts := []nats.Option{
		nats.ReconnectWait(2 * time.Second),
		nats.MaxReconnects(10),
		nats.DisconnectErrHandler(func(nc *nats.Conn, err error) {
			if err != nil {
				fmt.Printf("NATS disconnected: %v\n", err)
			}
		}),
		nats.ReconnectHandler(func(nc *nats.Conn) {
			fmt.Printf("NATS reconnected to %s\n", nc.ConnectedUrl())
		}),
	}

	conn, err := nats.Connect(url, opts...)
	if err != nil {
		return nil, fmt.Errorf("nats connect: %w", err)
	}

	js, err := conn.JetStream(nats.PublishAsyncMaxPending(256))
	if err != nil {
		conn.Close()
		return nil, fmt.Errorf("jetstream context: %w", err)
	}

	p := &Publisher{conn: conn, js: js}
	if err := p.ensureStreams(); err != nil {
		fmt.Printf("WARN: failed to ensure streams: %v\n", err)
	}
	return p, nil
}

// ensureStreams creates REVIEWS and RATINGS streams if absent
func (p *Publisher) ensureStreams() error {
	streams := []nats.StreamConfig{
		{
			Name:        "REVIEWS",
			Description: "Review lifecycle events",
			Subjects:    []string{"review.>"},
			Retention:   nats.InterestPolicy,
			MaxAge:      72 * time.Hour,
			Storage:     nats.FileStorage,
			Replicas:    1,
			Discard:     nats.DiscardOld,
		},
		{
			Name:        "RATINGS",
			Description: "Product rating update events",
			Subjects:    []string{"rating.>"},
			Retention:   nats.InterestPolicy,
			MaxAge:      24 * time.Hour,
			Storage:     nats.FileStorage,
			Replicas:    1,
			Discard:     nats.DiscardOld,
		},
	}

	for _, cfg := range streams {
		if _, err := p.js.AddStream(&cfg); err != nil {
			if _, err2 := p.js.UpdateStream(&cfg); err2 != nil {
				return fmt.Errorf("stream %s: %w", cfg.Name, err2)
			}
		}
	}
	return nil
}

// PublishReviewCreated publishes a review.created event with at-least-once delivery
func (p *Publisher) PublishReviewCreated(ctx context.Context, event *ReviewCreatedEvent) error {
	event.EventType = "review.created"
	return p.publish(ctx, SubjectReviewCreated, event.EventID, event)
}

// PublishReviewUpdated publishes a review.updated event (e.g., seller reply)
func (p *Publisher) PublishReviewUpdated(ctx context.Context, event *ReviewCreatedEvent) error {
	event.EventType = "review.updated"
	return p.publish(ctx, SubjectReviewUpdated, event.EventID, event)
}

// PublishRatingUpdated publishes a rating.updated event for catalog sync
func (p *Publisher) PublishRatingUpdated(ctx context.Context, event *RatingUpdatedEvent) error {
	subject := fmt.Sprintf("%s.%s", SubjectRatingUpdated, event.ProductID)
	return p.publish(ctx, subject, event.EventID, event)
}

func (p *Publisher) publish(ctx context.Context, subject, msgID string, payload interface{}) error {
	data, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("marshal: %w", err)
	}

	// Synchronous publish with ACK confirmation (reliable)
	// MsgId is used for server-side deduplication
	_, err = p.js.Publish(subject, data, nats.Context(ctx), nats.MsgId(msgID))
	return err
}

// Close gracefully closes the NATS connection
func (p *Publisher) Close() {
	if p.conn != nil {
		p.conn.Drain()
		p.conn.Close()
	}
}

// IsConnected returns true if NATS is connected
func (p *Publisher) IsConnected() bool {
	return p.conn != nil && p.conn.IsConnected()
}
