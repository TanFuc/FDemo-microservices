package outbox

import (
	"context"
	"log/slog"
	"sync"
	"time"

	"microservices/wallet/internal/port"
)

// OutboxWorker polls the outbox_events table and publishes events to NATS
type OutboxWorker struct {
	repo      port.WalletRepository
	publisher port.EventPublisher
	logger    *slog.Logger
	interval  time.Duration
	batchSize int
	stopChan  chan struct{}
	wg        sync.WaitGroup
}

// Config holds outbox worker configuration
type Config struct {
	Interval  time.Duration
	BatchSize int
}

// DefaultConfig returns default outbox worker configuration
func DefaultConfig() Config {
	return Config{
		Interval:  500 * time.Millisecond,
		BatchSize: 100,
	}
}

// NewOutboxWorker creates a new outbox worker
func NewOutboxWorker(repo port.WalletRepository, publisher port.EventPublisher, logger *slog.Logger, cfg Config) *OutboxWorker {
	if cfg.Interval == 0 {
		cfg.Interval = 500 * time.Millisecond
	}
	if cfg.BatchSize == 0 {
		cfg.BatchSize = 100
	}

	return &OutboxWorker{
		repo:      repo,
		publisher: publisher,
		logger:    logger,
		interval:  cfg.Interval,
		batchSize: cfg.BatchSize,
		stopChan:  make(chan struct{}),
	}
}

// Start starts the outbox worker
func (w *OutboxWorker) Start(ctx context.Context) {
	w.wg.Add(1)
	go func() {
		defer w.wg.Done()
		w.run(ctx)
	}()
	w.logger.Info("outbox worker started", "interval", w.interval, "batch_size", w.batchSize)
}

// Stop stops the outbox worker
func (w *OutboxWorker) Stop() {
	close(w.stopChan)
	w.wg.Wait()
	w.logger.Info("outbox worker stopped")
}

func (w *OutboxWorker) run(ctx context.Context) {
	ticker := time.NewTicker(w.interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-w.stopChan:
			return
		case <-ticker.C:
			w.processBatch(ctx)
		}
	}
}

func (w *OutboxWorker) processBatch(ctx context.Context) {
	events, err := w.repo.GetPendingOutboxEvents(ctx, w.batchSize)
	if err != nil {
		w.logger.Error("failed to fetch pending outbox events", "error", err)
		return
	}

	for _, e := range events {
		if err := w.publisher.PublishRaw(ctx, e.EventType, e.Payload); err != nil {
			w.logger.Error("failed to publish outbox event",
				"event_id", e.ID,
				"event_type", e.EventType,
				"error", err,
			)
			if markErr := w.repo.MarkOutboxFailed(ctx, e.ID); markErr != nil {
				w.logger.Error("failed to mark outbox event as failed",
					"event_id", e.ID,
					"error", markErr,
				)
			}
			continue
		}

		if err := w.repo.MarkOutboxPublished(ctx, e.ID); err != nil {
			w.logger.Error("failed to mark outbox event as published",
				"event_id", e.ID,
				"error", err,
			)
		} else {
			w.logger.Debug("outbox event published",
				"event_id", e.ID,
				"event_type", e.EventType,
			)
		}
	}
}
