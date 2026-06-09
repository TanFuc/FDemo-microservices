//go:build legacy

package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"
	"microservices/campaign/internal/config"
	postgresRepo "microservices/campaign/internal/repository/postgres"
	redisRepo "microservices/campaign/internal/repository/redis"
	"microservices/campaign/internal/usecase"
)

func main() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Load configuration
	cfg := config.Load()

	// Initialize PostgreSQL connection pool
	pgPool, err := initPostgres(ctx, cfg.Postgres)
	if err != nil {
		log.Fatalf("failed to initialize postgres: %v", err)
	}
	defer pgPool.Close()

	// Initialize Redis client
	redisClient, err := initRedis(ctx, cfg.Redis)
	if err != nil {
		log.Fatalf("failed to initialize redis: %v", err)
	}
	defer redisClient.Close()

	// Initialize repositories
	voucherRepo := postgresRepo.NewVoucherRepository(pgPool)
	userVoucherRepo := postgresRepo.NewUserVoucherRepository(pgPool)
	voucherCacheRepo := redisRepo.NewVoucherCacheRepository(redisClient)

	// Initialize use case
	campaignUsecase := usecase.NewCampaignUsecase(voucherRepo, userVoucherRepo, voucherCacheRepo)

	// Log successful initialization
	log.Printf("Campaign Service initialized successfully")
	log.Printf("Server port: %s", cfg.Server.Port)

	// Example usage (would typically be exposed via HTTP/gRPC)
	_ = campaignUsecase

	// Graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	<-sigChan

	log.Println("Shutting down...")
}

func initPostgres(ctx context.Context, cfg config.PostgresConfig) (*pgxpool.Pool, error) {
	poolConfig, err := pgxpool.ParseConfig(cfg.DSN())
	if err != nil {
		return nil, fmt.Errorf("failed to parse postgres config: %w", err)
	}

	pool, err := pgxpool.NewWithConfig(ctx, poolConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to create postgres pool: %w", err)
	}

	if err := pool.Ping(ctx); err != nil {
		return nil, fmt.Errorf("failed to ping postgres: %w", err)
	}

	log.Println("Connected to PostgreSQL")
	return pool, nil
}

func initRedis(ctx context.Context, cfg config.RedisConfig) (*redis.Client, error) {
	client := redis.NewClient(&redis.Options{
		Addr:     cfg.Addr(),
		Password: cfg.Password,
		DB:       cfg.DB,
	})

	if err := client.Ping(ctx).Err(); err != nil {
		return nil, fmt.Errorf("failed to ping redis: %w", err)
	}

	log.Println("Connected to Redis")
	return client, nil
}
