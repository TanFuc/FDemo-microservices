package outbox

import (
	"context"
	"encoding/json"
	"log/slog"
	"sync"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// OutboxEvent represents an event stored in the outbox table
type OutboxEvent struct {
	ID          uuid.UUID       `gorm:"type:uuid;primary_key"`
	AggregateID string          `gorm:"type:varchar(255);index"`
	EventType   string          `gorm:"type:varchar(100);index"`
	Payload     json.RawMessage `gorm:"type:jsonb"`
	Status      string          `gorm:"type:varchar(20);index;default:'PENDING'"` // PENDING, PUBLISHED, FAILED
	RetryCount  int             `gorm:"default:0"`
	CreatedAt   time.Time
	UpdatedAt   time.Time
	PublishedAt *time.Time
}

// TableName specifies the table name for OutboxEvent
func (OutboxEvent) TableName() string {
	return "outbox_events"
}

// OutboxRepository handles outbox event persistence
type OutboxRepository struct {
	db *gorm.DB
}

// NewOutboxRepository creates a new outbox repository
func NewOutboxRepository(db *gorm.DB) *OutboxRepository {
	return &OutboxRepository{db: db}
}

// Create creates a new outbox event within a transaction
func (r *OutboxRepository) Create(tx *gorm.DB, event *OutboxEvent) error {
	return tx.Create(event).Error
}

// GetPendingEvents retrieves pending events for publishing
func (r *OutboxRepository) GetPendingEvents(ctx context.Context, limit int) ([]*OutboxEvent, error) {
	var events []*OutboxEvent
	err := r.db.WithContext(ctx).
		Where("status = ?", "PENDING").
		Order("created_at ASC").
		Limit(limit).
		Find(&events).Error
	return events, err
}

// MarkAsPublished marks an event as successfully published
func (r *OutboxRepository) MarkAsPublished(ctx context.Context, eventID uuid.UUID) error {
	now := time.Now()
	return r.db.WithContext(ctx).
		Model(&OutboxEvent{}).
		Where("id = ?", eventID).
		Updates(map[string]interface{}{
			"status":       "PUBLISHED",
			"published_at": &now,
			"updated_at":   now,
		}).Error
}

// MarkAsFailed marks an event as failed and increments retry count
func (r *OutboxRepository) MarkAsFailed(ctx context.Context, eventID uuid.UUID) error {
	return r.db.WithContext(ctx).
		Model(&OutboxEvent{}).
		Where("id = ?", eventID).
		Updates(map[string]interface{}{
			"status":      "PENDING",
			"retry_count": gorm.Expr("retry_count + 1"),
			"updated_at":  time.Now(),
		}).Error
}

// DeleteOldPublishedEvents removes published events older than the given duration
func (r *OutboxRepository) DeleteOldPublishedEvents(ctx context.Context, olderThan time.Duration) (int64, error) {
	cutoff := time.Now().Add(-olderThan)
	result := r.db.WithContext(ctx).
		Where("status = ? AND published_at < ?", "PUBLISHED", cutoff).
		Delete(&OutboxEvent{})
	return result.RowsAffected, result.Error
}

// EventPublisher interface for publishing events
type EventPublisher interface {
	Publish(ctx context.Context, eventType string, payload []byte) error
}

// OutboxWorker processes outbox events and publishes them
type OutboxWorker struct {
	repo      *OutboxRepository
	publisher EventPublisher
	logger    *slog.Logger
	batchSize int
	interval  time.Duration
	stopChan  chan struct{}
	wg        sync.WaitGroup
}

// OutboxWorkerConfig holds configuration for the outbox worker
type OutboxWorkerConfig struct {
	BatchSize int           // Number of events to process per batch
	Interval  time.Duration // Time between processing batches
}

// DefaultOutboxWorkerConfig returns default configuration
func DefaultOutboxWorkerConfig() OutboxWorkerConfig {
	return OutboxWorkerConfig{
		BatchSize: 100,
		Interval:  100 * time.Millisecond,
	}
}

// NewOutboxWorker creates a new outbox worker
func NewOutboxWorker(repo *OutboxRepository, publisher EventPublisher, logger *slog.Logger, cfg OutboxWorkerConfig) *OutboxWorker {
	if cfg.BatchSize == 0 {
		cfg.BatchSize = 100
	}
	if cfg.Interval == 0 {
		cfg.Interval = 100 * time.Millisecond
	}

	return &OutboxWorker{
		repo:      repo,
		publisher: publisher,
		logger:    logger,
		batchSize: cfg.BatchSize,
		interval:  cfg.Interval,
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
	w.logger.Info("Outbox worker started", "batch_size", w.batchSize, "interval", w.interval)
}

// Stop stops the outbox worker
func (w *OutboxWorker) Stop() {
	close(w.stopChan)
	w.wg.Wait()
	w.logger.Info("Outbox worker stopped")
}

// run is the main processing loop
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

// processBatch processes a batch of pending events
func (w *OutboxWorker) processBatch(ctx context.Context) {
	events, err := w.repo.GetPendingEvents(ctx, w.batchSize)
	if err != nil {
		w.logger.Error("Failed to get pending events", "error", err)
		return
	}

	for _, event := range events {
		if err := w.publisher.Publish(ctx, event.EventType, event.Payload); err != nil {
			w.logger.Error("Failed to publish event",
				"event_id", event.ID,
				"event_type", event.EventType,
				"error", err,
			)
			_ = w.repo.MarkAsFailed(ctx, event.ID)
			continue
		}

		if err := w.repo.MarkAsPublished(ctx, event.ID); err != nil {
			w.logger.Error("Failed to mark event as published",
				"event_id", event.ID,
				"error", err,
			)
		} else {
			w.logger.Debug("Event published successfully",
				"event_id", event.ID,
				"event_type", event.EventType,
			)
		}
	}
}

// CreateOutboxEvent is a helper function to create an outbox event in a transaction
func CreateOutboxEvent(tx *gorm.DB, aggregateID string, eventType string, payload interface{}) error {
	data, err := json.Marshal(payload)
	if err != nil {
		return err
	}

	event := &OutboxEvent{
		ID:          uuid.New(),
		AggregateID: aggregateID,
		EventType:   eventType,
		Payload:     data,
		Status:      "PENDING",
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}

	return tx.Create(event).Error
}
