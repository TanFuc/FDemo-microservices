package worker

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"time"

	"github.com/google/uuid"
	"github.com/nats-io/nats.go"
	"microservices/notification/internal/adapter"
	"microservices/notification/internal/infrastructure"
	"microservices/notification/internal/models"
	"microservices/notification/internal/service"
)

// ReviewWorker consumes review events from NATS and triggers seller notifications
type ReviewWorker struct {
	nats          *infrastructure.NATS
	notifService  service.NotificationService
	catalogClient adapter.CatalogClient
	idempotency   *IdempotencyStore
	subscriptions []*nats.Subscription
}

// NewReviewWorker creates a new review event worker
func NewReviewWorker(
	nats *infrastructure.NATS,
	notifService service.NotificationService,
	catalogClient adapter.CatalogClient,
	idempotency *IdempotencyStore,
) *ReviewWorker {
	return &ReviewWorker{
		nats:          nats,
		notifService:  notifService,
		catalogClient: catalogClient,
		idempotency:   idempotency,
	}
}

// Start registers all JetStream consumers for review events
func (w *ReviewWorker) Start(ctx context.Context) error {
	// Consumer for review.created - notify seller
	sub1, err := w.nats.JetStreamQueueSubscribe(
		"review.created",
		"notification-review-created",
		w.handleReviewCreated,
		nats.Durable("notification-review-created"),
		nats.ManualAck(),
		nats.AckExplicit(),
		nats.MaxDeliver(5),
		nats.DeliverNew(),
	)
	if err != nil {
		return fmt.Errorf("subscribe review.created: %w", err)
	}
	w.subscriptions = append(w.subscriptions, sub1)

	// Consumer for review.updated - notify reviewer that seller replied
	sub2, err := w.nats.JetStreamQueueSubscribe(
		"review.updated",
		"notification-review-updated",
		w.handleReviewUpdated,
		nats.Durable("notification-review-updated"),
		nats.ManualAck(),
		nats.AckExplicit(),
		nats.MaxDeliver(5),
		nats.DeliverNew(),
	)
	if err != nil {
		return fmt.Errorf("subscribe review.updated: %w", err)
	}
	w.subscriptions = append(w.subscriptions, sub2)

	log.Println("ReviewWorker started: listening on review.created and review.updated")
	return nil
}

// Stop unsubscribes from all subscriptions
func (w *ReviewWorker) Stop() {
	for _, sub := range w.subscriptions {
		if sub != nil {
			_ = sub.Unsubscribe()
		}
	}
	log.Println("ReviewWorker stopped")
}

// handleReviewCreated processes a new review and notifies the product's seller
func (w *ReviewWorker) handleReviewCreated(msg *nats.Msg) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	var event models.ReviewCreatedEvent
	if err := json.Unmarshal(msg.Data, &event); err != nil {
		log.Printf("ERROR: failed to unmarshal ReviewCreatedEvent: %v", err)
		_ = msg.NakWithDelay(30 * time.Second)
		return
	}

	log.Printf("ReviewWorker: processing review.created reviewId=%s productId=%s", event.ReviewID, event.ProductID)

	// Idempotency check
	if w.idempotency != nil {
		processed, err := w.idempotency.IsProcessed(ctx, event.EventID)
		if err != nil {
			log.Printf("WARN: idempotency check failed: %v", err)
		}
		if processed {
			log.Printf("ReviewWorker: skipping duplicate event %s", event.EventID)
			_ = msg.Ack()
			return
		}
	}

	// Look up product seller from Catalog Service
	seller, err := w.catalogClient.GetProductSeller(ctx, event.ProductID)
	if err != nil {
		log.Printf("WARN: cannot find seller for product %s: %v - will retry", event.ProductID, err)
		_ = msg.NakWithDelay(10 * time.Second)
		return
	}

	// Build and send notification to seller
	stars := buildStarEmoji(event.Rating)
	truncContent := event.Content
	if len(truncContent) > 100 {
		truncContent = truncContent[:100] + "..."
	}

	notification := &models.Notification{
		ID:       uuid.New().String(),
		UserID:   seller.SellerID,
		Type:     "REVIEW_RECEIVED",
		Title:    fmt.Sprintf("%s - You have a new review!", seller.ShopName),
		Body:     fmt.Sprintf("%s (%s): \"%s\"", event.UserName, stars, truncContent),
		Channel:  "all",
		Priority: "normal",
		Data: map[string]interface{}{
			"reviewId":    event.ReviewID,
			"productId":   event.ProductID,
			"productName": seller.ProductName,
			"rating":      event.Rating,
			"userId":      event.UserID,
			"userName":    event.UserName,
			"shopId":      seller.ShopID,
			"action":      "open_review",
		},
	}

	if err := w.notifService.SendToUser(ctx, seller.SellerID, notification); err != nil {
		log.Printf("ERROR: failed to send review notification to seller %s: %v", seller.SellerID, err)
		_ = msg.NakWithDelay(15 * time.Second)
		return
	}

	// Mark as processed
	if w.idempotency != nil {
		_ = w.idempotency.MarkProcessed(ctx, event.EventID)
	}

	// ACK message
	if err := msg.Ack(); err != nil {
		log.Printf("ERROR: failed to ACK review.created message: %v", err)
	}

	log.Printf("ReviewWorker: review.created processed successfully reviewId=%s sellerID=%s", event.ReviewID, seller.SellerID)
}

// handleReviewUpdated notifies the original reviewer that the seller replied
func (w *ReviewWorker) handleReviewUpdated(msg *nats.Msg) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	var event models.ReviewCreatedEvent
	if err := json.Unmarshal(msg.Data, &event); err != nil {
		_ = msg.NakWithDelay(30 * time.Second)
		return
	}

	log.Printf("ReviewWorker: processing review.updated reviewId=%s userId=%s", event.ReviewID, event.UserID)

	// Idempotency check
	if w.idempotency != nil {
		processed, err := w.idempotency.IsProcessed(ctx, event.EventID)
		if err != nil {
			log.Printf("WARN: idempotency check failed: %v", err)
		}
		if processed {
			log.Printf("ReviewWorker: skipping duplicate event %s", event.EventID)
			_ = msg.Ack()
			return
		}
	}

	notification := &models.Notification{
		ID:       uuid.New().String(),
		UserID:   event.UserID,
		Type:     "REVIEW_REPLIED",
		Title:    "The seller has replied to your review",
		Body:     "Your product review has received a response from the seller.",
		Channel:  "all",
		Priority: "normal",
		Data: map[string]interface{}{
			"reviewId":  event.ReviewID,
			"productId": event.ProductID,
			"action":    "open_review",
		},
	}

	if err := w.notifService.SendToUser(ctx, event.UserID, notification); err != nil {
		_ = msg.NakWithDelay(15 * time.Second)
		return
	}

	// Mark as processed
	if w.idempotency != nil {
		_ = w.idempotency.MarkProcessed(ctx, event.EventID)
	}

	_ = msg.Ack()
	log.Printf("ReviewWorker: review.updated processed successfully reviewId=%s userId=%s", event.ReviewID, event.UserID)
}

func buildStarEmoji(rating int) string {
	stars := ""
	for i := 0; i < rating; i++ {
		stars += "*"
	}
	return stars
}
