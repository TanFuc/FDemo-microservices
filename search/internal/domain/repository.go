package domain

import "context"

// ProductSearchRepository defines the interface for Elasticsearch operations
type ProductSearchRepository interface {
	// IndexProduct indexes or updates a product document
	IndexProduct(ctx context.Context, product *Product) error
	// DeleteProduct removes a product document by ID
	DeleteProduct(ctx context.Context, id string) error
	// Search performs a search query and returns raw results
	Search(ctx context.Context, params *SearchParams) (*SearchResult, error)
	// EnsureIndex creates the index with mapping if it doesn't exist
	EnsureIndex(ctx context.Context) error
}

// CacheRepository defines the interface for Redis cache operations
type CacheRepository interface {
	// Get retrieves a cached value by key
	Get(ctx context.Context, key string) (string, error)
	// Set stores a value with TTL
	Set(ctx context.Context, key string, value string, ttlSeconds int) error
	// Delete removes a cached key
	Delete(ctx context.Context, key string) error
}

// EventConsumer defines the interface for NATS event consumption
type EventConsumer interface {
	// Start begins consuming events
	Start(ctx context.Context) error
	// Stop gracefully stops the consumer
	Stop() error
}
