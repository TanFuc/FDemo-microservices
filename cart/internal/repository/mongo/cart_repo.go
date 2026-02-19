package mongo

import (
	"context"
	"fmt"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"

	"microservices/cart/internal/model"
)

const (
	cartCollection = "carts"
)

// CartRepository implements the MongoDB cart repository interface.
type CartRepository struct {
	collection *mongo.Collection
}

// NewCartRepository creates a new MongoDB cart repository.
func NewCartRepository(db *mongo.Database) *CartRepository {
	return &CartRepository{
		collection: db.Collection(cartCollection),
	}
}

// UpsertCart inserts or updates the entire cart for a user.
func (r *CartRepository) UpsertCart(ctx context.Context, cart *model.Cart) error {
	filter := bson.M{"_id": cart.UserID}
	update := bson.M{
		"$set": bson.M{
			"items":     cart.Items,
			"updatedAt": time.Now(),
		},
	}
	opts := options.Update().SetUpsert(true)

	_, err := r.collection.UpdateOne(ctx, filter, update, opts)
	if err != nil {
		return fmt.Errorf("failed to upsert cart: %w", err)
	}

	return nil
}

// GetCart retrieves a user's cart from MongoDB.
func (r *CartRepository) GetCart(ctx context.Context, userID string) (*model.Cart, error) {
	filter := bson.M{"_id": userID}

	var cart model.Cart
	err := r.collection.FindOne(ctx, filter).Decode(&cart)
	if err == mongo.ErrNoDocuments {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get cart: %w", err)
	}

	return &cart, nil
}

// DeleteCart removes a user's cart from MongoDB.
func (r *CartRepository) DeleteCart(ctx context.Context, userID string) error {
	filter := bson.M{"_id": userID}

	_, err := r.collection.DeleteOne(ctx, filter)
	if err != nil {
		return fmt.Errorf("failed to delete cart: %w", err)
	}

	return nil
}

// RemoveItem removes a specific item from the cart.
func (r *CartRepository) RemoveItem(ctx context.Context, userID string, skuID string) error {
	filter := bson.M{"_id": userID}
	update := bson.M{
		"$pull": bson.M{
			"items": bson.M{"skuId": skuID},
		},
		"$set": bson.M{
			"updatedAt": time.Now(),
		},
	}

	_, err := r.collection.UpdateOne(ctx, filter, update)
	if err != nil {
		return fmt.Errorf("failed to remove item: %w", err)
	}

	return nil
}
