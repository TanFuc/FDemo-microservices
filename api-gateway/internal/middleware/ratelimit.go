package middleware

import (
	"fmt"
	"time"

	"api-gateway/internal/config"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/limiter"
	"github.com/gofiber/storage/redis/v3"
)

func NewRedisStorage(cfg *config.RedisConfig) *redis.Storage {
	return redis.New(redis.Config{
		Host:     cfg.Host,
		Port:     cfg.Port,
		Password: cfg.Password,
		Database: cfg.DB,
		Reset:    false,
	})
}

func RateLimiter(storage fiber.Storage, cfg *config.RateLimitConfig) fiber.Handler {
	return limiter.New(limiter.Config{
		Max:        cfg.MaxRequests,
		Expiration: time.Duration(cfg.ExpirationSeconds) * time.Second,
		KeyGenerator: func(c *fiber.Ctx) string {
			userID := c.Locals("user_id")
			if userID != nil {
				return fmt.Sprintf("rate_limit:user:%v", userID)
			}
			return fmt.Sprintf("rate_limit:ip:%s", c.IP())
		},
		LimitReached: func(c *fiber.Ctx) error {
			return c.Status(fiber.StatusTooManyRequests).JSON(fiber.Map{
				"error":   "rate_limit_exceeded",
				"message": "too many requests, please try again later",
			})
		},
		Storage: storage,
	})
}

func RateLimiterByIP(storage fiber.Storage, cfg *config.RateLimitConfig) fiber.Handler {
	return limiter.New(limiter.Config{
		Max:        cfg.MaxRequests,
		Expiration: time.Duration(cfg.ExpirationSeconds) * time.Second,
		KeyGenerator: func(c *fiber.Ctx) string {
			return fmt.Sprintf("rate_limit:ip:%s", c.IP())
		},
		LimitReached: func(c *fiber.Ctx) error {
			return c.Status(fiber.StatusTooManyRequests).JSON(fiber.Map{
				"error":   "rate_limit_exceeded",
				"message": "too many requests, please try again later",
			})
		},
		Storage: storage,
	})
}
