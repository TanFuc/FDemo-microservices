package worker

import (
	"context"
	"encoding/json"
	"log"
	"time"

	"github.com/nats-io/nats.go/jetstream"
	"microservices/catalog/internal/service"
)

// RatingUpdatedEvent received from Review Service
type RatingUpdatedEvent struct {
	EventID       string    `json:"eventId"`
	ProductID     string    `json:"productId"`
	AverageRating float64   `json:"averageRating"`
	TotalReviews  int       `json:"totalReviews"`
	OccurredAt    time.Time `json:"occurredAt"`
}

// RatingSyncWorker consumes rating events from NATS and updates product ratings
type RatingSyncWorker struct {
	js             jetstream.JetStream
	productService service.ProductService
	consumer       jetstream.Consumer
	stopCh         chan struct{}
}

// NewRatingSyncWorker creates a new rating sync worker
func NewRatingSyncWorker(js jetstream.JetStream, productService service.ProductService) *RatingSyncWorker {
	return &RatingSyncWorker{
		js:             js,
		productService: productService,
		stopCh:         make(chan struct{}),
	}
}

// Start registers the JetStream consumer for rating events
func (w *RatingSyncWorker) Start(ctx context.Context) error {
	// Create or get the RATINGS stream
	_, err := w.js.CreateOrUpdateStream(ctx, jetstream.StreamConfig{
		Name:      "RATINGS",
		Subjects:  []string{"rating.>"},
		Retention: jetstream.InterestPolicy,
		MaxAge:    24 * time.Hour,
		Storage:   jetstream.FileStorage,
		Replicas:  1,
	})
	if err != nil {
		log.Printf("WARN: failed to create/update RATINGS stream: %v", err)
	}

	// Create a durable consumer
	consumer, err := w.js.CreateOrUpdateConsumer(ctx, "RATINGS", jetstream.ConsumerConfig{
		Durable:       "catalog-rating-sync",
		FilterSubject: "rating.updated.>",
		AckPolicy:     jetstream.AckExplicitPolicy,
		MaxDeliver:    5,
		DeliverPolicy: jetstream.DeliverNewPolicy,
	})
	if err != nil {
		return err
	}
	w.consumer = consumer

	// Start consuming messages
	go w.consumeMessages(ctx)

	log.Println("RatingSyncWorker started: listening on rating.updated.>")
	return nil
}

// consumeMessages continuously fetches and processes messages
func (w *RatingSyncWorker) consumeMessages(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		case <-w.stopCh:
			return
		default:
			msgs, err := w.consumer.Fetch(10, jetstream.FetchMaxWait(5*time.Second))
			if err != nil {
				if err != context.DeadlineExceeded {
					log.Printf("ERROR: failed to fetch messages: %v", err)
				}
				continue
			}

			for msg := range msgs.Messages() {
				w.handleRatingUpdated(msg)
			}
		}
	}
}

// Stop stops the worker
func (w *RatingSyncWorker) Stop() {
	close(w.stopCh)
	log.Println("RatingSyncWorker stopped")
}

// handleRatingUpdated processes rating update events
func (w *RatingSyncWorker) handleRatingUpdated(msg jetstream.Msg) {
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	var event RatingUpdatedEvent
	if err := json.Unmarshal(msg.Data(), &event); err != nil {
		log.Printf("ERROR: failed to unmarshal RatingUpdatedEvent: %v", err)
		_ = msg.NakWithDelay(30 * time.Second)
		return
	}

	log.Printf("RatingSyncWorker: updating catalog rating for product %s avg=%.2f total=%d",
		event.ProductID, event.AverageRating, event.TotalReviews)

	// Update product metadata with new rating
	metadata := map[string]interface{}{
		"rating":          event.AverageRating,
		"reviewCount":     event.TotalReviews,
		"ratingUpdatedAt": event.OccurredAt,
	}

	_, err := w.productService.UpdateProductMetadata(ctx, event.ProductID, metadata)
	if err != nil {
		log.Printf("ERROR: failed to update product rating in catalog: %v", err)
		_ = msg.NakWithDelay(15 * time.Second)
		return
	}

	if err := msg.Ack(); err != nil {
		log.Printf("ERROR: failed to ACK rating.updated message: %v", err)
	}

	log.Printf("RatingSyncWorker: catalog rating updated for product %s", event.ProductID)
}
