package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"microservices/notification/internal/bridge"
	"microservices/notification/internal/config"
	"microservices/notification/internal/infrastructure"
	"microservices/notification/internal/provider"
	"microservices/notification/internal/worker"
)

func main() {
	log.Println("Starting Notification Service...")

	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("Failed to load configuration: %v", err)
	}

	// Create context with cancellation
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Initialize infrastructure
	rabbitmq, err := infrastructure.NewRabbitMQ(cfg.RabbitMQ.URL)
	if err != nil {
		log.Fatalf("Failed to connect to RabbitMQ: %v", err)
	}
	defer rabbitmq.Close()

	natsConn, err := infrastructure.NewNATS(cfg.NATS.URL)
	if err != nil {
		log.Fatalf("Failed to connect to NATS: %v", err)
	}
	defer natsConn.Close()

	mongodb, err := infrastructure.NewMongoDB(cfg.MongoDB.URI, cfg.MongoDB.Database)
	if err != nil {
		log.Fatalf("Failed to connect to MongoDB: %v", err)
	}
	defer mongodb.Close(context.Background())

	// Get templates directory path
	execPath, err := os.Executable()
	if err != nil {
		log.Fatalf("Failed to get executable path: %v", err)
	}
	templatesDir := filepath.Join(filepath.Dir(execPath), "..", "templates")

	// Try current working directory if templates not found
	if _, err := os.Stat(templatesDir); os.IsNotExist(err) {
		cwd, _ := os.Getwd()
		templatesDir = filepath.Join(cwd, "templates")
	}

	// Initialize template engine
	templateEngine, err := provider.NewTemplateEngine(templatesDir)
	if err != nil {
		log.Fatalf("Failed to initialize template engine: %v", err)
	}

	// Preload templates
	if err := templateEngine.PreloadTemplates(); err != nil {
		log.Printf("Warning: Failed to preload templates: %v", err)
	}

	// Initialize email provider
	var emailProvider provider.EmailProvider
	if cfg.SMTP.User == "" || cfg.SMTP.Pass == "" {
		log.Println("SMTP credentials not configured, using mock email provider")
		emailProvider = provider.NewMockEmailProvider()
	} else {
		emailProvider = provider.NewSMTPEmailProvider(cfg.SMTP)
	}

	// Initialize and start the bridge (NATS -> RabbitMQ)
	orderListener := bridge.NewOrderEventListener(natsConn, rabbitmq)
	if err := orderListener.Start(ctx); err != nil {
		log.Fatalf("Failed to start order event listener: %v", err)
	}
	defer orderListener.Stop()

	// Initialize and start the email consumer (RabbitMQ -> Email)
	emailConsumer := worker.NewEmailConsumer(rabbitmq, mongodb, emailProvider, templateEngine)
	if err := emailConsumer.Start(ctx); err != nil {
		log.Fatalf("Failed to start email consumer: %v", err)
	}
	defer emailConsumer.Stop()

	log.Println("Notification Service started successfully")
	log.Println("- Bridge: Listening for NATS events (order.created)")
	log.Println("- Worker: Processing email notifications from RabbitMQ")

	// Wait for shutdown signal
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)

	<-sigCh
	log.Println("Shutdown signal received, initiating graceful shutdown...")

	// Cancel context to stop workers
	cancel()

	// Give workers time to finish
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer shutdownCancel()

	// Wait for shutdown to complete or timeout
	select {
	case <-shutdownCtx.Done():
		log.Println("Shutdown timeout reached, forcing exit")
	case <-time.After(5 * time.Second):
		log.Println("Graceful shutdown completed")
	}

	log.Println("Notification Service stopped")
}
