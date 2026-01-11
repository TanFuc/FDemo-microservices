package main

import (
	"log"

	"microservices/inventory/internal/config"
	"microservices/inventory/internal/handler"
	"microservices/inventory/internal/infrastructure"
	"microservices/inventory/internal/usecase"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/logger"
	"github.com/gofiber/fiber/v2/middleware/recover"
)

func main() {
	cfg := config.Load()

	db, err := infrastructure.NewPostgresConnection(infrastructure.PostgresConfig{
		Host:     cfg.Postgres.Host,
		Port:     cfg.Postgres.Port,
		User:     cfg.Postgres.User,
		Password: cfg.Postgres.Password,
		DBName:   cfg.Postgres.DBName,
		SSLMode:  cfg.Postgres.SSLMode,
	})
	if err != nil {
		log.Fatalf("Failed to connect to PostgreSQL: %v", err)
	}

	if err := infrastructure.AutoMigrate(db); err != nil {
		log.Fatalf("Failed to run migrations: %v", err)
	}

	redisClient, err := infrastructure.NewRedisClient(infrastructure.RedisConfig{
		Addr:     cfg.Redis.Addr,
		Password: cfg.Redis.Password,
		DB:       cfg.Redis.DB,
	})
	if err != nil {
		log.Fatalf("Failed to connect to Redis: %v", err)
	}
	defer redisClient.Close()

	cacheRepo := infrastructure.NewCacheRepository(redisClient)
	inventoryRepo := infrastructure.NewInventoryRepository(db, cacheRepo)
	reservationRepo := infrastructure.NewReservationRepository(db)

	inventoryUseCase := usecase.NewInventoryUseCase(
		db,
		inventoryRepo,
		reservationRepo,
		cacheRepo,
		config.GetReservationTTL(),
	)

	app := fiber.New(fiber.Config{
		ErrorHandler: func(c *fiber.Ctx, err error) error {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"error": err.Error(),
			})
		},
	})

	app.Use(logger.New())
	app.Use(recover.New())

	app.Get("/health", func(c *fiber.Ctx) error {
		return c.JSON(fiber.Map{
			"status": "healthy",
		})
	})

	inventoryHandler := handler.NewInventoryHandler(inventoryUseCase)
	inventoryHandler.RegisterRoutes(app)

	log.Printf("Starting server on port %s", cfg.Server.Port)
	if err := app.Listen(":" + cfg.Server.Port); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}
