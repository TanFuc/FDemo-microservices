// Package memory provides an in-memory cache implementation.
package memory

import (
	"container/list"
	"context"
	"encoding/json"
	"log/slog"
	"sync"
	"time"

	"microservices/pkg/cache"
)

// Ensure MemoryCache implements the required interfaces.
var (
	_ cache.Cache        = (*MemoryCache)(nil)
	_ cache.BatchCache   = (*MemoryCache)(nil)
	_ cache.ListCache    = (*MemoryCache)(nil)
	_ cache.MetricsCache = (*MemoryCache)(nil)
)

// cacheItem represents an item in the cache.
type cacheItem struct {
	data      []byte
	expiresAt time.Time
	size      int64
}

// isExpired returns true if the item has expired.
func (i *cacheItem) isExpired() bool {
	if i.expiresAt.IsZero() {
		return false
	}
	return time.Now().After(i.expiresAt)
}

// MemoryCache implements cache.Cache using in-memory storage.
type MemoryCache struct {
	items       map[string]*cacheItem
	mu          sync.RWMutex
	serializer  cache.Serializer
	logger      *slog.Logger
	metrics     cache.MetricsRecorder
	maxSize     int64
	maxItems    int
	currentSize int64
	closed      bool
	done        chan struct{}
}

// Options contains configuration options for MemoryCache.
type Options struct {
	Serializer cache.Serializer
	Logger     *slog.Logger
	Metrics    bool
}

// DefaultOptions returns the default options.
func DefaultOptions() *Options {
	return &Options{
		Serializer: cache.DefaultSerializer(),
		Logger:     slog.Default(),
		Metrics:    false,
	}
}

// New creates a new MemoryCache with the given configuration.
func New(cfg *cache.MemoryConfig, opts *Options) (*MemoryCache, error) {
	if cfg == nil {
		cfg = cache.DefaultMemoryConfig()
	}

	cfg = cfg.WithDefaults()

	if opts == nil {
		opts = DefaultOptions()
	}

	logger := opts.Logger
	if logger == nil {
		logger = slog.Default()
	}

	serializer := opts.Serializer
	if serializer == nil {
		serializer = cache.DefaultSerializer()
	}

	c := &MemoryCache{
		items:      make(map[string]*cacheItem),
		serializer: serializer,
		logger:     logger,
		metrics:    cache.NewMetricsRecorder(opts.Metrics),
		maxSize:    cfg.MaxSize,
		maxItems:   cfg.MaxItems,
		done:       make(chan struct{}),
	}

	// Start cleanup goroutine
	if cfg.CleanupInterval > 0 {
		go c.cleanupLoop(cfg.CleanupInterval)
	}

	return c, nil
}

// cleanupLoop periodically removes expired items.
func (c *MemoryCache) cleanupLoop(interval time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			c.cleanup()
		case <-c.done:
			return
		}
	}
}

// cleanup removes expired items from the cache.
func (c *MemoryCache) cleanup() {
	c.mu.Lock()
	defer c.mu.Unlock()

	now := time.Now()
	for key, item := range c.items {
		if !item.expiresAt.IsZero() && now.After(item.expiresAt) {
			c.currentSize -= item.size
			delete(c.items, key)
		}
	}

	c.metrics.UpdateSize(int64(len(c.items)))
	c.metrics.UpdateMemoryUsage(c.currentSize)
}

// evict removes items to make room for new items (simple LRU-like eviction).
func (c *MemoryCache) evict() {
	// Simple eviction: remove oldest expired items first, then oldest items
	// For true LRU, use the LRU cache implementation

	// First, remove expired items
	now := time.Now()
	for key, item := range c.items {
		if !item.expiresAt.IsZero() && now.After(item.expiresAt) {
			c.currentSize -= item.size
			delete(c.items, key)
		}
	}

	// If still over capacity, remove items until under limit
	if c.maxItems > 0 && len(c.items) >= c.maxItems {
		// Remove ~10% of items
		toRemove := len(c.items) / 10
		if toRemove == 0 {
			toRemove = 1
		}

		count := 0
		for key, item := range c.items {
			c.currentSize -= item.size
			delete(c.items, key)
			count++
			if count >= toRemove {
				break
			}
		}
	}

	c.metrics.UpdateSize(int64(len(c.items)))
	c.metrics.UpdateMemoryUsage(c.currentSize)
}

// Get retrieves a value from cache and unmarshals it into dest.
func (c *MemoryCache) Get(ctx context.Context, key string, dest interface{}) error {
	if c.closed {
		return cache.ErrClosed
	}

	if dest == nil {
		return cache.ErrNilPointer
	}

	start := time.Now()

	c.mu.RLock()
	item, exists := c.items[key]
	c.mu.RUnlock()

	latency := time.Since(start)

	if !exists {
		c.metrics.RecordMiss(latency)
		return cache.ErrCacheMiss
	}

	if item.isExpired() {
		c.metrics.RecordMiss(latency)
		// Remove expired item
		c.mu.Lock()
		if item, ok := c.items[key]; ok && item.isExpired() {
			c.currentSize -= item.size
			delete(c.items, key)
		}
		c.mu.Unlock()
		return cache.ErrCacheMiss
	}

	if err := c.serializer.Unmarshal(item.data, dest); err != nil {
		c.metrics.RecordError(err)
		return cache.WrapError("Get", key, err)
	}

	c.metrics.RecordHit(latency)
	return nil
}

// Set stores a value in cache with the specified TTL.
func (c *MemoryCache) Set(ctx context.Context, key string, value interface{}, ttl time.Duration) error {
	if c.closed {
		return cache.ErrClosed
	}

	start := time.Now()

	data, err := c.serializer.Marshal(value)
	if err != nil {
		c.metrics.RecordError(err)
		return cache.WrapError("Set", key, err)
	}

	item := &cacheItem{
		data: data,
		size: int64(len(data)),
	}

	if ttl > 0 {
		item.expiresAt = time.Now().Add(ttl)
	}

	c.mu.Lock()

	// Check if we need to evict
	if c.maxItems > 0 && len(c.items) >= c.maxItems {
		c.evict()
	}

	// Check size limit
	if c.maxSize > 0 && c.currentSize+item.size > c.maxSize {
		c.evict()
	}

	// Update or add item
	if existing, ok := c.items[key]; ok {
		c.currentSize -= existing.size
	}

	c.items[key] = item
	c.currentSize += item.size

	c.mu.Unlock()

	c.metrics.RecordSet(time.Since(start))
	c.metrics.UpdateSize(int64(len(c.items)))
	c.metrics.UpdateMemoryUsage(c.currentSize)

	return nil
}

// Delete removes a key from cache.
func (c *MemoryCache) Delete(ctx context.Context, key string) error {
	if c.closed {
		return cache.ErrClosed
	}

	c.mu.Lock()
	if item, ok := c.items[key]; ok {
		c.currentSize -= item.size
		delete(c.items, key)
	}
	c.mu.Unlock()

	c.metrics.RecordDelete()
	c.metrics.UpdateSize(int64(len(c.items)))
	c.metrics.UpdateMemoryUsage(c.currentSize)

	return nil
}

// Exists checks if a key exists in cache.
func (c *MemoryCache) Exists(ctx context.Context, key string) (bool, error) {
	if c.closed {
		return false, cache.ErrClosed
	}

	c.mu.RLock()
	item, exists := c.items[key]
	c.mu.RUnlock()

	if !exists {
		return false, nil
	}

	if item.isExpired() {
		// Remove expired item
		c.mu.Lock()
		if item, ok := c.items[key]; ok && item.isExpired() {
			c.currentSize -= item.size
			delete(c.items, key)
		}
		c.mu.Unlock()
		return false, nil
	}

	return true, nil
}

// Close closes the cache and stops the cleanup goroutine.
func (c *MemoryCache) Close() error {
	if c.closed {
		return nil
	}
	c.closed = true
	close(c.done)
	return nil
}

// Ping always returns nil for memory cache.
func (c *MemoryCache) Ping(ctx context.Context) error {
	if c.closed {
		return cache.ErrClosed
	}
	return nil
}

// Stats returns the current cache statistics.
func (c *MemoryCache) Stats() cache.CacheStats {
	stats := c.metrics.Stats()

	c.mu.RLock()
	stats.Size = int64(len(c.items))
	stats.MemoryUsage = c.currentSize
	c.mu.RUnlock()

	return stats
}

// ResetStats resets all statistics.
func (c *MemoryCache) ResetStats() {
	c.metrics.Reset()
}

// MGet retrieves multiple keys at once.
func (c *MemoryCache) MGet(ctx context.Context, keys []string) (map[string][]byte, error) {
	if c.closed {
		return nil, cache.ErrClosed
	}

	results := make(map[string][]byte, len(keys))

	c.mu.RLock()
	defer c.mu.RUnlock()

	now := time.Now()
	for _, key := range keys {
		item, exists := c.items[key]
		if !exists {
			continue
		}

		if !item.expiresAt.IsZero() && now.After(item.expiresAt) {
			continue
		}

		results[key] = item.data
	}

	return results, nil
}

// MSet sets multiple key-value pairs with the same TTL.
func (c *MemoryCache) MSet(ctx context.Context, items map[string]interface{}, ttl time.Duration) error {
	if c.closed {
		return cache.ErrClosed
	}

	for key, value := range items {
		if err := c.Set(ctx, key, value, ttl); err != nil {
			return err
		}
	}

	return nil
}

// MDelete deletes multiple keys.
func (c *MemoryCache) MDelete(ctx context.Context, keys []string) (int64, error) {
	if c.closed {
		return 0, cache.ErrClosed
	}

	var deleted int64

	c.mu.Lock()
	defer c.mu.Unlock()

	for _, key := range keys {
		if item, ok := c.items[key]; ok {
			c.currentSize -= item.size
			delete(c.items, key)
			deleted++
			c.metrics.RecordDelete()
		}
	}

	c.metrics.UpdateSize(int64(len(c.items)))
	c.metrics.UpdateMemoryUsage(c.currentSize)

	return deleted, nil
}

// GetList retrieves an entire list from cache.
func (c *MemoryCache) GetList(ctx context.Context, key string, dest interface{}) error {
	return c.Get(ctx, key, dest)
}

// SetList stores an entire list in cache.
func (c *MemoryCache) SetList(ctx context.Context, key string, list interface{}, ttl time.Duration) error {
	return c.Set(ctx, key, list, ttl)
}

// AppendToList adds an item to the end of a cached list.
func (c *MemoryCache) AppendToList(ctx context.Context, key string, item interface{}, ttl time.Duration) error {
	if c.closed {
		return cache.ErrClosed
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	var currentList []json.RawMessage

	if existing, ok := c.items[key]; ok && !existing.isExpired() {
		if err := json.Unmarshal(existing.data, &currentList); err != nil {
			return cache.WrapError("AppendToList", key, err)
		}
	}

	itemData, err := c.serializer.Marshal(item)
	if err != nil {
		return cache.WrapError("AppendToList", key, err)
	}

	currentList = append(currentList, itemData)

	newData, err := json.Marshal(currentList)
	if err != nil {
		return cache.WrapError("AppendToList", key, err)
	}

	newItem := &cacheItem{
		data: newData,
		size: int64(len(newData)),
	}
	if ttl > 0 {
		newItem.expiresAt = time.Now().Add(ttl)
	}

	if existing, ok := c.items[key]; ok {
		c.currentSize -= existing.size
	}

	c.items[key] = newItem
	c.currentSize += newItem.size

	c.metrics.RecordSet(0)
	return nil
}

// PrependToList adds an item to the beginning of a cached list.
func (c *MemoryCache) PrependToList(ctx context.Context, key string, item interface{}, ttl time.Duration) error {
	if c.closed {
		return cache.ErrClosed
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	var currentList []json.RawMessage

	if existing, ok := c.items[key]; ok && !existing.isExpired() {
		if err := json.Unmarshal(existing.data, &currentList); err != nil {
			return cache.WrapError("PrependToList", key, err)
		}
	}

	itemData, err := c.serializer.Marshal(item)
	if err != nil {
		return cache.WrapError("PrependToList", key, err)
	}

	currentList = append([]json.RawMessage{itemData}, currentList...)

	newData, err := json.Marshal(currentList)
	if err != nil {
		return cache.WrapError("PrependToList", key, err)
	}

	newItem := &cacheItem{
		data: newData,
		size: int64(len(newData)),
	}
	if ttl > 0 {
		newItem.expiresAt = time.Now().Add(ttl)
	}

	if existing, ok := c.items[key]; ok {
		c.currentSize -= existing.size
	}

	c.items[key] = newItem
	c.currentSize += newItem.size

	c.metrics.RecordSet(0)
	return nil
}

// RemoveFromList removes items matching the predicate from the list.
func (c *MemoryCache) RemoveFromList(ctx context.Context, key string, predicate func(item interface{}) bool, ttl time.Duration) error {
	if c.closed {
		return cache.ErrClosed
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	existing, ok := c.items[key]
	if !ok || existing.isExpired() {
		return cache.ErrListNotFound
	}

	var currentList []json.RawMessage
	if err := json.Unmarshal(existing.data, &currentList); err != nil {
		return cache.WrapError("RemoveFromList", key, err)
	}

	filtered := make([]json.RawMessage, 0, len(currentList))
	for _, rawItem := range currentList {
		var item interface{}
		if err := json.Unmarshal(rawItem, &item); err != nil {
			continue
		}

		if !predicate(item) {
			filtered = append(filtered, rawItem)
		}
	}

	newData, err := json.Marshal(filtered)
	if err != nil {
		return cache.WrapError("RemoveFromList", key, err)
	}

	c.currentSize -= existing.size

	newItem := &cacheItem{
		data: newData,
		size: int64(len(newData)),
	}
	if ttl > 0 {
		newItem.expiresAt = time.Now().Add(ttl)
	}

	c.items[key] = newItem
	c.currentSize += newItem.size

	c.metrics.RecordSet(0)
	return nil
}

// UpdateInList updates items matching the predicate in the list.
func (c *MemoryCache) UpdateInList(ctx context.Context, key string, predicate func(item interface{}) bool, updater func(item interface{}) interface{}, ttl time.Duration) error {
	if c.closed {
		return cache.ErrClosed
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	existing, ok := c.items[key]
	if !ok || existing.isExpired() {
		return cache.ErrListNotFound
	}

	var currentList []json.RawMessage
	if err := json.Unmarshal(existing.data, &currentList); err != nil {
		return cache.WrapError("UpdateInList", key, err)
	}

	updated := make([]json.RawMessage, len(currentList))
	for i, rawItem := range currentList {
		var item interface{}
		if err := json.Unmarshal(rawItem, &item); err != nil {
			updated[i] = rawItem
			continue
		}

		if predicate(item) {
			newItem := updater(item)
			newRaw, err := json.Marshal(newItem)
			if err != nil {
				updated[i] = rawItem
				continue
			}
			updated[i] = newRaw
		} else {
			updated[i] = rawItem
		}
	}

	newData, err := json.Marshal(updated)
	if err != nil {
		return cache.WrapError("UpdateInList", key, err)
	}

	c.currentSize -= existing.size

	newItem := &cacheItem{
		data: newData,
		size: int64(len(newData)),
	}
	if ttl > 0 {
		newItem.expiresAt = time.Now().Add(ttl)
	}

	c.items[key] = newItem
	c.currentSize += newItem.size

	c.metrics.RecordSet(0)
	return nil
}

// GetListSize returns the number of items in the cached list.
func (c *MemoryCache) GetListSize(ctx context.Context, key string) (int, error) {
	if c.closed {
		return 0, cache.ErrClosed
	}

	c.mu.RLock()
	item, exists := c.items[key]
	c.mu.RUnlock()

	if !exists || item.isExpired() {
		return 0, nil
	}

	var list []json.RawMessage
	if err := json.Unmarshal(item.data, &list); err != nil {
		return 0, cache.WrapError("GetListSize", key, err)
	}

	return len(list), nil
}

// Clear removes all items from the cache.
func (c *MemoryCache) Clear() {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.items = make(map[string]*cacheItem)
	c.currentSize = 0

	c.metrics.UpdateSize(0)
	c.metrics.UpdateMemoryUsage(0)
}

// Size returns the current number of items in the cache.
func (c *MemoryCache) Size() int {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return len(c.items)
}

// MemoryUsage returns the current memory usage in bytes.
func (c *MemoryCache) MemoryUsage() int64 {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.currentSize
}

// =============================================================================
// LRU Cache Implementation
// =============================================================================

// Ensure LRUCache implements the required interfaces.
var (
	_ cache.Cache        = (*LRUCache)(nil)
	_ cache.MetricsCache = (*LRUCache)(nil)
)

// lruItem represents an item in the LRU cache.
type lruItem struct {
	key       string
	data      []byte
	expiresAt time.Time
	size      int64
}

// isExpired returns true if the item has expired.
func (i *lruItem) isExpired() bool {
	if i.expiresAt.IsZero() {
		return false
	}
	return time.Now().After(i.expiresAt)
}

// LRUCache implements cache.Cache with LRU eviction policy.
type LRUCache struct {
	capacity    int
	maxSize     int64
	items       map[string]*list.Element
	order       *list.List // Front = most recently used, Back = least recently used
	mu          sync.RWMutex
	serializer  cache.Serializer
	logger      *slog.Logger
	metrics     cache.MetricsRecorder
	currentSize int64
	closed      bool
	done        chan struct{}
}

// LRUOptions contains configuration options for LRUCache.
type LRUOptions struct {
	Capacity        int
	MaxSize         int64
	Serializer      cache.Serializer
	Logger          *slog.Logger
	Metrics         bool
	CleanupInterval time.Duration
}

// DefaultLRUOptions returns the default LRU options.
func DefaultLRUOptions() *LRUOptions {
	return &LRUOptions{
		Capacity:        10000,
		MaxSize:         100 * 1024 * 1024, // 100MB
		Serializer:      cache.DefaultSerializer(),
		Logger:          slog.Default(),
		Metrics:         false,
		CleanupInterval: 1 * time.Minute,
	}
}

// NewLRU creates a new LRU cache.
func NewLRU(opts *LRUOptions) (*LRUCache, error) {
	if opts == nil {
		opts = DefaultLRUOptions()
	}

	if opts.Capacity <= 0 {
		opts.Capacity = 10000
	}

	logger := opts.Logger
	if logger == nil {
		logger = slog.Default()
	}

	serializer := opts.Serializer
	if serializer == nil {
		serializer = cache.DefaultSerializer()
	}

	c := &LRUCache{
		capacity:   opts.Capacity,
		maxSize:    opts.MaxSize,
		items:      make(map[string]*list.Element),
		order:      list.New(),
		serializer: serializer,
		logger:     logger,
		metrics:    cache.NewMetricsRecorder(opts.Metrics),
		done:       make(chan struct{}),
	}

	// Start cleanup goroutine for expired items
	if opts.CleanupInterval > 0 {
		go c.cleanupLoop(opts.CleanupInterval)
	}

	return c, nil
}

// cleanupLoop periodically removes expired items.
func (c *LRUCache) cleanupLoop(interval time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			c.cleanup()
		case <-c.done:
			return
		}
	}
}

// cleanup removes expired items.
func (c *LRUCache) cleanup() {
	c.mu.Lock()
	defer c.mu.Unlock()

	now := time.Now()
	for key, elem := range c.items {
		item := elem.Value.(*lruItem)
		if !item.expiresAt.IsZero() && now.After(item.expiresAt) {
			c.removeElement(elem)
			delete(c.items, key)
		}
	}

	c.metrics.UpdateSize(int64(len(c.items)))
	c.metrics.UpdateMemoryUsage(c.currentSize)
}

// removeElement removes an element from the list and updates size.
func (c *LRUCache) removeElement(elem *list.Element) {
	item := elem.Value.(*lruItem)
	c.currentSize -= item.size
	c.order.Remove(elem)
}

// evict removes the least recently used item.
func (c *LRUCache) evict() {
	elem := c.order.Back()
	if elem == nil {
		return
	}

	item := elem.Value.(*lruItem)
	delete(c.items, item.key)
	c.removeElement(elem)
}

// Get retrieves a value from cache and unmarshals it into dest.
func (c *LRUCache) Get(ctx context.Context, key string, dest interface{}) error {
	if c.closed {
		return cache.ErrClosed
	}

	if dest == nil {
		return cache.ErrNilPointer
	}

	start := time.Now()

	c.mu.Lock()
	elem, exists := c.items[key]
	if !exists {
		c.mu.Unlock()
		c.metrics.RecordMiss(time.Since(start))
		return cache.ErrCacheMiss
	}

	item := elem.Value.(*lruItem)

	if item.isExpired() {
		c.removeElement(elem)
		delete(c.items, key)
		c.mu.Unlock()
		c.metrics.RecordMiss(time.Since(start))
		return cache.ErrCacheMiss
	}

	// Move to front (most recently used)
	c.order.MoveToFront(elem)
	data := item.data
	c.mu.Unlock()

	if err := c.serializer.Unmarshal(data, dest); err != nil {
		c.metrics.RecordError(err)
		return cache.WrapError("Get", key, err)
	}

	c.metrics.RecordHit(time.Since(start))
	return nil
}

// Set stores a value in cache with the specified TTL.
func (c *LRUCache) Set(ctx context.Context, key string, value interface{}, ttl time.Duration) error {
	if c.closed {
		return cache.ErrClosed
	}

	start := time.Now()

	data, err := c.serializer.Marshal(value)
	if err != nil {
		c.metrics.RecordError(err)
		return cache.WrapError("Set", key, err)
	}

	item := &lruItem{
		key:  key,
		data: data,
		size: int64(len(data)),
	}

	if ttl > 0 {
		item.expiresAt = time.Now().Add(ttl)
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	// Check if key already exists
	if elem, exists := c.items[key]; exists {
		// Update existing item
		oldItem := elem.Value.(*lruItem)
		c.currentSize -= oldItem.size
		elem.Value = item
		c.currentSize += item.size
		c.order.MoveToFront(elem)
	} else {
		// Evict if at capacity
		for len(c.items) >= c.capacity {
			c.evict()
		}

		// Evict if over size limit
		for c.maxSize > 0 && c.currentSize+item.size > c.maxSize && c.order.Len() > 0 {
			c.evict()
		}

		// Add new item
		elem := c.order.PushFront(item)
		c.items[key] = elem
		c.currentSize += item.size
	}

	c.metrics.RecordSet(time.Since(start))
	c.metrics.UpdateSize(int64(len(c.items)))
	c.metrics.UpdateMemoryUsage(c.currentSize)

	return nil
}

// Delete removes a key from cache.
func (c *LRUCache) Delete(ctx context.Context, key string) error {
	if c.closed {
		return cache.ErrClosed
	}

	c.mu.Lock()
	if elem, exists := c.items[key]; exists {
		c.removeElement(elem)
		delete(c.items, key)
	}
	c.mu.Unlock()

	c.metrics.RecordDelete()
	c.metrics.UpdateSize(int64(len(c.items)))
	c.metrics.UpdateMemoryUsage(c.currentSize)

	return nil
}

// Exists checks if a key exists in cache.
func (c *LRUCache) Exists(ctx context.Context, key string) (bool, error) {
	if c.closed {
		return false, cache.ErrClosed
	}

	c.mu.RLock()
	elem, exists := c.items[key]
	if !exists {
		c.mu.RUnlock()
		return false, nil
	}

	item := elem.Value.(*lruItem)
	expired := item.isExpired()
	c.mu.RUnlock()

	if expired {
		c.mu.Lock()
		if elem, ok := c.items[key]; ok {
			item := elem.Value.(*lruItem)
			if item.isExpired() {
				c.removeElement(elem)
				delete(c.items, key)
			}
		}
		c.mu.Unlock()
		return false, nil
	}

	return true, nil
}

// Close closes the cache.
func (c *LRUCache) Close() error {
	if c.closed {
		return nil
	}
	c.closed = true
	close(c.done)
	return nil
}

// Ping always returns nil for LRU cache.
func (c *LRUCache) Ping(ctx context.Context) error {
	if c.closed {
		return cache.ErrClosed
	}
	return nil
}

// Stats returns the current cache statistics.
func (c *LRUCache) Stats() cache.CacheStats {
	stats := c.metrics.Stats()

	c.mu.RLock()
	stats.Size = int64(len(c.items))
	stats.MemoryUsage = c.currentSize
	c.mu.RUnlock()

	return stats
}

// ResetStats resets all statistics.
func (c *LRUCache) ResetStats() {
	c.metrics.Reset()
}

// Len returns the current number of items in the cache.
func (c *LRUCache) Len() int {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return len(c.items)
}

// Clear removes all items from the cache.
func (c *LRUCache) Clear() {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.items = make(map[string]*list.Element)
	c.order.Init()
	c.currentSize = 0

	c.metrics.UpdateSize(0)
	c.metrics.UpdateMemoryUsage(0)
}
