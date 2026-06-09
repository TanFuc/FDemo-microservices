package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"microservices/pkg/authclient"
	grpcclient "microservices/review/internal/adapter/grpc"
	"microservices/review/internal/adapter/mongodb"
	rediscache "microservices/review/internal/adapter/redis"
	"microservices/review/internal/config"
	"microservices/review/internal/core/service"
	"microservices/review/internal/handler"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
	"github.com/gofiber/fiber/v2/middleware/logger"
	"github.com/gofiber/fiber/v2/middleware/recover"
	"github.com/redis/go-redis/v9"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

func main() {
	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("Failed to load configuration: %v", err)
	}

	// Initialize MongoDB
	mongoClient, err := connectMongoDB(cfg.MongoDB.URI)
	if err != nil {
		log.Fatalf("Failed to connect to MongoDB: %v", err)
	}
	defer func() {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		if err := mongoClient.Disconnect(ctx); err != nil {
			log.Printf("Error disconnecting from MongoDB: %v", err)
		}
	}()

	mongoDB := mongoClient.Database(cfg.MongoDB.Database)

	// Initialize Redis
	redisClient := redis.NewClient(&redis.Options{
		Addr:     cfg.Redis.Addr,
		Password: cfg.Redis.Password,
		DB:       cfg.Redis.DB,
	})
	defer redisClient.Close()

	// Verify Redis connection
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := redisClient.Ping(ctx).Err(); err != nil {
		log.Printf("Warning: Failed to connect to Redis: %v. Continuing without cache.", err)
	}

	// Initialize repositories
	reviewRepo := mongodb.NewReviewRepository(mongoDB)
	productRatingRepo := mongodb.NewProductRatingRepository(mongoDB)
	cacheRepo := rediscache.NewCacheRepository(redisClient)

	// Initialize gRPC client for Order Service
	orderClient, err := grpcclient.NewOrderServiceClient(cfg.GRPC.OrderServiceAddr)
	if err != nil {
		log.Printf("Warning: Failed to connect to Order Service: %v", err)
		// Use mock client for development
		orderClient = grpcclient.NewMockOrderServiceClient()
	}
	defer orderClient.Close()

	// Initialize service
	reviewService := service.NewReviewService(reviewRepo, productRatingRepo, cacheRepo, orderClient)

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

	// Create auth middleware
	var authMiddleware *authclient.FiberMiddleware
	if authClient != nil {
		authMiddleware = authclient.NewFiberMiddleware(authClient)
	}

	// Initialize handler
	reviewHandler := handler.NewReviewHandler(reviewService, authMiddleware)

	// Initialize Fiber app
	app := fiber.New(fiber.Config{
		ErrorHandler: customErrorHandler,
	})

	// Middleware
	app.Use(recover.New())
	app.Use(logger.New(logger.Config{
		Format: "[${time}] ${status} - ${method} ${path} ${latency}\n",
	}))
	app.Use(cors.New(cors.Config{
		AllowOrigins: "*",
		AllowMethods: "GET,POST,PUT,PATCH,DELETE,OPTIONS",
		AllowHeaders: "Origin, Content-Type, Accept, Authorization",
	}))

	// Health check
	app.Get("/health", func(c *fiber.Ctx) error {
		return c.JSON(fiber.Map{
			"status":  "healthy",
			"service": "tafu-review",
		})
	})

	// Register routes
	reviewHandler.RegisterRoutes(app)

	// Graceful shutdown
	go func() {
		if err := app.Listen(":" + cfg.App.Port); err != nil {
			log.Fatalf("Failed to start server: %v", err)
		}
	}()

	log.Printf("Server started on port %s", cfg.App.Port)

	// Wait for interrupt signal
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("Shutting down server...")

	if err := app.Shutdown(); err != nil {
		log.Printf("Error shutting down server: %v", err)
	}

	log.Println("Server stopped")
}

func connectMongoDB(uri string) (*mongo.Client, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	clientOptions := options.Client().ApplyURI(uri)
	client, err := mongo.Connect(ctx, clientOptions)
	if err != nil {
		return nil, err
	}

	// Verify connection
	if err := client.Ping(ctx, nil); err != nil {
		return nil, err
	}

	log.Println("Connected to MongoDB")
	return client, nil
}

func customErrorHandler(c *fiber.Ctx, err error) error {
	code := fiber.StatusInternalServerError

	if e, ok := err.(*fiber.Error); ok {
		code = e.Code
	}

	return c.Status(code).JSON(fiber.Map{
		"error":   "INTERNAL_ERROR",
		"message": err.Error(),
	})
}
