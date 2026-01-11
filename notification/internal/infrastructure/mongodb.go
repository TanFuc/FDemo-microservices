package infrastructure

import (
	"context"
	"fmt"
	"log"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"

	"notification-service/internal/models"
)

const (
	NotificationLogsCollection = "notification_logs"
)

// MongoDB holds the MongoDB client and database
type MongoDB struct {
	client   *mongo.Client
	database *mongo.Database
}

// NewMongoDB creates a new MongoDB connection
func NewMongoDB(uri, database string) (*MongoDB, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	clientOpts := options.Client().ApplyURI(uri)
	client, err := mongo.Connect(ctx, clientOpts)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to MongoDB: %w", err)
	}

	// Ping to verify connection
	if err := client.Ping(ctx, nil); err != nil {
		return nil, fmt.Errorf("failed to ping MongoDB: %w", err)
	}

	db := client.Database(database)

	m := &MongoDB{
		client:   client,
		database: db,
	}

	// Create indexes
	if err := m.createIndexes(ctx); err != nil {
		log.Printf("Warning: failed to create indexes: %v", err)
	}

	log.Println("MongoDB connected")
	return m, nil
}

// createIndexes creates necessary indexes for the collections
func (m *MongoDB) createIndexes(ctx context.Context) error {
	collection := m.database.Collection(NotificationLogsCollection)

	indexes := []mongo.IndexModel{
		{
			Keys: bson.D{{Key: "userId", Value: 1}},
		},
		{
			Keys: bson.D{{Key: "status", Value: 1}},
		},
		{
			Keys: bson.D{{Key: "createdAt", Value: -1}},
		},
		{
			Keys: bson.D{
				{Key: "userId", Value: 1},
				{Key: "status", Value: 1},
			},
		},
	}

	_, err := collection.Indexes().CreateMany(ctx, indexes)
	return err
}

// CreateNotificationLog creates a new notification log entry
func (m *MongoDB) CreateNotificationLog(ctx context.Context, log *models.NotificationLog) (*models.NotificationLog, error) {
	collection := m.database.Collection(NotificationLogsCollection)

	log.CreatedAt = time.Now()
	log.UpdatedAt = time.Now()

	result, err := collection.InsertOne(ctx, log)
	if err != nil {
		return nil, fmt.Errorf("failed to insert notification log: %w", err)
	}

	log.ID = result.InsertedID.(primitive.ObjectID)
	return log, nil
}

// UpdateNotificationLogStatus updates the status of a notification log
func (m *MongoDB) UpdateNotificationLogStatus(ctx context.Context, id primitive.ObjectID, status models.NotificationStatus, errorMsg string) error {
	collection := m.database.Collection(NotificationLogsCollection)

	update := bson.M{
		"$set": bson.M{
			"status":    status,
			"updatedAt": time.Now(),
		},
	}

	if errorMsg != "" {
		update["$set"].(bson.M)["error"] = errorMsg
	}

	_, err := collection.UpdateByID(ctx, id, update)
	if err != nil {
		return fmt.Errorf("failed to update notification log: %w", err)
	}

	return nil
}

// GetNotificationLogByID retrieves a notification log by ID
func (m *MongoDB) GetNotificationLogByID(ctx context.Context, id primitive.ObjectID) (*models.NotificationLog, error) {
	collection := m.database.Collection(NotificationLogsCollection)

	var log models.NotificationLog
	err := collection.FindOne(ctx, bson.M{"_id": id}).Decode(&log)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to find notification log: %w", err)
	}

	return &log, nil
}

// GetNotificationLogsByUserID retrieves notification logs for a user
func (m *MongoDB) GetNotificationLogsByUserID(ctx context.Context, userID string, limit int64) ([]*models.NotificationLog, error) {
	collection := m.database.Collection(NotificationLogsCollection)

	opts := options.Find().
		SetSort(bson.D{{Key: "createdAt", Value: -1}}).
		SetLimit(limit)

	cursor, err := collection.Find(ctx, bson.M{"userId": userID}, opts)
	if err != nil {
		return nil, fmt.Errorf("failed to find notification logs: %w", err)
	}
	defer cursor.Close(ctx)

	var logs []*models.NotificationLog
	if err := cursor.All(ctx, &logs); err != nil {
		return nil, fmt.Errorf("failed to decode notification logs: %w", err)
	}

	return logs, nil
}

// GetNotificationLogsByStatus retrieves notification logs by status
func (m *MongoDB) GetNotificationLogsByStatus(ctx context.Context, status models.NotificationStatus, limit int64) ([]*models.NotificationLog, error) {
	collection := m.database.Collection(NotificationLogsCollection)

	opts := options.Find().
		SetSort(bson.D{{Key: "createdAt", Value: -1}}).
		SetLimit(limit)

	cursor, err := collection.Find(ctx, bson.M{"status": status}, opts)
	if err != nil {
		return nil, fmt.Errorf("failed to find notification logs: %w", err)
	}
	defer cursor.Close(ctx)

	var logs []*models.NotificationLog
	if err := cursor.All(ctx, &logs); err != nil {
		return nil, fmt.Errorf("failed to decode notification logs: %w", err)
	}

	return logs, nil
}

// Database returns the MongoDB database
func (m *MongoDB) Database() *mongo.Database {
	return m.database
}

// Close closes the MongoDB connection
func (m *MongoDB) Close(ctx context.Context) error {
	if m.client != nil {
		if err := m.client.Disconnect(ctx); err != nil {
			return fmt.Errorf("error disconnecting MongoDB: %w", err)
		}
		log.Println("MongoDB connection closed")
	}
	return nil
}
