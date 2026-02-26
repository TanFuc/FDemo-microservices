package mongodb

import (
	"context"
	"math"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// OutboxEntry is a pending domain event stored in the DB before NATS publish
type OutboxEntry struct {
	ID          primitive.ObjectID `bson:"_id,omitempty"`
	EventID     string             `bson:"eventId"`
	Subject     string             `bson:"subject"`
	Payload     []byte             `bson:"payload"`
	Status      string             `bson:"status"`
	Attempts    int                `bson:"attempts"`
	NextRetryAt time.Time          `bson:"nextRetryAt"`
	CreatedAt   time.Time          `bson:"createdAt"`
	SentAt      *time.Time         `bson:"sentAt,omitempty"`
	Error       string             `bson:"error,omitempty"`
}

const (
	OutboxStatusPending = "PENDING"
	OutboxStatusSent    = "SENT"
	OutboxStatusFailed  = "FAILED"
)

type OutboxRepository struct {
	collection *mongo.Collection
}

func NewOutboxRepository(db *mongo.Database) *OutboxRepository {
	collection := db.Collection("review_outbox")

	// Create indexes
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	indexes := []mongo.IndexModel{
		{
			Keys:    bson.D{{Key: "eventId", Value: 1}},
			Options: options.Index().SetUnique(true),
		},
		{
			Keys: bson.D{
				{Key: "status", Value: 1},
				{Key: "nextRetryAt", Value: 1},
			},
		},
		{
			Keys:    bson.D{{Key: "sentAt", Value: 1}},
			Options: options.Index().SetExpireAfterSeconds(604800), // TTL 7 days
		},
	}

	_, _ = collection.Indexes().CreateMany(ctx, indexes)

	return &OutboxRepository{
		collection: collection,
	}
}

// SavePending saves a new outbox entry with status PENDING
func (r *OutboxRepository) SavePending(ctx context.Context, entry *OutboxEntry) error {
	entry.ID = primitive.NewObjectID()
	entry.Status = OutboxStatusPending
	entry.Attempts = 0
	entry.NextRetryAt = time.Now()
	entry.CreatedAt = time.Now()
	_, err := r.collection.InsertOne(ctx, entry)
	return err
}

// GetPendingEntries fetches up to `limit` entries ready for retry
func (r *OutboxRepository) GetPendingEntries(ctx context.Context, limit int) ([]*OutboxEntry, error) {
	filter := bson.M{
		"status":      bson.M{"$in": []string{OutboxStatusPending, OutboxStatusFailed}},
		"nextRetryAt": bson.M{"$lte": time.Now()},
		"attempts":    bson.M{"$lt": 10},
	}

	opts := options.Find().
		SetLimit(int64(limit)).
		SetSort(bson.D{{Key: "nextRetryAt", Value: 1}})

	cursor, err := r.collection.Find(ctx, filter, opts)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var entries []*OutboxEntry
	if err := cursor.All(ctx, &entries); err != nil {
		return nil, err
	}
	return entries, nil
}

// MarkSent marks an outbox entry as successfully sent
func (r *OutboxRepository) MarkSent(ctx context.Context, id primitive.ObjectID) error {
	now := time.Now()
	_, err := r.collection.UpdateByID(ctx, id, bson.M{
		"$set": bson.M{
			"status": OutboxStatusSent,
			"sentAt": now,
		},
	})
	return err
}

// MarkFailed increments attempts and schedules next retry with backoff
func (r *OutboxRepository) MarkFailed(ctx context.Context, id primitive.ObjectID, errMsg string) error {
	// First, get current entry to read attempts count
	var entry OutboxEntry
	err := r.collection.FindOne(ctx, bson.M{"_id": id}).Decode(&entry)
	if err != nil {
		return err
	}

	// Calculate next retry with exponential backoff (capped at 10 minutes)
	backoffSeconds := math.Min(float64(1<<(entry.Attempts+1)), 600)
	nextRetry := time.Now().Add(time.Duration(backoffSeconds) * time.Second)

	_, err = r.collection.UpdateByID(ctx, id, bson.M{
		"$inc": bson.M{"attempts": 1},
		"$set": bson.M{
			"status":      OutboxStatusFailed,
			"error":       errMsg,
			"nextRetryAt": nextRetry,
		},
	})
	return err
}

// CountPending returns the count of pending outbox entries
func (r *OutboxRepository) CountPending(ctx context.Context) (int64, error) {
	filter := bson.M{
		"status": bson.M{"$in": []string{OutboxStatusPending, OutboxStatusFailed}},
	}
	return r.collection.CountDocuments(ctx, filter)
}
