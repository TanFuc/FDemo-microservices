package services

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"
)

// HealthService provides health check functionality
type HealthService struct {
	dbPool      *pgxpool.Pool
	redisClient *redis.Client
}

// NewHealthService creates a new health service
func NewHealthService(dbPool *pgxpool.Pool, redisClient *redis.Client) *HealthService {
	return &HealthService{
		dbPool:      dbPool,
		redisClient: redisClient,
	}
}

// HealthStatus represents the health status of the service
type HealthStatus struct {
	Status    string                    `json:"status"`
	Timestamp time.Time                 `json:"timestamp"`
	Services  map[string]ServiceHealth  `json:"services"`
}

// ServiceHealth represents the health of an individual service
type ServiceHealth struct {
	Status  string `json:"status"`
	Latency string `json:"latency,omitempty"`
	Error   string `json:"error,omitempty"`
}

// Check performs health checks on all dependencies
func (s *HealthService) Check(ctx context.Context) *HealthStatus {
	status := &HealthStatus{
		Status:    "healthy",
		Timestamp: time.Now(),
		Services:  make(map[string]ServiceHealth),
	}

	// Check PostgreSQL
	status.Services["postgres"] = s.checkPostgres(ctx)

	// Check Redis
	status.Services["redis"] = s.checkRedis(ctx)

	// Determine overall status
	for _, svc := range status.Services {
		if svc.Status != "healthy" {
			status.Status = "degraded"
			break
		}
	}

	return status
}

// checkPostgres checks PostgreSQL connectivity
func (s *HealthService) checkPostgres(ctx context.Context) ServiceHealth {
	if s.dbPool == nil {
		return ServiceHealth{Status: "unconfigured"}
	}

	start := time.Now()
	err := s.dbPool.Ping(ctx)
	latency := time.Since(start)

	if err != nil {
		return ServiceHealth{
			Status:  "unhealthy",
			Latency: latency.String(),
			Error:   err.Error(),
		}
	}

	return ServiceHealth{
		Status:  "healthy",
		Latency: latency.String(),
	}
}

// checkRedis checks Redis connectivity
func (s *HealthService) checkRedis(ctx context.Context) ServiceHealth {
	if s.redisClient == nil {
		return ServiceHealth{Status: "unconfigured"}
	}

	start := time.Now()
	err := s.redisClient.Ping(ctx).Err()
	latency := time.Since(start)

	if err != nil {
		return ServiceHealth{
			Status:  "unhealthy",
			Latency: latency.String(),
			Error:   err.Error(),
		}
	}

	return ServiceHealth{
		Status:  "healthy",
		Latency: latency.String(),
	}
}

// GetDBStats returns database pool statistics
func (s *HealthService) GetDBStats() *pgxpool.Stat {
	if s.dbPool == nil {
		return nil
	}
	stat := s.dbPool.Stat()
	return stat
}
