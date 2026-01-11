package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"microservices/analytic/internal/api"
	"microservices/analytic/internal/config"
	"microservices/analytic/internal/infrastructure/clickhouse"
	natsClient "microservices/analytic/internal/infrastructure/nats"
	"microservices/analytic/internal/worker"
)

func main() {
	log.Println("Starting Analytics Service...")

	// Load configuration
	cfg := config.Load()

	// Setup context with cancellation
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Initialize ClickHouse connection
	log.Println("Connecting to ClickHouse...")
	chConn, err := clickhouse.NewConnection(cfg.ClickHouse)
	if err != nil {
		log.Fatalf("Failed to connect to ClickHouse: %v", err)
	}
	defer chConn.Close()

	// Initialize ClickHouse schema
	log.Println("Initializing ClickHouse schema...")
	if err := clickhouse.InitSchema(ctx, chConn); err != nil {
		log.Fatalf("Failed to initialize schema: %v", err)
	}

	// Create event repository
	repo := clickhouse.NewEventRepository(chConn)

	// Initialize NATS client
	log.Println("Connecting to NATS...")
	nats, err := natsClient.NewClient(cfg.NATS)
	if err != nil {
		log.Fatalf("Failed to connect to NATS: %v", err)
	}
	defer nats.Close()

	// Start batch worker
	batchWorker := worker.NewBatchWorker(nats, repo, cfg.Worker)
	go func() {
		if err := batchWorker.Start(ctx); err != nil {
			log.Printf("Batch worker error: %v", err)
		}
	}()

	// Setup HTTP server
	handler := api.NewHandler(nats, repo)
	router := api.NewRouter(handler)

	server := &http.Server{
		Addr:         ":" + cfg.Server.Port,
		Handler:      router,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
		IdleTimeout:  30 * time.Second,
	}

	// Start HTTP server in goroutine
	go func() {
		log.Printf("HTTP server listening on port %s", cfg.Server.Port)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("HTTP server error: %v", err)
		}
	}()

	// Wait for shutdown signal
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	<-sigCh

	log.Println("Shutdown signal received, gracefully shutting down...")

	// Cancel context to stop worker
	cancel()

	// Shutdown HTTP server with timeout
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer shutdownCancel()

	if err := server.Shutdown(shutdownCtx); err != nil {
		log.Printf("HTTP server shutdown error: %v", err)
	}

	log.Println("Analytics Service stopped")
}
