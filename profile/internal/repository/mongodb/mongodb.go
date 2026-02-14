package mongodb

import (
	"context"
	"time"

	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.mongodb.org/mongo-driver/mongo/readpref"

	"microservices/profile/internal/config"
	"microservices/profile/pkg/logger"
)

type MongoDB struct {
	client   *mongo.Client
	database *mongo.Database
}

func NewMongoDB(cfg *config.Config) (*MongoDB, error) {
	ctx, cancel := context.WithTimeout(context.Background(), cfg.MongoDB.ConnectTimeout)
	defer cancel()

	clientOptions := options.Client().
		ApplyURI(cfg.MongoDB.URI).
		SetMaxPoolSize(cfg.MongoDB.MaxPoolSize).
		SetMinPoolSize(cfg.MongoDB.MinPoolSize)

	client, err := mongo.Connect(ctx, clientOptions)
	if err != nil {
		return nil, err
	}

	// Ping to verify connection
	if err := client.Ping(ctx, readpref.Primary()); err != nil {
		return nil, err
	}

	logger.Info().
		Str("database", cfg.MongoDB.Database).
		Msg("Connected to MongoDB")

	db := &MongoDB{
		client:   client,
		database: client.Database(cfg.MongoDB.Database),
	}

	// Create indexes
	if err := db.CreateIndexes(ctx); err != nil {
		logger.Warn().Err(err).Msg("Failed to create indexes")
	}

	return db, nil
}

func (m *MongoDB) Database() *mongo.Database {
	return m.database
}

func (m *MongoDB) Client() *mongo.Client {
	return m.client
}

func (m *MongoDB) Collection(name string) *mongo.Collection {
	return m.database.Collection(name)
}

func (m *MongoDB) Close(ctx context.Context) error {
	return m.client.Disconnect(ctx)
}

func (m *MongoDB) HealthCheck(ctx context.Context) error {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	return m.client.Ping(ctx, readpref.Primary())
}

func (m *MongoDB) CreateIndexes(ctx context.Context) error {
	// Profile indexes
	profileCollection := m.Collection("profiles")
	profileIndexes := []mongo.IndexModel{
		{
			Keys:    map[string]interface{}{"userId": 1},
			Options: options.Index().SetUnique(true),
		},
		{
			Keys: map[string]interface{}{"shopConfig.shopSlug": 1},
			Options: options.Index().
				SetUnique(true).
				SetSparse(true).
				SetPartialFilterExpression(map[string]interface{}{
					"shopConfig.shopSlug": map[string]interface{}{"$exists": true},
				}),
		},
		{
			Keys: map[string]interface{}{"membershipTier": 1},
		},
		{
			Keys: map[string]interface{}{"lastActiveAt": -1},
		},
		{
			Keys: map[string]interface{}{"isDeleted": 1},
		},
		{
			Keys: map[string]interface{}{"shopConfig.verificationStatus": 1},
		},
	}

	_, err := profileCollection.Indexes().CreateMany(ctx, profileIndexes)
	if err != nil {
		return err
	}

	// Address indexes
	addressCollection := m.Collection("addresses")
	addressIndexes := []mongo.IndexModel{
		{
			Keys: map[string]interface{}{"userId": 1, "isDeleted": 1},
		},
		{
			Keys: map[string]interface{}{"userId": 1, "isDefault": 1},
		},
		{
			Keys: map[string]interface{}{"userId": 1, "isDefaultBilling": 1},
		},
		{
			Keys: map[string]interface{}{"userId": 1, "isDefaultPickup": 1},
		},
	}

	_, err = addressCollection.Indexes().CreateMany(ctx, addressIndexes)
	if err != nil {
		return err
	}

	logger.Info().Msg("MongoDB indexes created successfully")
	return nil
}
