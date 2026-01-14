package cache

import (
	"context"
	"fmt"
	"strings"
	"time"
)

// KeyBuilder provides flexible cache key generation with a fluent interface.
type KeyBuilder interface {
	// Build constructs a cache key from the given parts.
	Build(parts ...string) string

	// WithPrefix adds a prefix to the key.
	WithPrefix(prefix string) KeyBuilder

	// WithNamespace adds a namespace to the key.
	WithNamespace(namespace string) KeyBuilder

	// WithID adds an ID to the key.
	WithID(id interface{}) KeyBuilder

	// WithTimestamp adds a timestamp to the key.
	WithTimestamp() KeyBuilder

	// WithVersion adds a version to the key.
	WithVersion(version string) KeyBuilder

	// ForService creates a service-specific key builder.
	ForService(serviceName string) KeyBuilder

	// ForEntity creates an entity-specific key builder.
	ForEntity(entityName string) KeyBuilder

	// ForList creates a list/collection key builder.
	ForList(entityName string) KeyBuilder

	// ForItem creates an item key builder with ID.
	ForItem(entityName string, id interface{}) KeyBuilder

	// ForSearch creates a search query key builder.
	ForSearch(entityName string, query string) KeyBuilder

	// ForSession creates a session key builder.
	ForSession(token string) KeyBuilder

	// String returns the final key as a string.
	String() string

	// Clone creates a copy of the key builder.
	Clone() KeyBuilder
}

// KeyPattern represents common cache key patterns.
type KeyPattern string

const (
	// KeyPatternItem represents an item key: "{service}:{entity}:{id}"
	KeyPatternItem KeyPattern = "{service}:{entity}:{id}"

	// KeyPatternList represents a list key: "{service}:{entity}:all"
	KeyPatternList KeyPattern = "{service}:{entity}:all"

	// KeyPatternSearch represents a search key: "{service}:{entity}:search:{query}"
	KeyPatternSearch KeyPattern = "{service}:{entity}:search:{query}"

	// KeyPatternCount represents a count key: "{service}:{entity}:count"
	KeyPatternCount KeyPattern = "{service}:{entity}:count"

	// KeyPatternSession represents a session key: "{service}:session:{token}"
	KeyPatternSession KeyPattern = "{service}:session:{token}"

	// KeyPatternTimeSeries represents a time-series key: "{service}:{entity}:{timestamp}"
	KeyPatternTimeSeries KeyPattern = "{service}:{entity}:{timestamp}"
)

// keyBuilder implements KeyBuilder.
type keyBuilder struct {
	parts     []string
	separator string
	prefix    string
	namespace string
}

// NewKeyBuilder creates a new KeyBuilder with the specified separator.
// If separator is empty, ":" is used as the default.
func NewKeyBuilder(separator string) KeyBuilder {
	if separator == "" {
		separator = ":"
	}
	return &keyBuilder{
		parts:     make([]string, 0),
		separator: separator,
	}
}

// Build constructs a cache key from the builder's parts and the given additional parts.
func (kb *keyBuilder) Build(parts ...string) string {
	allParts := make([]string, 0, len(kb.parts)+len(parts)+2)

	if kb.prefix != "" {
		allParts = append(allParts, kb.prefix)
	}
	if kb.namespace != "" {
		allParts = append(allParts, kb.namespace)
	}

	allParts = append(allParts, kb.parts...)
	allParts = append(allParts, parts...)

	// Filter out empty strings
	filtered := make([]string, 0, len(allParts))
	for _, p := range allParts {
		if p != "" {
			filtered = append(filtered, p)
		}
	}

	return strings.Join(filtered, kb.separator)
}

// String returns the final key as a string.
func (kb *keyBuilder) String() string {
	return kb.Build()
}

// Clone creates a deep copy of the key builder.
func (kb *keyBuilder) Clone() KeyBuilder {
	newParts := make([]string, len(kb.parts))
	copy(newParts, kb.parts)
	return &keyBuilder{
		parts:     newParts,
		separator: kb.separator,
		prefix:    kb.prefix,
		namespace: kb.namespace,
	}
}

// WithPrefix adds a prefix to the key.
func (kb *keyBuilder) WithPrefix(prefix string) KeyBuilder {
	newKb := kb.Clone().(*keyBuilder)
	newKb.prefix = prefix
	return newKb
}

// WithNamespace adds a namespace to the key.
func (kb *keyBuilder) WithNamespace(namespace string) KeyBuilder {
	newKb := kb.Clone().(*keyBuilder)
	newKb.namespace = namespace
	return newKb
}

// WithID adds an ID to the key.
func (kb *keyBuilder) WithID(id interface{}) KeyBuilder {
	newKb := kb.Clone().(*keyBuilder)
	newKb.parts = append(newKb.parts, fmt.Sprint(id))
	return newKb
}

// WithTimestamp adds the current timestamp to the key.
func (kb *keyBuilder) WithTimestamp() KeyBuilder {
	newKb := kb.Clone().(*keyBuilder)
	newKb.parts = append(newKb.parts, fmt.Sprint(time.Now().Unix()))
	return newKb
}

// WithVersion adds a version to the key.
func (kb *keyBuilder) WithVersion(version string) KeyBuilder {
	newKb := kb.Clone().(*keyBuilder)
	newKb.parts = append(newKb.parts, "v"+version)
	return newKb
}

// ForService creates a service-specific key builder.
func (kb *keyBuilder) ForService(serviceName string) KeyBuilder {
	newKb := kb.Clone().(*keyBuilder)
	newKb.parts = append(newKb.parts, serviceName)
	return newKb
}

// ForEntity creates an entity-specific key builder.
func (kb *keyBuilder) ForEntity(entityName string) KeyBuilder {
	newKb := kb.Clone().(*keyBuilder)
	newKb.parts = append(newKb.parts, entityName)
	return newKb
}

// ForList creates a list key: "entity:all"
func (kb *keyBuilder) ForList(entityName string) KeyBuilder {
	newKb := kb.Clone().(*keyBuilder)
	newKb.parts = append(newKb.parts, entityName, "all")
	return newKb
}

// ForItem creates an item key: "entity:id"
func (kb *keyBuilder) ForItem(entityName string, id interface{}) KeyBuilder {
	newKb := kb.Clone().(*keyBuilder)
	newKb.parts = append(newKb.parts, entityName, fmt.Sprint(id))
	return newKb
}

// ForSearch creates a search key: "entity:search:query"
func (kb *keyBuilder) ForSearch(entityName string, query string) KeyBuilder {
	newKb := kb.Clone().(*keyBuilder)
	newKb.parts = append(newKb.parts, entityName, "search", query)
	return newKb
}

// ForSession creates a session key: "session:token"
func (kb *keyBuilder) ForSession(token string) KeyBuilder {
	newKb := kb.Clone().(*keyBuilder)
	newKb.parts = append(newKb.parts, "session", token)
	return newKb
}

// BuildKey is a convenience function to build a simple key from parts.
func BuildKey(separator string, parts ...string) string {
	if separator == "" {
		separator = ":"
	}
	filtered := make([]string, 0, len(parts))
	for _, p := range parts {
		if p != "" {
			filtered = append(filtered, p)
		}
	}
	return strings.Join(filtered, separator)
}

// ItemKey creates a standard item key: "service:entity:id"
func ItemKey(service, entity string, id interface{}) string {
	return BuildKey(":", service, entity, fmt.Sprint(id))
}

// ListKey creates a standard list key: "service:entity:all"
func ListKey(service, entity string) string {
	return BuildKey(":", service, entity, "all")
}

// SearchKey creates a standard search key: "service:entity:search:query"
func SearchKey(service, entity, query string) string {
	return BuildKey(":", service, entity, "search", query)
}

// SessionKey creates a standard session key: "service:session:token"
func SessionKey(service, token string) string {
	return BuildKey(":", service, "session", token)
}

// CountKey creates a standard count key: "service:entity:count"
func CountKey(service, entity string) string {
	return BuildKey(":", service, entity, "count")
}

// PatternKey creates a pattern key for scanning: "service:entity:*"
func PatternKey(service, entity string) string {
	return BuildKey(":", service, entity, "*")
}

// crudCache implements CRUDCache interface.
type crudCache struct {
	cache       Cache
	keyBuilder  KeyBuilder
	serviceName string
}

// Ensure crudCache implements CRUDCache.
var _ CRUDCache = (*crudCache)(nil)

// NewCRUDCache creates a new CRUD cache wrapper.
func NewCRUDCache(cache Cache, keyBuilder KeyBuilder, serviceName string) CRUDCache {
	if keyBuilder == nil {
		keyBuilder = NewKeyBuilder(":")
	}
	return &crudCache{
		cache:       cache,
		keyBuilder:  keyBuilder,
		serviceName: serviceName,
	}
}

// itemKey generates a key for an individual item.
func (cc *crudCache) itemKey(entityName string, id interface{}) string {
	return cc.keyBuilder.ForService(cc.serviceName).ForItem(entityName, id).String()
}

// listKey generates a key for a list of items.
func (cc *crudCache) listKey(entityName string) string {
	return cc.keyBuilder.ForService(cc.serviceName).ForList(entityName).String()
}

// Create stores a new item in cache.
func (cc *crudCache) Create(ctx context.Context, entityName string, id interface{}, data interface{}, ttl time.Duration) error {
	key := cc.itemKey(entityName, id)
	return cc.cache.Set(ctx, key, data, ttl)
}

// Read retrieves an item from cache.
func (cc *crudCache) Read(ctx context.Context, entityName string, id interface{}, dest interface{}) error {
	key := cc.itemKey(entityName, id)
	return cc.cache.Get(ctx, key, dest)
}

// Update updates an item in cache.
func (cc *crudCache) Update(ctx context.Context, entityName string, id interface{}, data interface{}, ttl time.Duration) error {
	key := cc.itemKey(entityName, id)
	return cc.cache.Set(ctx, key, data, ttl)
}

// Delete removes an item from cache.
func (cc *crudCache) Delete(ctx context.Context, entityName string, id interface{}) error {
	key := cc.itemKey(entityName, id)
	return cc.cache.Delete(ctx, key)
}

// List retrieves all items of an entity type from the list cache.
func (cc *crudCache) List(ctx context.Context, entityName string, dest interface{}) error {
	key := cc.listKey(entityName)

	// Try to use ListCache interface if available
	if listCache, ok := cc.cache.(ListCache); ok {
		return listCache.GetList(ctx, key, dest)
	}

	// Fallback to regular Get
	return cc.cache.Get(ctx, key, dest)
}

// CreateWithListSync creates an item and appends it to the list cache.
func (cc *crudCache) CreateWithListSync(ctx context.Context, entityName string, id interface{}, data interface{}, ttl time.Duration) error {
	// 1. Store individual item
	itemKey := cc.itemKey(entityName, id)
	if err := cc.cache.Set(ctx, itemKey, data, ttl); err != nil {
		return fmt.Errorf("failed to store item: %w", err)
	}

	// 2. Update list cache
	listKey := cc.listKey(entityName)

	// Try to use ListCache interface
	if listCache, ok := cc.cache.(ListCache); ok {
		return listCache.AppendToList(ctx, listKey, data, ttl)
	}

	// Fallback: manual list management
	var currentList []interface{}
	err := cc.cache.Get(ctx, listKey, &currentList)
	if err != nil && !IsNotFound(err) {
		return fmt.Errorf("failed to get list: %w", err)
	}

	if IsNotFound(err) {
		currentList = []interface{}{}
	}

	currentList = append(currentList, data)
	return cc.cache.Set(ctx, listKey, currentList, ttl)
}

// UpdateWithListSync updates an item and updates it in the list cache.
func (cc *crudCache) UpdateWithListSync(ctx context.Context, entityName string, id interface{}, data interface{}, matchFn func(interface{}) bool, ttl time.Duration) error {
	// 1. Update individual item
	itemKey := cc.itemKey(entityName, id)
	if err := cc.cache.Set(ctx, itemKey, data, ttl); err != nil {
		return fmt.Errorf("failed to update item: %w", err)
	}

	// 2. Update in list cache
	listKey := cc.listKey(entityName)

	// Try to use ListCache interface
	if listCache, ok := cc.cache.(ListCache); ok {
		updater := func(item interface{}) interface{} {
			return data
		}
		return listCache.UpdateInList(ctx, listKey, matchFn, updater, ttl)
	}

	// Fallback: manual list management
	var currentList []interface{}
	if err := cc.cache.Get(ctx, listKey, &currentList); err != nil {
		if IsNotFound(err) {
			return nil // List doesn't exist, nothing to update
		}
		return fmt.Errorf("failed to get list: %w", err)
	}

	// Update matching items
	for i, item := range currentList {
		if matchFn(item) {
			currentList[i] = data
		}
	}

	return cc.cache.Set(ctx, listKey, currentList, ttl)
}

// DeleteWithListSync deletes an item and removes it from the list cache.
func (cc *crudCache) DeleteWithListSync(ctx context.Context, entityName string, id interface{}, matchFn func(interface{}) bool, ttl time.Duration) error {
	// 1. Delete individual item
	itemKey := cc.itemKey(entityName, id)
	if err := cc.cache.Delete(ctx, itemKey); err != nil && !IsNotFound(err) {
		return fmt.Errorf("failed to delete item: %w", err)
	}

	// 2. Remove from list cache
	listKey := cc.listKey(entityName)

	// Try to use ListCache interface
	if listCache, ok := cc.cache.(ListCache); ok {
		err := listCache.RemoveFromList(ctx, listKey, matchFn, ttl)
		if IsNotFound(err) {
			return nil // List doesn't exist, nothing to remove
		}
		return err
	}

	// Fallback: manual list management
	var currentList []interface{}
	if err := cc.cache.Get(ctx, listKey, &currentList); err != nil {
		if IsNotFound(err) {
			return nil // List doesn't exist, nothing to update
		}
		return fmt.Errorf("failed to get list: %w", err)
	}

	// Filter list
	filtered := make([]interface{}, 0, len(currentList))
	for _, item := range currentList {
		if !matchFn(item) {
			filtered = append(filtered, item)
		}
	}

	return cc.cache.Set(ctx, listKey, filtered, ttl)
}

// InvalidateItem deletes an item cache.
func (cc *crudCache) InvalidateItem(ctx context.Context, entityName string, id interface{}) error {
	return cc.Delete(ctx, entityName, id)
}

// InvalidateList deletes a list cache.
func (cc *crudCache) InvalidateList(ctx context.Context, entityName string) error {
	key := cc.listKey(entityName)
	return cc.cache.Delete(ctx, key)
}

// InvalidateAll deletes both item and list caches for an entity.
func (cc *crudCache) InvalidateAll(ctx context.Context, entityName string, id interface{}) error {
	if err := cc.InvalidateItem(ctx, entityName, id); err != nil && !IsNotFound(err) {
		return err
	}
	if err := cc.InvalidateList(ctx, entityName); err != nil && !IsNotFound(err) {
		return err
	}
	return nil
}

// GetCache returns the underlying cache.
func (cc *crudCache) GetCache() Cache {
	return cc.cache
}

// GetKeyBuilder returns the key builder.
func (cc *crudCache) GetKeyBuilder() KeyBuilder {
	return cc.keyBuilder
}

// GetServiceName returns the service name.
func (cc *crudCache) GetServiceName() string {
	return cc.serviceName
}

// CRUDCacheTyped provides type-safe CRUD operations.
type CRUDCacheTyped[T any] struct {
	crud       CRUDCache
	entityName string
	defaultTTL time.Duration
}

// NewCRUDCacheTyped creates a new typed CRUD cache.
func NewCRUDCacheTyped[T any](cache Cache, serviceName, entityName string, defaultTTL time.Duration) *CRUDCacheTyped[T] {
	return &CRUDCacheTyped[T]{
		crud:       NewCRUDCache(cache, NewKeyBuilder(":"), serviceName),
		entityName: entityName,
		defaultTTL: defaultTTL,
	}
}

// Create stores a new item.
func (c *CRUDCacheTyped[T]) Create(ctx context.Context, id interface{}, data T) error {
	return c.crud.Create(ctx, c.entityName, id, data, c.defaultTTL)
}

// Read retrieves an item.
func (c *CRUDCacheTyped[T]) Read(ctx context.Context, id interface{}) (T, error) {
	var result T
	err := c.crud.Read(ctx, c.entityName, id, &result)
	return result, err
}

// Update updates an item.
func (c *CRUDCacheTyped[T]) Update(ctx context.Context, id interface{}, data T) error {
	return c.crud.Update(ctx, c.entityName, id, data, c.defaultTTL)
}

// Delete removes an item.
func (c *CRUDCacheTyped[T]) Delete(ctx context.Context, id interface{}) error {
	return c.crud.Delete(ctx, c.entityName, id)
}

// List retrieves all items.
func (c *CRUDCacheTyped[T]) List(ctx context.Context) ([]T, error) {
	var result []T
	err := c.crud.List(ctx, c.entityName, &result)
	return result, err
}

// CreateWithSync creates an item and syncs with list cache.
func (c *CRUDCacheTyped[T]) CreateWithSync(ctx context.Context, id interface{}, data T) error {
	return c.crud.CreateWithListSync(ctx, c.entityName, id, data, c.defaultTTL)
}

// UpdateWithSync updates an item and syncs with list cache.
func (c *CRUDCacheTyped[T]) UpdateWithSync(ctx context.Context, id interface{}, data T, matchFn func(interface{}) bool) error {
	return c.crud.UpdateWithListSync(ctx, c.entityName, id, data, matchFn, c.defaultTTL)
}

// DeleteWithSync deletes an item and removes from list cache.
func (c *CRUDCacheTyped[T]) DeleteWithSync(ctx context.Context, id interface{}, matchFn func(interface{}) bool) error {
	return c.crud.DeleteWithListSync(ctx, c.entityName, id, matchFn, c.defaultTTL)
}
