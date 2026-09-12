package idempotency

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

var (
	// ErrConcurrentRequest is returned when an in-flight operation for the idempotency key is already running.
	ErrConcurrentRequest = errors.New("concurrent request in progress with same idempotency key")
	// ErrEmptyKey is returned when the supplied idempotency key is empty.
	ErrEmptyKey = errors.New("idempotency key cannot be empty")
)

// CachedResponse encapsulates the cached output of an idempotent operation.
type CachedResponse struct {
	StatusCode int                 `json:"status_code"`
	Headers    map[string][]string `json:"headers,omitempty"`
	Body       []byte              `json:"body"`
	CreatedAt  time.Time           `json:"created_at"`
}

// Manager defines the interface for distributed idempotency handling.
type Manager interface {
	// TryAcquire attempts to claim the idempotency key.
	// Returns (cachedResponse, acquired, error).
	// If acquired == true, the caller is granted exclusive rights to perform the operation.
	// If acquired == false and cachedResponse != nil, the previous result is returned.
	// If acquired == false and cachedResponse == nil, a concurrent request is currently in progress.
	TryAcquire(ctx context.Context, key string, inFlightTTL time.Duration) (*CachedResponse, bool, error)

	// SaveResult stores the final output for the idempotency key and releases the in-flight lock.
	SaveResult(ctx context.Context, key string, resp CachedResponse, retentionTTL time.Duration) error

	// Release cancels the lock without saving a cached result (used when operation fails and can be retried).
	Release(ctx context.Context, key string) error
}

type redisManager struct {
	client    redis.UniversalClient
	keyPrefix string
}

// Config provides configuration for the Redis Idempotency Manager.
type Config struct {
	Client    redis.UniversalClient
	KeyPrefix string
}

// NewRedisManager creates a new Redis-backed idempotency manager.
func NewRedisManager(cfg Config) Manager {
	prefix := cfg.KeyPrefix
	if prefix == "" {
		prefix = "nexus:idempotency:"
	}
	return &redisManager{
		client:    cfg.Client,
		keyPrefix: prefix,
	}
}

func (m *redisManager) lockKey(key string) string {
	return fmt.Sprintf("%slock:%s", m.keyPrefix, key)
}

func (m *redisManager) dataKey(key string) string {
	return fmt.Sprintf("%sdata:%s", m.keyPrefix, key)
}

func (m *redisManager) TryAcquire(ctx context.Context, key string, inFlightTTL time.Duration) (*CachedResponse, bool, error) {
	if key == "" {
		return nil, false, ErrEmptyKey
	}

	dKey := m.dataKey(key)
	val, err := m.client.Get(ctx, dKey).Bytes()
	if err == nil {
		var cached CachedResponse
		if err := json.Unmarshal(val, &cached); err == nil {
			return &cached, false, nil
		}
	} else if err != redis.Nil {
		return nil, false, fmt.Errorf("failed checking cached result: %w", err)
	}

	lKey := m.lockKey(key)
	acquired, err := m.client.SetNX(ctx, lKey, "LOCKED", inFlightTTL).Result()
	if err != nil {
		return nil, false, fmt.Errorf("failed acquiring in-flight lock: %w", err)
	}

	if !acquired {
		val, err := m.client.Get(ctx, dKey).Bytes()
		if err == nil {
			var cached CachedResponse
			if err := json.Unmarshal(val, &cached); err == nil {
				return &cached, false, nil
			}
		}
		return nil, false, ErrConcurrentRequest
	}

	return nil, true, nil
}

func (m *redisManager) SaveResult(ctx context.Context, key string, resp CachedResponse, retentionTTL time.Duration) error {
	if key == "" {
		return ErrEmptyKey
	}
	if resp.CreatedAt.IsZero() {
		resp.CreatedAt = time.Now().UTC()
	}

	payload, err := json.Marshal(resp)
	if err != nil {
		return fmt.Errorf("failed marshaling cached response: %w", err)
	}

	pipe := m.client.Pipeline()
	dKey := m.dataKey(key)
	lKey := m.lockKey(key)

	pipe.Set(ctx, dKey, payload, retentionTTL)
	pipe.Del(ctx, lKey)

	_, err = pipe.Exec(ctx)
	if err != nil {
		return fmt.Errorf("failed saving idempotent response: %w", err)
	}

	return nil
}

func (m *redisManager) Release(ctx context.Context, key string) error {
	if key == "" {
		return ErrEmptyKey
	}
	lKey := m.lockKey(key)
	return m.client.Del(ctx, lKey).Err()
}
