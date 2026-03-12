package usercontext

import (
	"context"
	"errors"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// MongoRepository implements UserContextRepository for MongoDB
type MongoRepository struct {
	collection *mongo.Collection
}

// NewMongoRepository creates a new MongoDB-based user context repository
func NewMongoRepository(db *mongo.Database, collectionName string) (*MongoRepository, error) {
	if collectionName == "" {
		collectionName = "users"
	}

	collection := db.Collection(collectionName)

	// Create indexes
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	indexes := []mongo.IndexModel{
		{
			Keys:    bson.D{{Key: "userId", Value: 1}},
			Options: options.Index().SetUnique(true),
		},
		{
			Keys: bson.D{{Key: "shopId", Value: 1}},
		},
		{
			Keys: bson.D{{Key: "email", Value: 1}},
		},
	}

	_, err := collection.Indexes().CreateMany(ctx, indexes)
	if err != nil {
		// Ignore duplicate key errors for indexes
		if !mongo.IsDuplicateKeyError(err) {
			return nil, err
		}
	}

	return &MongoRepository{
		collection: collection,
	}, nil
}

// FindByUserID retrieves user context from MongoDB
func (r *MongoRepository) FindByUserID(ctx context.Context, userID string) (*UserContext, error) {
	var user UserContext
	err := r.collection.FindOne(ctx, bson.M{"userId": userID}).Decode(&user)
	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return nil, errors.New("user not found")
		}
		return nil, err
	}
	return &user, nil
}

// Upsert inserts or updates user context in MongoDB
func (r *MongoRepository) Upsert(ctx context.Context, user *UserContext) error {
	opts := options.Update().SetUpsert(true)
	filter := bson.M{"userId": user.UserID}
	update := bson.M{
		"$set": bson.M{
			"userId":      user.UserID,
			"role":        user.Role,
			"shopId":      user.ShopID,
			"displayName": user.DisplayName,
			"avatarUrl":   user.AvatarURL,
			"email":       user.Email,
			"status":      user.Status,
			"version":     user.Version,
			"updatedAt":   user.UpdatedAt,
		},
		"$setOnInsert": bson.M{
			"createdAt": user.CreatedAt,
		},
	}

	_, err := r.collection.UpdateOne(ctx, filter, update, opts)
	return err
}

// Delete removes user context from MongoDB (soft delete by setting status)
func (r *MongoRepository) Delete(ctx context.Context, userID string) error {
	filter := bson.M{"userId": userID}
	update := bson.M{
		"$set": bson.M{
			"status":    StatusInactive,
			"updatedAt": time.Now(),
		},
	}
	_, err := r.collection.UpdateOne(ctx, filter, update)
	return err
}

// FindByShopID retrieves user context by shop ID
func (r *MongoRepository) FindByShopID(ctx context.Context, shopID string) (*UserContext, error) {
	var user UserContext
	err := r.collection.FindOne(ctx, bson.M{"shopId": shopID}).Decode(&user)
	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return nil, errors.New("user not found")
		}
		return nil, err
	}
	return &user, nil
}

// FindByEmail retrieves user context by email
func (r *MongoRepository) FindByEmail(ctx context.Context, email string) (*UserContext, error) {
	var user UserContext
	err := r.collection.FindOne(ctx, bson.M{"email": email}).Decode(&user)
	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return nil, errors.New("user not found")
		}
		return nil, err
	}
	return &user, nil
}
