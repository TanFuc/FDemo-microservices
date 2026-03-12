package outbox

import (
	"context"
	"encoding/json"
	"log"
	"sync"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// OutboxStatus represents the status of an outbox event
type OutboxStatus string

const (
	StatusPending   OutboxStatus = "PENDING"
	StatusPublished OutboxStatus = "PUBLISHED"
	StatusFailed    OutboxStatus = "FAILED"
)

// MongoOutboxEvent represents an event stored in the MongoDB outbox collection
type MongoOutboxEvent struct {
	ID          primitive.ObjectID `bson:"_id,omitempty"`
	AggregateID string             `bson:"aggregateId"`
	EventType   string             `bson:"eventType"`
	Payload     json.RawMessage    `bson:"payload"`
	Status      OutboxStatus       `bson:"status"`
	RetryCount  int                `bson:"retryCount"`
	CreatedAt   time.Time          `bson:"createdAt"`
	UpdatedAt   time.Time          `bson:"updatedAt"`
	PublishedAt *time.Time         `bson:"publishedAt,omitempty"`
}

// MongoOutboxRepository handles outbox event persistence for MongoDB
type MongoOutboxRepository struct {
	collection *mongo.Collection
}

// NewMongoOutboxRepository creates a new MongoDB outbox repository
func NewMongoOutboxRepository(db *mongo.Database, collectionName string) (*MongoOutboxRepository, error) {
	if collectionName == "" {
		collectionName = "outbox_events"
	}

	collection := db.Collection(collectionName)

	// Create indexes
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	indexes := []mongo.IndexModel{
		{
			Keys: bson.D{{Key: "status", Value: 1}, {Key: "createdAt", Value: 1}},
		},
		{
			Keys: bson.D{{Key: "aggregateId", Value: 1}},
		},
		{
			Keys: bson.D{{Key: "eventType", Value: 1}},
		},
		{
			Keys:    bson.D{{Key: "status", Value: 1}, {Key: "publishedAt", Value: 1}},
			Options: options.Index().SetExpireAfterSeconds(86400 * 7), // 7 days TTL for published events
		},
	}

	_, err := collection.Indexes().CreateMany(ctx, indexes)
	if err != nil {
		// Ignore duplicate key errors
		if !mongo.IsDuplicateKeyError(err) {
			log.Printf("Warning: Failed to create outbox indexes: %v", err)
		}
	}

	return &MongoOutboxRepository{
		collection: collection,
	}, nil
}

// Create creates a new outbox event within a session/transaction
func (r *MongoOutboxRepository) Create(ctx context.Context, event *MongoOutboxEvent) error {
	event.ID = primitive.NewObjectID()
	event.Status = StatusPending
	event.CreatedAt = time.Now()
	event.UpdatedAt = time.Now()

	_, err := r.collection.InsertOne(ctx, event)
	return err
}

// CreateWithSession creates a new outbox event within a MongoDB session (for transactions)
func (r *MongoOutboxRepository) CreateWithSession(sessCtx mongo.SessionContext, event *MongoOutboxEvent) error {
	event.ID = primitive.NewObjectID()
	event.Status = StatusPending
	event.CreatedAt = time.Now()
	event.UpdatedAt = time.Now()

	_, err := r.collection.InsertOne(sessCtx, event)
	return err
}

// GetPendingEvents retrieves pending events for publishing
func (r *MongoOutboxRepository) GetPendingEvents(ctx context.Context, limit int) ([]*MongoOutboxEvent, error) {
	filter := bson.M{"status": StatusPending}
	opts := options.Find().
		SetSort(bson.D{{Key: "createdAt", Value: 1}}).
		SetLimit(int64(limit))

	cursor, err := r.collection.Find(ctx, filter, opts)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var events []*MongoOutboxEvent
	if err := cursor.All(ctx, &events); err != nil {
		return nil, err
	}
	return events, nil
}

// MarkAsPublished marks an event as successfully published
func (r *MongoOutboxRepository) MarkAsPublished(ctx context.Context, eventID primitive.ObjectID) error {
	now := time.Now()
	filter := bson.M{"_id": eventID}
	update := bson.M{
		"$set": bson.M{
			"status":      StatusPublished,
			"publishedAt": &now,
			"updatedAt":   now,
		},
	}
	_, err := r.collection.UpdateOne(ctx, filter, update)
	return err
}

// MarkAsFailed marks an event as failed and increments retry count
func (r *MongoOutboxRepository) MarkAsFailed(ctx context.Context, eventID primitive.ObjectID) error {
	filter := bson.M{"_id": eventID}
	update := bson.M{
		"$set": bson.M{
			"status":    StatusPending, // Reset to pending for retry
			"updatedAt": time.Now(),
		},
		"$inc": bson.M{
			"retryCount": 1,
		},
	}
	_, err := r.collection.UpdateOne(ctx, filter, update)
	return err
}

// DeleteOldPublishedEvents removes published events older than the given duration
func (r *MongoOutboxRepository) DeleteOldPublishedEvents(ctx context.Context, olderThan time.Duration) (int64, error) {
	cutoff := time.Now().Add(-olderThan)
	filter := bson.M{
		"status":      StatusPublished,
		"publishedAt": bson.M{"$lt": cutoff},
	}
	result, err := r.collection.DeleteMany(ctx, filter)
	if err != nil {
		return 0, err
	}
	return result.DeletedCount, nil
}

// OutboxPublisher interface for publishing events
type OutboxPublisher interface {
	Publish(ctx context.Context, eventType string, payload []byte) error
}

// MongoOutboxWorker processes outbox events and publishes them
type MongoOutboxWorker struct {
	repo        *MongoOutboxRepository
	publisher   OutboxPublisher
	serviceName string
	batchSize   int
	interval    time.Duration
	stopChan    chan struct{}
	wg          sync.WaitGroup
}

// MongoOutboxWorkerConfig holds configuration for the outbox worker
type MongoOutboxWorkerConfig struct {
	ServiceName string        // Service name for logging
	BatchSize   int           // Number of events to process per batch
	Interval    time.Duration // Time between processing batches
}

// DefaultMongoOutboxWorkerConfig returns default configuration
func DefaultMongoOutboxWorkerConfig(serviceName string) MongoOutboxWorkerConfig {
	return MongoOutboxWorkerConfig{
		ServiceName: serviceName,
		BatchSize:   100,
		Interval:    100 * time.Millisecond,
	}
}

// NewMongoOutboxWorker creates a new MongoDB outbox worker
func NewMongoOutboxWorker(repo *MongoOutboxRepository, publisher OutboxPublisher, cfg MongoOutboxWorkerConfig) *MongoOutboxWorker {
	if cfg.BatchSize == 0 {
		cfg.BatchSize = 100
	}
	if cfg.Interval == 0 {
		cfg.Interval = 100 * time.Millisecond
	}

	return &MongoOutboxWorker{
		repo:        repo,
		publisher:   publisher,
		serviceName: cfg.ServiceName,
		batchSize:   cfg.BatchSize,
		interval:    cfg.Interval,
		stopChan:    make(chan struct{}),
	}
}

// Start starts the outbox worker
func (w *MongoOutboxWorker) Start(ctx context.Context) {
	w.wg.Add(1)
	go func() {
		defer w.wg.Done()
		w.run(ctx)
	}()
	log.Printf("[%s] Outbox worker started (batch_size=%d, interval=%v)", w.serviceName, w.batchSize, w.interval)
}

// Stop stops the outbox worker
func (w *MongoOutboxWorker) Stop() {
	close(w.stopChan)
	w.wg.Wait()
	log.Printf("[%s] Outbox worker stopped", w.serviceName)
}

// run is the main processing loop
func (w *MongoOutboxWorker) run(ctx context.Context) {
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
func (w *MongoOutboxWorker) processBatch(ctx context.Context) {
	events, err := w.repo.GetPendingEvents(ctx, w.batchSize)
	if err != nil {
		log.Printf("[%s] Failed to get pending events: %v", w.serviceName, err)
		return
	}

	for _, event := range events {
		if err := w.publisher.Publish(ctx, event.EventType, event.Payload); err != nil {
			log.Printf("[%s] Failed to publish event %s (%s): %v",
				w.serviceName, event.ID.Hex(), event.EventType, err)
			_ = w.repo.MarkAsFailed(ctx, event.ID)
			continue
		}

		if err := w.repo.MarkAsPublished(ctx, event.ID); err != nil {
			log.Printf("[%s] Failed to mark event as published %s: %v",
				w.serviceName, event.ID.Hex(), err)
		}
	}
}

// CreateOutboxEvent is a helper function to create an outbox event
func CreateOutboxEvent(aggregateID, eventType string, payload interface{}) (*MongoOutboxEvent, error) {
	data, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}

	return &MongoOutboxEvent{
		AggregateID: aggregateID,
		EventType:   eventType,
		Payload:     data,
	}, nil
}
