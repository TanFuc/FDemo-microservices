package http

import (
	"context"

	"github.com/gofiber/fiber/v2"
	"github.com/jackc/pgx/v5/pgxpool"

	"microservices/pkg/response"
)

// HealthHandler handles health check HTTP requests
type HealthHandler struct {
	pool *pgxpool.Pool
}

// NewHealthHandler creates a new health handler
func NewHealthHandler(pool *pgxpool.Pool) *HealthHandler {
	return &HealthHandler{
		pool: pool,
	}
}

// Health returns basic health status
func (h *HealthHandler) Health(c *fiber.Ctx) error {
	return response.Success(c, fiber.Map{
		"status":  "healthy",
		"service": "payment-service",
	})
}

// Ready returns readiness status with dependency checks
func (h *HealthHandler) Ready(c *fiber.Ctx) error {
	if h.pool == nil {
		return c.Status(fiber.StatusServiceUnavailable).JSON(fiber.Map{
			"success": false,
			"status":  "unhealthy",
			"message": "Database pool not initialized",
		})
	}

	if err := h.pool.Ping(context.Background()); err != nil {
		return c.Status(fiber.StatusServiceUnavailable).JSON(fiber.Map{
			"success": false,
			"status":  "unhealthy",
			"message": "Database connection failed",
		})
	}

	return response.Success(c, fiber.Map{
		"status": "ready",
	})
}
