package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"microservices/catalog/internal/config"
	httpdelivery "microservices/catalog/internal/delivery/http"
	"microservices/catalog/internal/delivery/http/middleware"
	authgrpc "microservices/catalog/internal/infrastructure/grpc"
	"microservices/catalog/internal/infrastructure/nats"
	mongorepo "microservices/catalog/internal/repository/mongo"
	redisrepo "microservices/catalog/internal/repository/redis"
	"microservices/catalog/internal/usecase"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
	"github.com/gofiber/fiber/v2/middleware/logger"
	"github.com/gofiber/fiber/v2/middleware/recover"
)

func main() {
	cfg := config.Load()

	// Connect to MongoDB
	ctx, cancel := context.WithTimeout(context.Background(), cfg.MongoDB.Timeout)
	defer cancel()

	db, err := mongorepo.Connect(ctx, cfg.MongoDB.URI, cfg.MongoDB.Database)
	if err != nil {
		log.Fatalf("Failed to connect to MongoDB: %v", err)
	}
	log.Println("Connected to MongoDB")

	// Create MongoDB repositories
	categoryRepo, err := mongorepo.NewCategoryRepository(db)
	if err != nil {
		log.Fatalf("Failed to create category repository: %v", err)
	}

	brandRepo, err := mongorepo.NewBrandRepository(db)
	if err != nil {
		log.Fatalf("Failed to create brand repository: %v", err)
	}

	productRepo, err := mongorepo.NewProductRepository(db)
	if err != nil {
		log.Fatalf("Failed to create product repository: %v", err)
	}

	// Connect to Redis
	redisClient, err := redisrepo.Connect(cfg.Redis.Addr, cfg.Redis.Password, cfg.Redis.DB)
	if err != nil {
		log.Printf("Warning: Failed to connect to Redis: %v (continuing without cache)", err)
		redisClient = nil
	} else {
		log.Println("Connected to Redis")
	}

	var cacheRepo *redisrepo.CacheRepository
	if redisClient != nil {
		cacheRepo = redisrepo.NewCacheRepository(redisClient)
	}

	// Connect to NATS
	publisher, err := nats.NewPublisher(cfg.NATS.URL, cfg.NATS.StreamName)
	if err != nil {
		log.Printf("Warning: Failed to connect to NATS: %v (continuing without event publishing)", err)
		publisher = nil
	} else {
		log.Println("Connected to NATS JetStream")
		defer publisher.Close()
	}

	// Connect to Auth gRPC Service
	authClient, err := authgrpc.NewAuthClient(cfg.Auth.GRPCAddr)
	if err != nil {
		log.Printf("Warning: Failed to connect to Auth service: %v (authorization will fail)", err)
		authClient = nil
	} else {
		log.Printf("Connected to Auth gRPC Service at %s", cfg.Auth.GRPCAddr)
		defer authClient.Close()
	}

	// Create usecases
	categoryUsecase := usecase.NewCategoryUsecase(categoryRepo, cacheRepo)
	brandUsecase := usecase.NewBrandUsecase(brandRepo)
	productUsecase := usecase.NewProductUsecase(productRepo, categoryRepo, brandRepo, cacheRepo, publisher)

	// Create Fiber app
	app := fiber.New(fiber.Config{
		ReadTimeout:  cfg.Server.ReadTimeout,
		WriteTimeout: cfg.Server.WriteTimeout,
		AppName:      "Catalog Service",
	})

	// Middleware
	app.Use(recover.New())
	app.Use(logger.New())
	app.Use(cors.New())

	// Health check
	app.Get("/health", func(c *fiber.Ctx) error {
		return c.JSON(fiber.Map{
			"status": "ok",
			"time":   time.Now().Format(time.RFC3339),
		})
	})

	// Create auth middleware
	var authMiddleware *middleware.AuthMiddleware
	if authClient != nil {
		authMiddleware = middleware.NewAuthMiddleware(authClient)
	}

	// API routes
	api := app.Group("/api/v1")

	// Register handlers
	categoryHandler := httpdelivery.NewCategoryHandler(categoryUsecase)
	categoryHandler.RegisterRoutes(api)

	brandHandler := httpdelivery.NewBrandHandler(brandUsecase)
	brandHandler.RegisterRoutes(api)

	productHandler := httpdelivery.NewProductHandler(productUsecase, authClient)
	productHandler.RegisterRoutes(api, authMiddleware)

	// Graceful shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		<-quit
		log.Println("Shutting down server...")
		if err := app.Shutdown(); err != nil {
			log.Printf("Error shutting down server: %v", err)
		}
	}()

	// Start server
	addr := fmt.Sprintf("%s:%d", cfg.Server.Host, cfg.Server.Port)
	log.Printf("Starting server on %s", addr)
	if err := app.Listen(addr); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}
