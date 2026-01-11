package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	httpdelivery "microservices/cart/internal/delivery/http"
	mongorepo "microservices/cart/internal/infrastructure/mongo"
	redisrepo "microservices/cart/internal/infrastructure/redis"
	"microservices/cart/internal/usecase"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
	"github.com/gofiber/fiber/v2/middleware/logger"
	"github.com/gofiber/fiber/v2/middleware/recover"
)

func main() {
	// Initialize context
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Initialize Redis
	log.Println("Connecting to Redis...")
	redisConfig := redisrepo.NewConfigFromEnv()
	redisClient, err := redisrepo.NewClient(redisConfig)
	if err != nil {
		log.Fatalf("Failed to connect to Redis: %v", err)
	}
	defer redisClient.Close()
	log.Println("Connected to Redis successfully")

	// Initialize MongoDB
	log.Println("Connecting to MongoDB...")
	mongoConfig := mongorepo.NewConfigFromEnv()
	mongoClient, err := mongorepo.NewClient(ctx, mongoConfig)
	if err != nil {
		log.Fatalf("Failed to connect to MongoDB: %v", err)
	}
	defer func() {
		if err := mongoClient.Disconnect(context.Background()); err != nil {
			log.Printf("Error disconnecting from MongoDB: %v", err)
		}
	}()
	log.Println("Connected to MongoDB successfully")

	// Initialize repositories
	redisRepo := redisrepo.NewCartRepository(redisClient)
	mongoDB := mongorepo.GetDatabase(mongoClient, mongoConfig.Database)
	mongoRepo := mongorepo.NewCartRepository(mongoDB)

	// Initialize usecase
	cartUsecase := usecase.NewCartUsecase(redisRepo, mongoRepo)

	// Initialize Fiber app
	app := fiber.New(fiber.Config{
		AppName:      "Cart Service",
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
		IdleTimeout:  120 * time.Second,
	})

	// Middleware
	app.Use(recover.New())
	app.Use(logger.New(logger.Config{
		Format:     "${time} | ${status} | ${latency} | ${method} | ${path}\n",
		TimeFormat: "2006-01-02 15:04:05",
	}))
	app.Use(cors.New(cors.Config{
		AllowOrigins: "*",
		AllowMethods: "GET,POST,PUT,DELETE,OPTIONS",
		AllowHeaders: "Origin, Content-Type, Accept, Authorization",
	}))

	// Health check endpoint
	app.Get("/health", func(c *fiber.Ctx) error {
		return c.JSON(fiber.Map{
			"status":  "healthy",
			"service": "cart-service",
		})
	})

	// Register routes
	cartHandler := httpdelivery.NewCartHandler(cartUsecase)
	cartHandler.RegisterRoutes(app)

	// Get port from environment or default to 8080
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	// Start server in a goroutine
	go func() {
		log.Printf("Starting Cart Service on port %s...", port)
		if err := app.Listen(":" + port); err != nil {
			log.Fatalf("Failed to start server: %v", err)
		}
	}()

	// Graceful shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt, syscall.SIGTERM)
	<-quit

	log.Println("Shutting down server...")
	if err := app.ShutdownWithTimeout(30 * time.Second); err != nil {
		log.Fatalf("Server forced to shutdown: %v", err)
	}

	log.Println("Server exited properly")
}
