package main

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
	"github.com/gofiber/fiber/v2/middleware/logger"
	"github.com/gofiber/fiber/v2/middleware/recover"
	"microservices/search/internal/api/handler"
	"microservices/search/internal/config"
	"microservices/search/internal/infrastructure/cache"
	"microservices/search/internal/infrastructure/elastic"
	"microservices/search/internal/usecase"
)

func main() {
	// Setup structured logging
	slogger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelDebug,
	}))
	slog.SetDefault(slogger)

	slogger.Info("starting search-service api")

	// Load configuration
	cfg := config.Load()

	// Create context with cancellation
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Initialize Elasticsearch client
	elasticClient, err := elastic.NewClient(cfg.ElasticAddresses, slogger)
	if err != nil {
		slogger.Error("failed to create elasticsearch client", "error", err)
		os.Exit(1)
	}
	defer elasticClient.Close()

	// Ensure index exists with proper mapping
	if err := elasticClient.EnsureIndex(ctx); err != nil {
		slogger.Error("failed to ensure index", "error", err)
		os.Exit(1)
	}

	// Initialize Redis client
	redisClient, err := cache.NewRedisClient(cfg.RedisAddr, cfg.RedisPassword, cfg.RedisDB, slogger)
	if err != nil {
		slogger.Error("failed to create redis client", "error", err)
		os.Exit(1)
	}
	defer redisClient.Close()

	// Initialize usecase
	searchUsecase := usecase.NewSearchUsecase(elasticClient, redisClient, slogger)

	// Initialize handler
	searchHandler := handler.NewSearchHandler(searchUsecase, slogger)

	// Setup Fiber app
	app := fiber.New(fiber.Config{
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
		IdleTimeout:  120 * time.Second,
	})

	// Middleware
	app.Use(recover.New())
	app.Use(cors.New())
	app.Use(logger.New(logger.Config{
		Format: "[${time}] ${status} - ${latency} ${method} ${path}\n",
	}))

	// Routes
	app.Get("/health", searchHandler.Health)

	api := app.Group("/api")
	api.Get("/search", searchHandler.SearchGet)
	api.Post("/search", searchHandler.Search)

	// Start server in goroutine
	go func() {
		if err := app.Listen(":" + cfg.APIPort); err != nil {
			slogger.Error("server error", "error", err)
		}
	}()

	slogger.Info("api server started", "port", cfg.APIPort)

	// Wait for shutdown signal
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	<-sigChan

	slogger.Info("shutdown signal received")

	// Graceful shutdown
	cancel()
	if err := app.ShutdownWithTimeout(30 * time.Second); err != nil {
		slogger.Error("error during server shutdown", "error", err)
	}

	slogger.Info("api server stopped")
}
