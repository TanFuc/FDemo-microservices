package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"microservices/logistic/config"
	rediscache "microservices/logistic/internal/adapters/cache/redis"
	"microservices/logistic/internal/adapters/messaging/nats"
	"microservices/logistic/internal/adapters/providers"
	"microservices/logistic/internal/adapters/providers/ghn"
	"microservices/logistic/internal/adapters/providers/ghtk"
	"microservices/logistic/internal/adapters/repository/postgres"
	"microservices/logistic/internal/api/http"
	"microservices/logistic/internal/core/services"
	"microservices/pkg/authclient"
)

func main() {
	log.Println("Starting logistics-service...")

	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	if err := cfg.Validate(); err != nil {
		log.Fatalf("Invalid config: %v", err)
	}

	// Create context for graceful shutdown
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Initialize PostgreSQL connection pool
	log.Println("Connecting to PostgreSQL...")
	dbPool, err := pgxpool.New(ctx, cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer dbPool.Close()

	// Test database connection
	if err := dbPool.Ping(ctx); err != nil {
		log.Fatalf("Failed to ping database: %v", err)
	}
	log.Println("Connected to PostgreSQL")

	// Initialize Redis client
	log.Println("Connecting to Redis...")
	redisClient, err := rediscache.NewRedisClient(cfg.RedisURL)
	if err != nil {
		log.Fatalf("Failed to connect to Redis: %v", err)
	}
	defer redisClient.Close()
	log.Println("Connected to Redis")

	// Initialize NATS publisher
	log.Println("Connecting to NATS...")
	natsPublisher, err := nats.NewPublisher(nats.Config{
		URL:        cfg.NATSURL,
		StreamName: cfg.NATSStreamName,
	})
	if err != nil {
		log.Fatalf("Failed to connect to NATS: %v", err)
	}
	defer natsPublisher.Close()
	log.Println("Connected to NATS")

	// Initialize adapters
	shippingRepo := postgres.NewShippingRepository(dbPool)
	webhookLogRepo := postgres.NewWebhookLogRepository(dbPool)
	cache := rediscache.NewCache(redisClient)

	// Initialize provider factory
	providerCfg := providers.ProviderConfig{}
	if cfg.HasGHNCredentials() {
		providerCfg.GHN = ghn.Config{
			APIURL: cfg.GHNAPIURL,
			Token:  cfg.GHNToken,
			ShopID: cfg.GHNShopID,
		}
		log.Println("GHN provider configured")
	}
	if cfg.HasGHTKCredentials() {
		providerCfg.GHTK = ghtk.Config{
			APIURL: cfg.GHTKAPIURL,
			Token:  cfg.GHTKToken,
		}
		log.Println("GHTK provider configured")
	}

	providerFactory, err := providers.NewFactory(providerCfg)
	if err != nil {
		log.Fatalf("Failed to create provider factory: %v", err)
	}
	log.Printf("Available providers: %v", providerFactory.ListProviders())

	// Initialize services
	shippingService := services.NewShippingService(
		providerFactory,
		shippingRepo,
		cache,
		natsPublisher,
	)

	webhookService := services.NewWebhookServiceWithLogging(
		providerFactory,
		shippingRepo,
		webhookLogRepo,
		natsPublisher,
	)

	// Initialize health service
	healthService := services.NewHealthService(dbPool, redisClient)

	// Initialize Auth gRPC Client
	authGRPCAddr := os.Getenv("AUTH_GRPC_ADDR")
	if authGRPCAddr == "" {
		authGRPCAddr = "localhost:50051"
	}
	authClient, err := authclient.NewClient(&authclient.Config{
		GRPCAddr: authGRPCAddr,
	})
	if err != nil {
		log.Printf("Warning: Failed to connect to Auth service: %v (authorization will fail)", err)
	} else {
		log.Printf("Connected to Auth gRPC Service at %s", authGRPCAddr)
		defer authClient.Close()
	}

	// Create auth middleware for Gin
	var authMiddleware *authclient.GinMiddleware
	if authClient != nil {
		authMiddleware = authclient.NewGinMiddleware(authClient)
	}

	// Initialize HTTP router with health service
	router := http.NewRouterWithHealth(shippingService, webhookService, healthService, authMiddleware)

	// Initialize and start NATS consumer for order.packed events
	natsConsumer, err := nats.NewConsumer(nats.ConsumerConfig{
		URL:          cfg.NATSURL,
		StreamName:   "ORDERS",
		ConsumerName: "logistics-service",
		Subject:      "order.packed",
	}, shippingService)
	if err != nil {
		log.Printf("Warning: Failed to create NATS consumer: %v", err)
	} else {
		if err := natsConsumer.Start(ctx, nats.ConsumerConfig{
			URL:          cfg.NATSURL,
			ConsumerName: "logistics-service",
		}); err != nil {
			log.Printf("Warning: Failed to start NATS consumer: %v", err)
		} else {
			defer natsConsumer.Close()
			log.Println("NATS consumer started for order.packed events")
		}
	}

	// Initialize and start payment consumer for auto-shipment creation
	// Uses mock order fetcher for now - replace with HTTPOrderFetcher in production
	orderFetcher := &nats.MockOrderFetcher{}
	paymentConsumer, err := nats.NewPaymentConsumer(nats.PaymentConsumerConfig{
		URL:          cfg.NATSURL,
		ConsumerName: "logistics-payment-consumer",
	}, shippingService, orderFetcher)
	if err != nil {
		log.Printf("Warning: Failed to create payment consumer: %v", err)
	} else {
		if err := paymentConsumer.Start(ctx, nats.PaymentConsumerConfig{
			URL:          cfg.NATSURL,
			ConsumerName: "logistics-payment-consumer",
		}); err != nil {
			log.Printf("Warning: Failed to start payment consumer: %v", err)
		} else {
			defer paymentConsumer.Close()
			log.Println("Payment consumer started for auto-shipment creation")
		}
	}

	// Start server in goroutine
	go func() {
		addr := ":" + cfg.Port
		log.Printf("HTTP server listening on %s", addr)
		if err := router.Run(addr); err != nil {
			log.Printf("HTTP server error: %v", err)
			cancel()
		}
	}()

	// Wait for shutdown signal
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("Shutting down server...")

	// Give outstanding requests time to complete
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer shutdownCancel()

	<-shutdownCtx.Done()
	log.Println("Server stopped")
}
