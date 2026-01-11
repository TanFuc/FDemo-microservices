package ports

import (
	"context"
	"time"
)

// Cache defines the interface for caching operations
type Cache interface {
	// Get retrieves a value from cache
	Get(ctx context.Context, key string) (string, error)

	// Set stores a value in cache with TTL
	Set(ctx context.Context, key string, value string, ttl time.Duration) error

	// Delete removes a value from cache
	Delete(ctx context.Context, key string) error

	// Exists checks if a key exists in cache
	Exists(ctx context.Context, key string) (bool, error)
}

// FeeCache provides specialized caching for shipping fees
type FeeCache interface {
	// GetFee retrieves cached shipping fee
	GetFee(ctx context.Context, provider string, fromDistrict, toDistrict, weight int) (float64, bool, error)

	// SetFee caches a shipping fee
	SetFee(ctx context.Context, provider string, fromDistrict, toDistrict, weight int, fee float64) error
}
