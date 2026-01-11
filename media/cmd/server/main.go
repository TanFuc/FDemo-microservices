package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/tafu-media/media-service/internal/config"
	"github.com/tafu-media/media-service/internal/handler"
	"github.com/tafu-media/media-service/internal/infrastructure/queue"
	"github.com/tafu-media/media-service/internal/infrastructure/storage"
	"github.com/tafu-media/media-service/internal/usecase"
	"github.com/tafu-media/media-service/internal/worker"
)

func main() {
	// Load configuration
	cfg := config.Load()

	// Initialize MinIO client
	minioClient, err := storage.NewMinIOClient(cfg.MinIO)
	if err != nil {
		log.Fatalf("Failed to create MinIO client: %v", err)
	}

	// Ensure bucket exists with proper configuration
	ctx := context.Background()
	if err := minioClient.EnsureBucket(ctx); err != nil {
		log.Fatalf("Failed to ensure bucket: %v", err)
	}
	log.Println("MinIO bucket ready")

	// Initialize NATS client
	natsClient, err := queue.NewNATSClient(cfg.NATS)
	if err != nil {
		log.Fatalf("Failed to create NATS client: %v", err)
	}
	defer natsClient.Close()
	log.Println("NATS JetStream ready")

	// Initialize use case
	mediaUseCase := usecase.NewMediaUseCase(minioClient, natsClient, cfg.Media, cfg.NATS.Subject)

	// Initialize and start image processor worker
	processor := worker.NewImageProcessor(minioClient, natsClient, cfg.Media, cfg.NATS.Subject, 4)

	workerCtx, workerCancel := context.WithCancel(context.Background())
	defer workerCancel()

	if err := processor.Start(workerCtx); err != nil {
		log.Fatalf("Failed to start image processor: %v", err)
	}
	log.Println("Image processor worker started")

	// Initialize HTTP handler and router
	mediaHandler := handler.NewMediaHandler(mediaUseCase)
	router := handler.NewRouter(mediaHandler)

	// Create HTTP server
	server := &http.Server{
		Addr:         ":" + cfg.Server.Port,
		Handler:      router,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	// Start server in goroutine
	go func() {
		log.Printf("Starting server on port %s", cfg.Server.Port)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Server failed: %v", err)
		}
	}()

	// Wait for interrupt signal
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("Shutting down server...")

	// Graceful shutdown
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer shutdownCancel()

	// Stop worker
	workerCancel()
	processor.Stop()
	log.Println("Worker stopped")

	// Shutdown HTTP server
	if err := server.Shutdown(shutdownCtx); err != nil {
		log.Printf("Server forced to shutdown: %v", err)
	}

	fmt.Println("Server exited")
}
