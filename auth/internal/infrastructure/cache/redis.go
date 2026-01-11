package cache

import (
	"context"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"

	"tafu-auth/internal/config"
	"tafu-auth/pkg/logger"
)

type RedisClient struct {
	client *redis.Client
}

func NewRedisClient(cfg *config.Config) (*RedisClient, error) {
	client := redis.NewClient(&redis.Options{
		Addr:     cfg.GetRedisAddr(),
		Password: cfg.Redis.Password,
		DB:       cfg.Redis.DB,
	})

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := client.Ping(ctx).Err(); err != nil {
		return nil, fmt.Errorf("failed to connect to Redis: %w", err)
	}

	logger.Info().Msg("Connected to Redis")

	return &RedisClient{client: client}, nil
}

func (r *RedisClient) Client() *redis.Client {
	return r.client
}

func (r *RedisClient) Close() error {
	return r.client.Close()
}

func (r *RedisClient) HealthCheck(ctx context.Context) error {
	return r.client.Ping(ctx).Err()
}

// Key prefixes for different data types
const (
	PrefixUserPermissions = "identity:user:%s:permissions"
	PrefixTokenBlacklist  = "identity:blacklist:%s"
	PrefixActiveSession   = "identity:session:%s:%s"
	PrefixRateLimit       = "identity:ratelimit:%s"
)

// Permission cache operations
func (r *RedisClient) CacheUserPermissions(ctx context.Context, userID string, permissions []string, ttl time.Duration) error {
	key := fmt.Sprintf(PrefixUserPermissions, userID)
	return r.client.SAdd(ctx, key, stringsToInterfaces(permissions)...).Err()
}

func (r *RedisClient) GetUserPermissions(ctx context.Context, userID string) ([]string, error) {
	key := fmt.Sprintf(PrefixUserPermissions, userID)
	return r.client.SMembers(ctx, key).Result()
}

func (r *RedisClient) InvalidateUserPermissions(ctx context.Context, userID string) error {
	key := fmt.Sprintf(PrefixUserPermissions, userID)
	return r.client.Del(ctx, key).Err()
}

// Token blacklist operations
func (r *RedisClient) BlacklistToken(ctx context.Context, jti string, ttl time.Duration) error {
	key := fmt.Sprintf(PrefixTokenBlacklist, jti)
	return r.client.Set(ctx, key, "1", ttl).Err()
}

func (r *RedisClient) IsTokenBlacklisted(ctx context.Context, jti string) (bool, error) {
	key := fmt.Sprintf(PrefixTokenBlacklist, jti)
	result, err := r.client.Exists(ctx, key).Result()
	if err != nil {
		return false, err
	}
	return result > 0, nil
}

// Session operations
type SessionData struct {
	RefreshTokenID string `json:"refreshTokenId"`
	IPAddress      string `json:"ipAddress"`
	DeviceInfo     string `json:"deviceInfo"`
	CreatedAt      string `json:"createdAt"`
}

func (r *RedisClient) SetActiveSession(ctx context.Context, userID, deviceID string, data *SessionData, ttl time.Duration) error {
	key := fmt.Sprintf(PrefixActiveSession, userID, deviceID)
	return r.client.HSet(ctx, key, map[string]interface{}{
		"refreshTokenId": data.RefreshTokenID,
		"ipAddress":      data.IPAddress,
		"deviceInfo":     data.DeviceInfo,
		"createdAt":      data.CreatedAt,
	}).Err()
}

func (r *RedisClient) GetActiveSession(ctx context.Context, userID, deviceID string) (*SessionData, error) {
	key := fmt.Sprintf(PrefixActiveSession, userID, deviceID)
	result, err := r.client.HGetAll(ctx, key).Result()
	if err != nil {
		return nil, err
	}
	if len(result) == 0 {
		return nil, nil
	}
	return &SessionData{
		RefreshTokenID: result["refreshTokenId"],
		IPAddress:      result["ipAddress"],
		DeviceInfo:     result["deviceInfo"],
		CreatedAt:      result["createdAt"],
	}, nil
}

func (r *RedisClient) RemoveActiveSession(ctx context.Context, userID, deviceID string) error {
	key := fmt.Sprintf(PrefixActiveSession, userID, deviceID)
	return r.client.Del(ctx, key).Err()
}

func (r *RedisClient) GetUserActiveSessions(ctx context.Context, userID string) ([]string, error) {
	pattern := fmt.Sprintf(PrefixActiveSession, userID, "*")
	return r.client.Keys(ctx, pattern).Result()
}

// Rate limiting operations
func (r *RedisClient) IncrementRateLimit(ctx context.Context, key string, ttl time.Duration) (int64, error) {
	pipe := r.client.Pipeline()
	incr := pipe.Incr(ctx, key)
	pipe.Expire(ctx, key, ttl)
	_, err := pipe.Exec(ctx)
	if err != nil {
		return 0, err
	}
	return incr.Val(), nil
}

func (r *RedisClient) GetRateLimit(ctx context.Context, key string) (int64, error) {
	result, err := r.client.Get(ctx, key).Int64()
	if err == redis.Nil {
		return 0, nil
	}
	return result, err
}

// Generic operations
func (r *RedisClient) Set(ctx context.Context, key string, value interface{}, ttl time.Duration) error {
	return r.client.Set(ctx, key, value, ttl).Err()
}

func (r *RedisClient) Get(ctx context.Context, key string) (string, error) {
	return r.client.Get(ctx, key).Result()
}

func (r *RedisClient) Del(ctx context.Context, keys ...string) error {
	return r.client.Del(ctx, keys...).Err()
}

func (r *RedisClient) Exists(ctx context.Context, key string) (bool, error) {
	result, err := r.client.Exists(ctx, key).Result()
	return result > 0, err
}

func stringsToInterfaces(strs []string) []interface{} {
	result := make([]interface{}, len(strs))
	for i, s := range strs {
		result[i] = s
	}
	return result
}
