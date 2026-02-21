package redis

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"

	"microservices/cart/internal/model"
)

const (
	cartKeyPrefix = "cart:"
	cartTTL       = 30 * 24 * time.Hour // 30 days
)

// CartRepository implements the Redis cart repository interface.
type CartRepository struct {
	client *redis.Client
}

// NewCartRepository creates a new Redis cart repository.
func NewCartRepository(client *redis.Client) *CartRepository {
	return &CartRepository{client: client}
}

func cartKey(userID string) string {
	return cartKeyPrefix + userID
}

func itemField(skuID string) string {
	return skuID
}

// GetCart retrieves all items from a user's cart.
func (r *CartRepository) GetCart(ctx context.Context, userID string) ([]model.CartItem, error) {
	key := cartKey(userID)

	result, err := r.client.HGetAll(ctx, key).Result()
	if err != nil {
		return nil, fmt.Errorf("failed to get cart: %w", err)
	}

	if len(result) == 0 {
		return nil, nil
	}

	items := make([]model.CartItem, 0, len(result))
	for _, data := range result {
		var item model.CartItem
		if err := json.Unmarshal([]byte(data), &item); err != nil {
			continue // Skip invalid items
		}
		items = append(items, item)
	}

	return items, nil
}

// SetItem adds or updates an item in the cart.
func (r *CartRepository) SetItem(ctx context.Context, userID string, item model.CartItem) error {
	key := cartKey(userID)
	field := itemField(item.SkuID)

	data, err := json.Marshal(item)
	if err != nil {
		return fmt.Errorf("failed to marshal item: %w", err)
	}

	pipe := r.client.Pipeline()
	pipe.HSet(ctx, key, field, data)
	pipe.Expire(ctx, key, cartTTL)

	_, err = pipe.Exec(ctx)
	if err != nil {
		return fmt.Errorf("failed to set item: %w", err)
	}

	return nil
}

// IncrementQuantity atomically increments the quantity of an existing item.
func (r *CartRepository) IncrementQuantity(ctx context.Context, userID string, skuID string, delta int) error {
	key := cartKey(userID)
	field := itemField(skuID)

	// Get current item
	data, err := r.client.HGet(ctx, key, field).Result()
	if err != nil {
		return fmt.Errorf("failed to get item: %w", err)
	}

	var item model.CartItem
	if err := json.Unmarshal([]byte(data), &item); err != nil {
		return fmt.Errorf("failed to unmarshal item: %w", err)
	}

	item.Quantity += delta

	return r.SetItem(ctx, userID, item)
}

// RemoveItem removes a specific item from the cart.
func (r *CartRepository) RemoveItem(ctx context.Context, userID string, skuID string) error {
	key := cartKey(userID)
	field := itemField(skuID)

	if err := r.client.HDel(ctx, key, field).Err(); err != nil {
		return fmt.Errorf("failed to remove item: %w", err)
	}

	return nil
}

// ClearCart removes all items from a user's cart.
func (r *CartRepository) ClearCart(ctx context.Context, userID string) error {
	key := cartKey(userID)

	if err := r.client.Del(ctx, key).Err(); err != nil {
		return fmt.Errorf("failed to clear cart: %w", err)
	}

	return nil
}

// ItemExists checks if an item exists in the cart.
func (r *CartRepository) ItemExists(ctx context.Context, userID string, skuID string) (bool, error) {
	key := cartKey(userID)
	field := itemField(skuID)

	exists, err := r.client.HExists(ctx, key, field).Result()
	if err != nil {
		return false, fmt.Errorf("failed to check item existence: %w", err)
	}

	return exists, nil
}

// GetItem retrieves a specific item from the cart.
func (r *CartRepository) GetItem(ctx context.Context, userID string, skuID string) (*model.CartItem, error) {
	key := cartKey(userID)
	field := itemField(skuID)

	data, err := r.client.HGet(ctx, key, field).Result()
	if err == redis.Nil {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get item: %w", err)
	}

	var item model.CartItem
	if err := json.Unmarshal([]byte(data), &item); err != nil {
		return nil, fmt.Errorf("failed to unmarshal item: %w", err)
	}

	return &item, nil
}

// GetItemCount returns the number of unique items in the cart.
func (r *CartRepository) GetItemCount(ctx context.Context, userID string) (int, error) {
	key := cartKey(userID)

	count, err := r.client.HLen(ctx, key).Result()
	if err != nil {
		return 0, fmt.Errorf("failed to get item count: %w", err)
	}

	return int(count), nil
}

// RefreshTTL refreshes the TTL on the cart.
func (r *CartRepository) RefreshTTL(ctx context.Context, userID string) error {
	key := cartKey(userID)

	if err := r.client.Expire(ctx, key, cartTTL).Err(); err != nil {
		return fmt.Errorf("failed to refresh TTL: %w", err)
	}

	return nil
}

// SetCartFromItems sets cart items from a slice (for loading from MongoDB).
func (r *CartRepository) SetCartFromItems(ctx context.Context, userID string, items []model.CartItem) error {
	if len(items) == 0 {
		return nil
	}

	key := cartKey(userID)
	pipe := r.client.Pipeline()

	for _, item := range items {
		data, err := json.Marshal(item)
		if err != nil {
			continue
		}
		pipe.HSet(ctx, key, itemField(item.SkuID), data)
	}

	pipe.Expire(ctx, key, cartTTL)

	_, err := pipe.Exec(ctx)
	if err != nil {
		return fmt.Errorf("failed to set cart from items: %w", err)
	}

	return nil
}
