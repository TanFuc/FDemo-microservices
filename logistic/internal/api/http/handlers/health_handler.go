package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"microservices/logistic/internal/core/services"
)

// HealthHandler handles health check requests
type HealthHandler struct {
	healthService *services.HealthService
}

// NewHealthHandler creates a new health handler
func NewHealthHandler(healthService *services.HealthService) *HealthHandler {
	return &HealthHandler{
		healthService: healthService,
	}
}

// Health returns basic health status
// GET /health
func (h *HealthHandler) Health(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"status":  "ok",
		"service": "logistics-service",
	})
}

// HealthDetailed returns detailed health status with dependency checks
// GET /health/detailed
func (h *HealthHandler) HealthDetailed(c *gin.Context) {
	if h.healthService == nil {
		c.JSON(http.StatusOK, gin.H{
			"status":  "ok",
			"service": "logistics-service",
		})
		return
	}

	status := h.healthService.Check(c.Request.Context())

	httpStatus := http.StatusOK
	if status.Status != "healthy" {
		httpStatus = http.StatusServiceUnavailable
	}

	c.JSON(httpStatus, status)
}

// Ready returns readiness status
// GET /ready
func (h *HealthHandler) Ready(c *gin.Context) {
	if h.healthService == nil {
		c.JSON(http.StatusOK, gin.H{"ready": true})
		return
	}

	status := h.healthService.Check(c.Request.Context())

	if status.Status != "healthy" {
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"ready":  false,
			"reason": "dependencies not healthy",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{"ready": true})
}

// DBStats returns database pool statistics
// GET /metrics/db
func (h *HealthHandler) DBStats(c *gin.Context) {
	if h.healthService == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "health service not configured"})
		return
	}

	stats := h.healthService.GetDBStats()
	if stats == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "database not configured"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"acquired_connections":   stats.AcquiredConns(),
		"idle_connections":       stats.IdleConns(),
		"total_connections":      stats.TotalConns(),
		"max_connections":        stats.MaxConns(),
		"constructing_connections": stats.ConstructingConns(),
	})
}
