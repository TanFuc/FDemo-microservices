package main

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"microservices/search/internal/config"
	"microservices/search/internal/infrastructure/elastic"
	"microservices/search/internal/infrastructure/nats"
)

func main() {
	// Setup structured logging
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelDebug,
	}))
	slog.SetDefault(logger)

	logger.Info("starting search-service worker")

	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		logger.Error("failed to load configuration", "error", err)
		os.Exit(1)
	}

	// Create context with cancellation
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Initialize Elasticsearch client
	elasticClient, err := elastic.NewClient(cfg.Elasticsearch.Addresses, logger)
	if err != nil {
		logger.Error("failed to create elasticsearch client", "error", err)
		os.Exit(1)
	}
	defer elasticClient.Close()

	// Ensure index exists with proper mapping
	if err := elasticClient.EnsureIndex(ctx); err != nil {
		logger.Error("failed to ensure index", "error", err)
		os.Exit(1)
	}

	// Initialize NATS consumer
	consumer, err := nats.NewConsumer(cfg.NATS.URL, elasticClient, logger)
	if err != nil {
		logger.Error("failed to create nats consumer", "error", err)
		os.Exit(1)
	}

	// Ensure stream exists
	if err := consumer.EnsureStream(ctx); err != nil {
		logger.Error("failed to ensure stream", "error", err)
		os.Exit(1)
	}

	// Start consuming events
	if err := consumer.Start(ctx); err != nil {
		logger.Error("failed to start consumer", "error", err)
		os.Exit(1)
	}

	logger.Info("worker started, waiting for events")

	// Wait for shutdown signal
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	<-sigChan

	logger.Info("shutdown signal received")

	// Graceful shutdown
	cancel()
	if err := consumer.Stop(); err != nil {
		logger.Error("error during consumer shutdown", "error", err)
	}

	logger.Info("worker stopped")
}
