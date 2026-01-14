package redis

import (
	"context"
	"encoding/json"
	"time"

	"github.com/redis/go-redis/v9"

	"microservices/pkg/cache"
)

// GetList retrieves an entire list from cache.
// The list is stored as a JSON array.
func (c *RedisCache) GetList(ctx context.Context, key string, dest interface{}) error {
	if c.closed {
		return cache.ErrClosed
	}

	if dest == nil {
		return cache.ErrNilPointer
	}

	prefixedKey := c.prefixKey(key)

	data, err := c.client.Get(ctx, prefixedKey).Bytes()
	if err != nil {
		if err == redis.Nil {
			return cache.ErrListNotFound
		}
		c.metrics.RecordError(err)
		return cache.WrapError("GetList", key, err)
	}

	if err := c.serializer.Unmarshal(data, dest); err != nil {
		c.metrics.RecordError(err)
		return cache.WrapError("GetList", key, err)
	}

	c.metrics.RecordHit(0)
	return nil
}

// SetList stores an entire list in cache.
// The list is stored as a JSON array.
func (c *RedisCache) SetList(ctx context.Context, key string, list interface{}, ttl time.Duration) error {
	if c.closed {
		return cache.ErrClosed
	}

	prefixedKey := c.prefixKey(key)

	data, err := c.serializer.Marshal(list)
	if err != nil {
		c.metrics.RecordError(err)
		return cache.WrapError("SetList", key, err)
	}

	if err := c.client.Set(ctx, prefixedKey, data, ttl).Err(); err != nil {
		c.metrics.RecordError(err)
		return cache.WrapError("SetList", key, err)
	}

	c.metrics.RecordSet(0)
	return nil
}

// AppendToList adds an item to the end of a cached list.
// Creates the list if it doesn't exist.
func (c *RedisCache) AppendToList(ctx context.Context, key string, item interface{}, ttl time.Duration) error {
	if c.closed {
		return cache.ErrClosed
	}

	prefixedKey := c.prefixKey(key)

	// Use WATCH/MULTI/EXEC for optimistic locking
	err := c.client.Watch(ctx, func(tx *redis.Tx) error {
		// Get current list
		var currentList []json.RawMessage
		data, err := tx.Get(ctx, prefixedKey).Bytes()
		if err != nil && err != redis.Nil {
			return err
		}

		if err != redis.Nil {
			if err := json.Unmarshal(data, &currentList); err != nil {
				return err
			}
		}

		// Serialize new item
		itemData, err := c.serializer.Marshal(item)
		if err != nil {
			return err
		}

		// Append item
		currentList = append(currentList, itemData)

		// Serialize updated list
		newData, err := json.Marshal(currentList)
		if err != nil {
			return err
		}

		// Execute in transaction
		_, err = tx.TxPipelined(ctx, func(pipe redis.Pipeliner) error {
			pipe.Set(ctx, prefixedKey, newData, ttl)
			return nil
		})
		return err
	}, prefixedKey)

	if err != nil {
		c.metrics.RecordError(err)
		return cache.WrapError("AppendToList", key, err)
	}

	c.metrics.RecordSet(0)
	return nil
}

// PrependToList adds an item to the beginning of a cached list.
// Creates the list if it doesn't exist.
func (c *RedisCache) PrependToList(ctx context.Context, key string, item interface{}, ttl time.Duration) error {
	if c.closed {
		return cache.ErrClosed
	}

	prefixedKey := c.prefixKey(key)

	err := c.client.Watch(ctx, func(tx *redis.Tx) error {
		// Get current list
		var currentList []json.RawMessage
		data, err := tx.Get(ctx, prefixedKey).Bytes()
		if err != nil && err != redis.Nil {
			return err
		}

		if err != redis.Nil {
			if err := json.Unmarshal(data, &currentList); err != nil {
				return err
			}
		}

		// Serialize new item
		itemData, err := c.serializer.Marshal(item)
		if err != nil {
			return err
		}

		// Prepend item
		currentList = append([]json.RawMessage{itemData}, currentList...)

		// Serialize updated list
		newData, err := json.Marshal(currentList)
		if err != nil {
			return err
		}

		// Execute in transaction
		_, err = tx.TxPipelined(ctx, func(pipe redis.Pipeliner) error {
			pipe.Set(ctx, prefixedKey, newData, ttl)
			return nil
		})
		return err
	}, prefixedKey)

	if err != nil {
		c.metrics.RecordError(err)
		return cache.WrapError("PrependToList", key, err)
	}

	c.metrics.RecordSet(0)
	return nil
}

// RemoveFromList removes items matching the predicate from the list.
func (c *RedisCache) RemoveFromList(ctx context.Context, key string, predicate func(item interface{}) bool, ttl time.Duration) error {
	if c.closed {
		return cache.ErrClosed
	}

	prefixedKey := c.prefixKey(key)

	err := c.client.Watch(ctx, func(tx *redis.Tx) error {
		// Get current list
		data, err := tx.Get(ctx, prefixedKey).Bytes()
		if err != nil {
			if err == redis.Nil {
				return cache.ErrListNotFound
			}
			return err
		}

		var currentList []json.RawMessage
		if err := json.Unmarshal(data, &currentList); err != nil {
			return err
		}

		// Filter list
		filtered := make([]json.RawMessage, 0, len(currentList))
		for _, rawItem := range currentList {
			// Unmarshal to interface{} for predicate
			var item interface{}
			if err := json.Unmarshal(rawItem, &item); err != nil {
				continue
			}

			if !predicate(item) {
				filtered = append(filtered, rawItem)
			}
		}

		// Serialize filtered list
		newData, err := json.Marshal(filtered)
		if err != nil {
			return err
		}

		// Execute in transaction
		_, err = tx.TxPipelined(ctx, func(pipe redis.Pipeliner) error {
			pipe.Set(ctx, prefixedKey, newData, ttl)
			return nil
		})
		return err
	}, prefixedKey)

	if err != nil {
		c.metrics.RecordError(err)
		return cache.WrapError("RemoveFromList", key, err)
	}

	c.metrics.RecordSet(0)
	return nil
}

// UpdateInList updates items matching the predicate in the list.
func (c *RedisCache) UpdateInList(ctx context.Context, key string, predicate func(item interface{}) bool, updater func(item interface{}) interface{}, ttl time.Duration) error {
	if c.closed {
		return cache.ErrClosed
	}

	prefixedKey := c.prefixKey(key)

	err := c.client.Watch(ctx, func(tx *redis.Tx) error {
		// Get current list
		data, err := tx.Get(ctx, prefixedKey).Bytes()
		if err != nil {
			if err == redis.Nil {
				return cache.ErrListNotFound
			}
			return err
		}

		var currentList []json.RawMessage
		if err := json.Unmarshal(data, &currentList); err != nil {
			return err
		}

		// Update matching items
		updated := make([]json.RawMessage, len(currentList))
		for i, rawItem := range currentList {
			// Unmarshal to interface{} for predicate
			var item interface{}
			if err := json.Unmarshal(rawItem, &item); err != nil {
				updated[i] = rawItem
				continue
			}

			if predicate(item) {
				// Apply updater
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

		// Serialize updated list
		newData, err := json.Marshal(updated)
		if err != nil {
			return err
		}

		// Execute in transaction
		_, err = tx.TxPipelined(ctx, func(pipe redis.Pipeliner) error {
			pipe.Set(ctx, prefixedKey, newData, ttl)
			return nil
		})
		return err
	}, prefixedKey)

	if err != nil {
		c.metrics.RecordError(err)
		return cache.WrapError("UpdateInList", key, err)
	}

	c.metrics.RecordSet(0)
	return nil
}

// GetListSize returns the number of items in the cached list.
// Returns 0 if the list doesn't exist.
func (c *RedisCache) GetListSize(ctx context.Context, key string) (int, error) {
	if c.closed {
		return 0, cache.ErrClosed
	}

	prefixedKey := c.prefixKey(key)

	data, err := c.client.Get(ctx, prefixedKey).Bytes()
	if err != nil {
		if err == redis.Nil {
			return 0, nil
		}
		c.metrics.RecordError(err)
		return 0, cache.WrapError("GetListSize", key, err)
	}

	var list []json.RawMessage
	if err := json.Unmarshal(data, &list); err != nil {
		c.metrics.RecordError(err)
		return 0, cache.WrapError("GetListSize", key, err)
	}

	return len(list), nil
}

// DeleteByPattern deletes keys matching the pattern.
// Returns the number of keys deleted.
func (c *RedisCache) DeleteByPattern(ctx context.Context, pattern string) (int64, error) {
	if c.closed {
		return 0, cache.ErrClosed
	}

	prefixedPattern := c.prefixKey(pattern)

	var deleted int64
	var cursor uint64

	for {
		keys, nextCursor, err := c.client.Scan(ctx, cursor, prefixedPattern, 100).Result()
		if err != nil {
			c.metrics.RecordError(err)
			return deleted, cache.WrapError("DeleteByPattern", pattern, err)
		}

		if len(keys) > 0 {
			result, err := c.client.Del(ctx, keys...).Result()
			if err != nil {
				c.metrics.RecordError(err)
				return deleted, cache.WrapError("DeleteByPattern", pattern, err)
			}
			deleted += result
		}

		cursor = nextCursor
		if cursor == 0 {
			break
		}
	}

	return deleted, nil
}

// Keys returns all keys matching the pattern.
func (c *RedisCache) Keys(ctx context.Context, pattern string) ([]string, error) {
	if c.closed {
		return nil, cache.ErrClosed
	}

	prefixedPattern := c.prefixKey(pattern)

	keys, err := c.client.Keys(ctx, prefixedPattern).Result()
	if err != nil {
		c.metrics.RecordError(err)
		return nil, cache.WrapError("Keys", pattern, err)
	}

	// Remove prefix from keys
	if c.prefix != "" {
		prefixLen := len(c.prefix) + 1 // +1 for ":"
		for i, key := range keys {
			if len(key) > prefixLen {
				keys[i] = key[prefixLen:]
			}
		}
	}

	return keys, nil
}

// Scan iterates over keys matching the pattern with cursor-based pagination.
func (c *RedisCache) Scan(ctx context.Context, pattern string, cursor uint64, count int64) ([]string, uint64, error) {
	if c.closed {
		return nil, 0, cache.ErrClosed
	}

	prefixedPattern := c.prefixKey(pattern)

	keys, nextCursor, err := c.client.Scan(ctx, cursor, prefixedPattern, count).Result()
	if err != nil {
		c.metrics.RecordError(err)
		return nil, 0, cache.WrapError("Scan", pattern, err)
	}

	// Remove prefix from keys
	if c.prefix != "" {
		prefixLen := len(c.prefix) + 1 // +1 for ":"
		for i, key := range keys {
			if len(key) > prefixLen {
				keys[i] = key[prefixLen:]
			}
		}
	}

	return keys, nextCursor, nil
}
