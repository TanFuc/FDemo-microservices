package http

import (
	"context"
	"time"

	"github.com/gofiber/fiber/v2"
	"gorm.io/gorm"

	"microservices/auth/internal/cache"
	"microservices/auth/internal/queue"
	"microservices/auth/internal/repository/postgres"
	"microservices/auth/pkg/response"
)

type HealthStatus struct {
	Status    string            `json:"status"`
	Timestamp string            `json:"timestamp"`
	Services  map[string]string `json:"services"`
}

type HealthHandler struct {
	db         *gorm.DB
	redis      *cache.RedisClient
	natsClient *queue.NATSClient
}

func NewHealthHandler(db *gorm.DB, redis *cache.RedisClient, natsClient *queue.NATSClient) *HealthHandler {
	return &HealthHandler{
		db:         db,
		redis:      redis,
		natsClient: natsClient,
	}
}

func (h *HealthHandler) Health(c *fiber.Ctx) error {
	services := make(map[string]string)
	overallStatus := "healthy"

	if err := postgres.HealthCheck(h.db); err != nil {
		services["postgres"] = "unhealthy: " + err.Error()
		overallStatus = "unhealthy"
	} else {
		services["postgres"] = "healthy"
	}

	ctx, cancel := context.WithTimeout(c.Context(), 5*time.Second)
	defer cancel()
	if err := h.redis.HealthCheck(ctx); err != nil {
		services["redis"] = "unhealthy: " + err.Error()
		overallStatus = "unhealthy"
	} else {
		services["redis"] = "healthy"
	}

	if h.natsClient != nil {
		if err := h.natsClient.HealthCheck(); err != nil {
			services["nats"] = "unhealthy: " + err.Error()
		} else {
			services["nats"] = "healthy"
		}
	} else {
		services["nats"] = "not configured"
	}

	status := &HealthStatus{
		Status:    overallStatus,
		Timestamp: time.Now().UTC().Format(time.RFC3339),
		Services:  services,
	}

	if overallStatus == "unhealthy" {
		return c.Status(fiber.StatusServiceUnavailable).JSON(fiber.Map{
			"success":   false,
			"code":      "SERVICE_UNAVAILABLE",
			"message":   "Service is unhealthy",
			"data":      status,
			"timestamp": time.Now().UTC().Format(time.RFC3339),
			"path":      c.Path(),
		})
	}

	return response.Success(c, status)
}

func (h *HealthHandler) Liveness(c *fiber.Ctx) error {
	return c.JSON(fiber.Map{
		"status": "ok",
	})
}

func (h *HealthHandler) Readiness(c *fiber.Ctx) error {
	if err := postgres.HealthCheck(h.db); err != nil {
		return c.Status(fiber.StatusServiceUnavailable).JSON(fiber.Map{
			"status": "not ready",
			"reason": "database connection failed",
		})
	}

	ctx, cancel := context.WithTimeout(c.Context(), 5*time.Second)
	defer cancel()
	if err := h.redis.HealthCheck(ctx); err != nil {
		return c.Status(fiber.StatusServiceUnavailable).JSON(fiber.Map{
			"status": "not ready",
			"reason": "redis connection failed",
		})
	}

	return c.JSON(fiber.Map{
		"status": "ready",
	})
}
