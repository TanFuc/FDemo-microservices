package cache

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"sync/atomic"
	"time"
)

// =============================================================================
// Serialization
// =============================================================================

// Serializer defines the interface for data serialization/deserialization.
type Serializer interface {
	// Marshal serializes the value to bytes.
	Marshal(v interface{}) ([]byte, error)

	// Unmarshal deserializes the bytes into the destination.
	Unmarshal(data []byte, v interface{}) error

	// ContentType returns the content type of the serialized data.
	ContentType() string
}

// SerializerType represents the type of serializer.
type SerializerType string

const (
	// SerializerJSON uses JSON serialization.
	SerializerJSON SerializerType = "json"

	// SerializerMsgPack uses MessagePack serialization (requires msgpack package).
	SerializerMsgPack SerializerType = "msgpack"
)

// JSONSerializer implements Serializer using JSON encoding.
type JSONSerializer struct{}

// NewJSONSerializer creates a new JSON serializer.
func NewJSONSerializer() *JSONSerializer {
	return &JSONSerializer{}
}

// Marshal serializes the value to JSON bytes.
func (s *JSONSerializer) Marshal(v interface{}) ([]byte, error) {
	data, err := json.Marshal(v)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrSerialization, err)
	}
	return data, nil
}

// Unmarshal deserializes the JSON bytes into the destination.
func (s *JSONSerializer) Unmarshal(data []byte, v interface{}) error {
	if err := json.Unmarshal(data, v); err != nil {
		return fmt.Errorf("%w: %v", ErrDeserialization, err)
	}
	return nil
}

// ContentType returns the MIME type for JSON.
func (s *JSONSerializer) ContentType() string {
	return "application/json"
}

// NewSerializer creates a serializer based on the type.
// Currently only JSON is supported. MessagePack can be added as an optional dependency.
func NewSerializer(serializerType SerializerType) Serializer {
	switch serializerType {
	case SerializerMsgPack:
		// MessagePack support can be added by implementing a MsgPackSerializer
		// that wraps github.com/vmihailenco/msgpack/v5
		// For now, fall back to JSON
		return NewJSONSerializer()
	case SerializerJSON:
		fallthrough
	default:
		return NewJSONSerializer()
	}
}

// DefaultSerializer returns the default serializer (JSON).
func DefaultSerializer() Serializer {
	return NewJSONSerializer()
}

// MarshalString is a helper that marshals and returns a string.
func MarshalString(s Serializer, v interface{}) (string, error) {
	data, err := s.Marshal(v)
	if err != nil {
		return "", err
	}
	return string(data), nil
}

// UnmarshalString is a helper that unmarshals from a string.
func UnmarshalString(s Serializer, data string, v interface{}) error {
	return s.Unmarshal([]byte(data), v)
}

// =============================================================================
// Generic Helpers
// =============================================================================

// GetTyped is a type-safe version of Get using generics.
func GetTyped[T any](ctx context.Context, c Cache, key string) (T, error) {
	var result T
	err := c.Get(ctx, key, &result)
	return result, err
}

// SetTyped is a type-safe version of Set using generics.
func SetTyped[T any](ctx context.Context, c Cache, key string, value T, ttl time.Duration) error {
	return c.Set(ctx, key, value, ttl)
}

// GetListTyped retrieves a typed list from cache.
func GetListTyped[T any](ctx context.Context, c ListCache, key string) ([]T, error) {
	var result []T
	err := c.GetList(ctx, key, &result)
	return result, err
}

// SetListTyped stores a typed list in cache.
func SetListTyped[T any](ctx context.Context, c ListCache, key string, list []T, ttl time.Duration) error {
	return c.SetList(ctx, key, list, ttl)
}

// AppendToListTyped adds a typed item to a cached list.
func AppendToListTyped[T any](ctx context.Context, c ListCache, key string, item T, ttl time.Duration) error {
	return c.AppendToList(ctx, key, item, ttl)
}

// Remember implements the cache-aside pattern.
// It tries to get a value from cache, and if not found, computes and stores it.
func Remember[T any](ctx context.Context, c Cache, key string, ttl time.Duration, fn func() (T, error)) (T, error) {
	var result T

	// Try cache first
	err := c.Get(ctx, key, &result)
	if err == nil {
		return result, nil
	}

	// If error is not a cache miss, return it
	if !IsNotFound(err) {
		return result, err
	}

	// Cache miss - compute value
	result, err = fn()
	if err != nil {
		return result, err
	}

	// Store in cache (ignore cache errors)
	_ = c.Set(ctx, key, result, ttl)

	return result, nil
}

// RememberForever implements the cache-aside pattern without TTL.
func RememberForever[T any](ctx context.Context, c Cache, key string, fn func() (T, error)) (T, error) {
	return Remember(ctx, c, key, 0, fn)
}

// RememberList implements the cache-aside pattern for lists.
func RememberList[T any](ctx context.Context, c ListCache, key string, ttl time.Duration, fn func() ([]T, error)) ([]T, error) {
	// Try cache first
	result, err := GetListTyped[T](ctx, c, key)
	if err == nil {
		return result, nil
	}

	// If error is not a cache miss, return it
	if !IsNotFound(err) {
		return nil, err
	}

	// Cache miss - compute value
	result, err = fn()
	if err != nil {
		return nil, err
	}

	// Store in cache (ignore cache errors)
	_ = SetListTyped(ctx, c, key, result, ttl)

	return result, nil
}

// GetOrSet gets a value from cache, or sets it if not found.
// Unlike Remember, it returns whether the value was retrieved from cache.
func GetOrSet[T any](ctx context.Context, c Cache, key string, ttl time.Duration, fn func() (T, error)) (value T, fromCache bool, err error) {
	// Try cache first
	err = c.Get(ctx, key, &value)
	if err == nil {
		return value, true, nil
	}

	// If error is not a cache miss, return it
	if !IsNotFound(err) {
		return value, false, err
	}

	// Cache miss - compute value
	value, err = fn()
	if err != nil {
		return value, false, err
	}

	// Store in cache (ignore cache errors)
	_ = c.Set(ctx, key, value, ttl)

	return value, false, nil
}

// MGetTyped retrieves multiple typed values from a batch cache.
func MGetTyped[T any](ctx context.Context, c BatchCache, keys []string) (map[string]T, error) {
	// Get raw data
	rawResults, err := c.MGet(ctx, keys)
	if err != nil {
		return nil, err
	}

	// Get serializer if available
	serializer := DefaultSerializer()
	if s, ok := c.(interface{ GetSerializer() Serializer }); ok {
		serializer = s.GetSerializer()
	}

	// Unmarshal results
	results := make(map[string]T, len(rawResults))
	for key, data := range rawResults {
		var value T
		if err := serializer.Unmarshal(data, &value); err != nil {
			continue // Skip items that fail to unmarshal
		}
		results[key] = value
	}

	return results, nil
}

// MSetTyped sets multiple typed values in a batch cache.
func MSetTyped[T any](ctx context.Context, c BatchCache, items map[string]T, ttl time.Duration) error {
	// Convert to interface map
	interfaceItems := make(map[string]interface{}, len(items))
	for key, value := range items {
		interfaceItems[key] = value
	}

	return c.MSet(ctx, interfaceItems, ttl)
}

// Prefetch loads multiple items into cache in the background.
// It takes a list of keys and a loader function that fetches the values.
func Prefetch[T any](ctx context.Context, c Cache, keys []string, ttl time.Duration, loader func(key string) (T, error)) <-chan error {
	errCh := make(chan error, 1)

	go func() {
		defer close(errCh)

		for _, key := range keys {
			// Check if context is done
			select {
			case <-ctx.Done():
				errCh <- ctx.Err()
				return
			default:
			}

			// Check if already in cache
			var existing T
			if err := c.Get(ctx, key, &existing); err == nil {
				continue
			}

			// Load and cache
			value, err := loader(key)
			if err != nil {
				continue // Skip errors, don't fail the whole prefetch
			}

			_ = c.Set(ctx, key, value, ttl)
		}
	}()

	return errCh
}

// CacheValue wraps a value with metadata for caching.
type CacheValue[T any] struct {
	Value     T         `json:"value"`
	CachedAt  time.Time `json:"cachedAt"`
	ExpiresAt time.Time `json:"expiresAt,omitempty"`
}

// NewCacheValue creates a new CacheValue with the given TTL.
func NewCacheValue[T any](value T, ttl time.Duration) CacheValue[T] {
	cv := CacheValue[T]{
		Value:    value,
		CachedAt: time.Now(),
	}
	if ttl > 0 {
		cv.ExpiresAt = cv.CachedAt.Add(ttl)
	}
	return cv
}

// IsExpired returns true if the cache value has expired.
func (cv CacheValue[T]) IsExpired() bool {
	if cv.ExpiresAt.IsZero() {
		return false
	}
	return time.Now().After(cv.ExpiresAt)
}

// Age returns how long the value has been cached.
func (cv CacheValue[T]) Age() time.Duration {
	return time.Since(cv.CachedAt)
}

// TTLRemaining returns the remaining TTL.
func (cv CacheValue[T]) TTLRemaining() time.Duration {
	if cv.ExpiresAt.IsZero() {
		return -1 // No expiration
	}
	remaining := time.Until(cv.ExpiresAt)
	if remaining < 0 {
		return 0
	}
	return remaining
}

// metricsCollector collects cache operation metrics.
type metricsCollector struct {
	hits     atomic.Int64
	misses   atomic.Int64
	sets     atomic.Int64
	deletes  atomic.Int64
	errors   atomic.Int64
	size     atomic.Int64
	memUsage atomic.Int64

	getLatencies []time.Duration
	setLatencies []time.Duration
	maxSamples   int
	mu           sync.RWMutex

	lastError   error
	lastErrorAt time.Time
	errorMu     sync.RWMutex
}

// newMetricsCollector creates a new metrics collector.
func newMetricsCollector() *metricsCollector {
	return &metricsCollector{
		getLatencies: make([]time.Duration, 0, 1000),
		setLatencies: make([]time.Duration, 0, 1000),
		maxSamples:   1000,
	}
}

// RecordHit records a cache hit with its latency.
func (m *metricsCollector) RecordHit(latency time.Duration) {
	m.hits.Add(1)
	m.addGetLatency(latency)
}

// RecordMiss records a cache miss with its latency.
func (m *metricsCollector) RecordMiss(latency time.Duration) {
	m.misses.Add(1)
	m.addGetLatency(latency)
}

// RecordSet records a set operation with its latency.
func (m *metricsCollector) RecordSet(latency time.Duration) {
	m.sets.Add(1)
	m.addSetLatency(latency)
}

// RecordDelete records a delete operation.
func (m *metricsCollector) RecordDelete() {
	m.deletes.Add(1)
}

// RecordError records an error.
func (m *metricsCollector) RecordError(err error) {
	m.errors.Add(1)
	m.errorMu.Lock()
	defer m.errorMu.Unlock()
	m.lastError = err
	m.lastErrorAt = time.Now()
}

// UpdateSize updates the current cache size.
func (m *metricsCollector) UpdateSize(size int64) {
	m.size.Store(size)
}

// UpdateMemoryUsage updates the current memory usage.
func (m *metricsCollector) UpdateMemoryUsage(bytes int64) {
	m.memUsage.Store(bytes)
}

// addGetLatency adds a Get operation latency sample.
func (m *metricsCollector) addGetLatency(latency time.Duration) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if len(m.getLatencies) >= m.maxSamples {
		// Remove oldest sample
		m.getLatencies = m.getLatencies[1:]
	}
	m.getLatencies = append(m.getLatencies, latency)
}

// addSetLatency adds a Set operation latency sample.
func (m *metricsCollector) addSetLatency(latency time.Duration) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if len(m.setLatencies) >= m.maxSamples {
		// Remove oldest sample
		m.setLatencies = m.setLatencies[1:]
	}
	m.setLatencies = append(m.setLatencies, latency)
}

// avgLatency calculates the average latency from samples.
func avgLatency(latencies []time.Duration) time.Duration {
	if len(latencies) == 0 {
		return 0
	}
	var total time.Duration
	for _, l := range latencies {
		total += l
	}
	return total / time.Duration(len(latencies))
}

// Stats returns the current cache statistics.
func (m *metricsCollector) Stats() CacheStats {
	hits := m.hits.Load()
	misses := m.misses.Load()

	var hitRate float64
	total := hits + misses
	if total > 0 {
		hitRate = float64(hits) / float64(total)
	}

	m.mu.RLock()
	avgGetLat := avgLatency(m.getLatencies)
	avgSetLat := avgLatency(m.setLatencies)
	m.mu.RUnlock()

	m.errorMu.RLock()
	lastErr := m.lastError
	lastErrAt := m.lastErrorAt
	m.errorMu.RUnlock()

	return CacheStats{
		Hits:          hits,
		Misses:        misses,
		Sets:          m.sets.Load(),
		Deletes:       m.deletes.Load(),
		Errors:        m.errors.Load(),
		HitRate:       hitRate,
		Size:          m.size.Load(),
		MemoryUsage:   m.memUsage.Load(),
		AvgGetLatency: avgGetLat,
		AvgSetLatency: avgSetLat,
		LastError:     lastErr,
		LastErrorAt:   lastErrAt,
	}
}

// Reset resets all statistics to zero.
func (m *metricsCollector) Reset() {
	m.hits.Store(0)
	m.misses.Store(0)
	m.sets.Store(0)
	m.deletes.Store(0)
	m.errors.Store(0)

	func() {
		m.mu.Lock()
		defer m.mu.Unlock()
		m.getLatencies = m.getLatencies[:0]
		m.setLatencies = m.setLatencies[:0]
	}()

	func() {
		m.errorMu.Lock()
		defer m.errorMu.Unlock()
		m.lastError = nil
		m.lastErrorAt = time.Time{}
	}()
}

// noopMetrics is a no-op metrics collector for when metrics are disabled.
type noopMetrics struct{}

func (m *noopMetrics) RecordHit(latency time.Duration)  {}
func (m *noopMetrics) RecordMiss(latency time.Duration) {}
func (m *noopMetrics) RecordSet(latency time.Duration)  {}
func (m *noopMetrics) RecordDelete()                    {}
func (m *noopMetrics) RecordError(err error)            {}
func (m *noopMetrics) UpdateSize(size int64)            {}
func (m *noopMetrics) UpdateMemoryUsage(bytes int64)    {}
func (m *noopMetrics) Stats() CacheStats                { return CacheStats{} }
func (m *noopMetrics) Reset()                           {}

// MetricsRecorder interface for both real and noop metrics.
// This interface is exported for use by cache implementations.
type MetricsRecorder interface {
	RecordHit(latency time.Duration)
	RecordMiss(latency time.Duration)
	RecordSet(latency time.Duration)
	RecordDelete()
	RecordError(err error)
	UpdateSize(size int64)
	UpdateMemoryUsage(bytes int64)
	Stats() CacheStats
	Reset()
}

// Ensure implementations satisfy the interface.
var (
	_ MetricsRecorder = (*metricsCollector)(nil)
	_ MetricsRecorder = (*noopMetrics)(nil)
)

// NewMetricsRecorder creates a metrics recorder based on whether metrics are enabled.
func NewMetricsRecorder(enabled bool) MetricsRecorder {
	if enabled {
		return newMetricsCollector()
	}
	return &noopMetrics{}
}
