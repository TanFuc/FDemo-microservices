package mongo

import (
	"context"
	"fmt"
	"time"

	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"

	"microservices/cart/internal/config"
)

// NewClient creates a new MongoDB client with the given configuration.
func NewClient(ctx context.Context, cfg *config.MongoConfig) (*mongo.Client, error) {
	ctx, cancel := context.WithTimeout(ctx, time.Duration(cfg.Timeout)*time.Second)
	defer cancel()

	clientOptions := options.Client().ApplyURI(cfg.URI)

	client, err := mongo.Connect(ctx, clientOptions)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to MongoDB: %w", err)
	}

	// Verify connection
	if err := client.Ping(ctx, nil); err != nil {
		return nil, fmt.Errorf("failed to ping MongoDB: %w", err)
	}

	return client, nil
}

// GetDatabase returns a database instance for the given client.
func GetDatabase(client *mongo.Client, dbName string) *mongo.Database {
	return client.Database(dbName)
}

// Close disconnects the MongoDB client.
func Close(ctx context.Context, client *mongo.Client) error {
	if client != nil {
		return client.Disconnect(ctx)
	}
	return nil
}
