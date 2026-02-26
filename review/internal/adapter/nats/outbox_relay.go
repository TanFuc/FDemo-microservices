package nats

import (
	"context"
	"log"
	"time"

	"github.com/nats-io/nats.go"
	"microservices/review/internal/adapter/mongodb"
)

// OutboxRelay periodically polls the outbox and publishes unsent events
type OutboxRelay struct {
	outboxRepo *mongodb.OutboxRepository
	publisher  *Publisher
	interval   time.Duration
	batchSize  int
}

// NewOutboxRelay creates a new outbox relay
func NewOutboxRelay(outboxRepo *mongodb.OutboxRepository, publisher *Publisher) *OutboxRelay {
	return &OutboxRelay{
		outboxRepo: outboxRepo,
		publisher:  publisher,
		interval:   10 * time.Second,
		batchSize:  50,
	}
}

// Run starts the outbox relay loop
func (r *OutboxRelay) Run(ctx context.Context) {
	ticker := time.NewTicker(r.interval)
	defer ticker.Stop()

	log.Println("OutboxRelay started: polling every", r.interval)

	// Process immediately on start
	r.processOutbox(ctx)

	for {
		select {
		case <-ctx.Done():
			log.Println("OutboxRelay stopped")
			return
		case <-ticker.C:
			r.processOutbox(ctx)
		}
	}
}

func (r *OutboxRelay) processOutbox(ctx context.Context) {
	if r.publisher == nil || !r.publisher.IsConnected() {
		return
	}

	entries, err := r.outboxRepo.GetPendingEntries(ctx, r.batchSize)
	if err != nil {
		log.Printf("OutboxRelay: error fetching pending entries: %v", err)
		return
	}

	if len(entries) == 0 {
		return
	}

	log.Printf("OutboxRelay: processing %d pending entries", len(entries))

	for _, entry := range entries {
		// Use MsgId for NATS deduplication
		_, err := r.publisher.js.Publish(
			entry.Subject,
			entry.Payload,
			nats.Context(ctx),
			nats.MsgId(entry.EventID),
		)

		if err != nil {
			log.Printf("OutboxRelay: failed to publish %s (attempt %d): %v",
				entry.EventID, entry.Attempts+1, err)
			_ = r.outboxRepo.MarkFailed(ctx, entry.ID, err.Error())
		} else {
			log.Printf("OutboxRelay: published %s to %s", entry.EventID, entry.Subject)
			_ = r.outboxRepo.MarkSent(ctx, entry.ID)
		}
	}
}

// SetInterval sets the polling interval
func (r *OutboxRelay) SetInterval(interval time.Duration) {
	r.interval = interval
}

// SetBatchSize sets the batch size for processing
func (r *OutboxRelay) SetBatchSize(size int) {
	r.batchSize = size
}
