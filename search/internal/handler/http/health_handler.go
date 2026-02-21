package http

import (
	"github.com/gofiber/fiber/v2"
)

// HealthHandler handles health check HTTP requests
type HealthHandler struct{}

// NewHealthHandler creates a new health handler
func NewHealthHandler() *HealthHandler {
	return &HealthHandler{}
}

// Health returns basic health status
func (h *HealthHandler) Health(c *fiber.Ctx) error {
	return c.JSON(fiber.Map{
		"success": true,
		"data": fiber.Map{
			"status":  "healthy",
			"service": "search-service",
		},
	})
}

// Ready returns readiness status
func (h *HealthHandler) Ready(c *fiber.Ctx) error {
	return c.JSON(fiber.Map{
		"success": true,
		"data": fiber.Map{
			"status": "ready",
		},
	})
}
