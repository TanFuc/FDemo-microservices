package http

import (
	"context"
	"time"

	"github.com/gofiber/fiber/v2"
	"go.mongodb.org/mongo-driver/mongo"
)

// HealthHandler handles health check HTTP requests
type HealthHandler struct {
	mongoDB *mongo.Database
}

// NewHealthHandler creates a new health handler
func NewHealthHandler(mongoDB *mongo.Database) *HealthHandler {
	return &HealthHandler{
		mongoDB: mongoDB,
	}
}

// Health returns basic health status
func (h *HealthHandler) Health(c *fiber.Ctx) error {
	return c.JSON(fiber.Map{
		"success": true,
		"data": fiber.Map{
			"status":  "healthy",
			"service": "review-service",
		},
	})
}

// Ready returns readiness status with dependency checks
func (h *HealthHandler) Ready(c *fiber.Ctx) error {
	if h.mongoDB == nil {
		return c.Status(fiber.StatusServiceUnavailable).JSON(fiber.Map{
			"success": false,
			"status":  "unhealthy",
			"message": "MongoDB not initialized",
		})
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := h.mongoDB.Client().Ping(ctx, nil); err != nil {
		return c.Status(fiber.StatusServiceUnavailable).JSON(fiber.Map{
			"success": false,
			"status":  "unhealthy",
			"message": "MongoDB connection failed",
		})
	}

	return c.JSON(fiber.Map{
		"success": true,
		"data": fiber.Map{
			"status": "ready",
		},
	})
}
