// Package multilayer provides a multi-layer cache implementation.
package multilayer

import (
	"context"
	"log/slog"
	"time"

	"microservices/pkg/cache"
)

// Ensure MultiLayerCache implements the required interfaces.
var (
	_ cache.Cache        = (*MultiLayerCache)(nil)
	_ cache.MetricsCache = (*MultiLayerCache)(nil)
)

// MultiLayerCache implements a two-layer cache (L1 memory + L2 Redis).
type MultiLayerCache struct {
	l1      cache.Cache // Fast, small capacity (memory)
	l2      cache.Cache // Slower, large capacity (Redis)
	logger  *slog.Logger
	metrics cache.MetricsRecorder
	l1TTL   time.Duration // L1 has shorter TTL than L2
	closed  bool
}

// Options contains configuration for MultiLayerCache.
type Options struct {
	Logger  *slog.Logger
	Metrics bool
	L1TTL   time.Duration // TTL for L1 cache (default: 1 minute)
}

// DefaultOptions returns the default options.
func DefaultOptions() *Options {
	return &Options{
		Logger:  slog.Default(),
		Metrics: false,
		L1TTL:   1 * time.Minute,
	}
}

// New creates a new MultiLayerCache.
func New(l1, l2 cache.Cache, opts *Options) (*MultiLayerCache, error) {
	if l1 == nil || l2 == nil {
		return nil, cache.ErrInvalidConfig
	}

	if opts == nil {
		opts = DefaultOptions()
	}

	logger := opts.Logger
	if logger == nil {
		logger = slog.Default()
	}

	l1TTL := opts.L1TTL
	if l1TTL == 0 {
		l1TTL = 1 * time.Minute
	}

	return &MultiLayerCache{
		l1:      l1,
		l2:      l2,
		logger:  logger,
		metrics: cache.NewMetricsRecorder(opts.Metrics),
		l1TTL:   l1TTL,
	}, nil
}

// Get retrieves a value from cache.
// It first tries L1, then L2. If found in L2, it populates L1.
func (c *MultiLayerCache) Get(ctx context.Context, key string, dest interface{}) error {
	if c.closed {
		return cache.ErrClosed
	}

	start := time.Now()

	// Try L1 first
	err := c.l1.Get(ctx, key, dest)
	if err == nil {
		c.metrics.RecordHit(time.Since(start))
		c.logger.Debug("L1 cache hit", "key", key)
		return nil
	}

	if !cache.IsNotFound(err) {
		c.logger.Warn("L1 cache error", "key", key, "error", err)
	}

	// Try L2
	err = c.l2.Get(ctx, key, dest)
	if err != nil {
		if cache.IsNotFound(err) {
			c.metrics.RecordMiss(time.Since(start))
		} else {
			c.metrics.RecordError(err)
			c.logger.Error("L2 cache error", "key", key, "error", err)
		}
		return err
	}

	c.metrics.RecordHit(time.Since(start))
	c.logger.Debug("L2 cache hit", "key", key)

	// Populate L1 from L2 (async to not block)
	go func() {
		if err := c.l1.Set(context.Background(), key, dest, c.l1TTL); err != nil {
			c.logger.Warn("failed to populate L1 from L2", "key", key, "error", err)
		}
	}()

	return nil
}

// Set stores a value in both L1 and L2 caches.
func (c *MultiLayerCache) Set(ctx context.Context, key string, value interface{}, ttl time.Duration) error {
	if c.closed {
		return cache.ErrClosed
	}

	start := time.Now()

	// Calculate L1 TTL (shorter than L2)
	l1TTL := c.l1TTL
	if ttl > 0 && ttl < l1TTL {
		l1TTL = ttl
	}

	// Set in L1 (with shorter TTL)
	if err := c.l1.Set(ctx, key, value, l1TTL); err != nil {
		c.logger.Warn("failed to set in L1", "key", key, "error", err)
	}

	// Set in L2 (with full TTL)
	if err := c.l2.Set(ctx, key, value, ttl); err != nil {
		c.metrics.RecordError(err)
		c.logger.Error("failed to set in L2", "key", key, "error", err)
		return err
	}

	c.metrics.RecordSet(time.Since(start))
	return nil
}

// Delete removes a key from both L1 and L2 caches.
func (c *MultiLayerCache) Delete(ctx context.Context, key string) error {
	if c.closed {
		return cache.ErrClosed
	}

	// Delete from L1
	if err := c.l1.Delete(ctx, key); err != nil && !cache.IsNotFound(err) {
		c.logger.Warn("failed to delete from L1", "key", key, "error", err)
	}

	// Delete from L2
	if err := c.l2.Delete(ctx, key); err != nil && !cache.IsNotFound(err) {
		c.metrics.RecordError(err)
		c.logger.Error("failed to delete from L2", "key", key, "error", err)
		return err
	}

	c.metrics.RecordDelete()
	return nil
}

// Exists checks if a key exists in either L1 or L2.
func (c *MultiLayerCache) Exists(ctx context.Context, key string) (bool, error) {
	if c.closed {
		return false, cache.ErrClosed
	}

	// Check L1 first
	exists, err := c.l1.Exists(ctx, key)
	if err == nil && exists {
		return true, nil
	}

	// Check L2
	return c.l2.Exists(ctx, key)
}

// Close closes both L1 and L2 caches.
func (c *MultiLayerCache) Close() error {
	if c.closed {
		return nil
	}
	c.closed = true

	var lastErr error
	if err := c.l1.Close(); err != nil {
		lastErr = err
	}
	if err := c.l2.Close(); err != nil {
		lastErr = err
	}
	return lastErr
}

// Ping checks if both L1 and L2 are healthy.
func (c *MultiLayerCache) Ping(ctx context.Context) error {
	if c.closed {
		return cache.ErrClosed
	}

	// L1 (memory) should always be available
	if err := c.l1.Ping(ctx); err != nil {
		return err
	}

	// L2 (Redis) is the critical one
	return c.l2.Ping(ctx)
}

// Stats returns combined statistics from both layers.
func (c *MultiLayerCache) Stats() cache.CacheStats {
	return c.metrics.Stats()
}

// ResetStats resets statistics.
func (c *MultiLayerCache) ResetStats() {
	c.metrics.Reset()
}

// L1 returns the L1 cache.
func (c *MultiLayerCache) L1() cache.Cache {
	return c.l1
}

// L2 returns the L2 cache.
func (c *MultiLayerCache) L2() cache.Cache {
	return c.l2
}

// InvalidateL1 removes a key only from L1 (useful for cache consistency).
func (c *MultiLayerCache) InvalidateL1(ctx context.Context, key string) error {
	return c.l1.Delete(ctx, key)
}

// WarmL1 pre-populates L1 from L2 for the given keys.
func (c *MultiLayerCache) WarmL1(ctx context.Context, keys []string) error {
	for _, key := range keys {
		// Get from L2
		var value interface{}
		if err := c.l2.Get(ctx, key, &value); err != nil {
			if cache.IsNotFound(err) {
				continue
			}
			return err
		}

		// Set in L1
		if err := c.l1.Set(ctx, key, value, c.l1TTL); err != nil {
			c.logger.Warn("failed to warm L1", "key", key, "error", err)
		}
	}
	return nil
}

// SetL1TTL updates the L1 TTL.
func (c *MultiLayerCache) SetL1TTL(ttl time.Duration) {
	c.l1TTL = ttl
}

// GetL1TTL returns the current L1 TTL.
func (c *MultiLayerCache) GetL1TTL() time.Duration {
	return c.l1TTL
}
