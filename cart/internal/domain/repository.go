package domain

import "context"

// CartRedisRepository defines the interface for Redis cart operations.
type CartRedisRepository interface {
	// GetCart retrieves all items from a user's cart.
	GetCart(ctx context.Context, userID string) ([]CartItem, error)

	// SetItem adds or updates an item in the cart.
	// If the item exists, it updates the quantity. If new, it adds the item.
	SetItem(ctx context.Context, userID string, item CartItem) error

	// IncrementQuantity atomically increments the quantity of an existing item.
	IncrementQuantity(ctx context.Context, userID string, skuID string, delta int) error

	// RemoveItem removes a specific item from the cart.
	RemoveItem(ctx context.Context, userID string, skuID string) error

	// ClearCart removes all items from a user's cart.
	ClearCart(ctx context.Context, userID string) error

	// ItemExists checks if an item exists in the cart.
	ItemExists(ctx context.Context, userID string, skuID string) (bool, error)

	// GetItem retrieves a specific item from the cart.
	GetItem(ctx context.Context, userID string, skuID string) (*CartItem, error)

	// GetItemCount returns the number of unique items in the cart.
	GetItemCount(ctx context.Context, userID string) (int, error)

	// RefreshTTL refreshes the TTL on the cart.
	RefreshTTL(ctx context.Context, userID string) error
}

// CartMongoRepository defines the interface for MongoDB cart operations.
type CartMongoRepository interface {
	// UpsertCart inserts or updates the entire cart for a user.
	UpsertCart(ctx context.Context, cart *Cart) error

	// GetCart retrieves a user's cart from MongoDB.
	GetCart(ctx context.Context, userID string) (*Cart, error)

	// DeleteCart removes a user's cart from MongoDB.
	DeleteCart(ctx context.Context, userID string) error

	// RemoveItem removes a specific item from the cart.
	RemoveItem(ctx context.Context, userID string, skuID string) error
}
