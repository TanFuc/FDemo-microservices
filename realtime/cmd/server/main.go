package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
	"github.com/gofiber/fiber/v2/middleware/recover"
	"github.com/nats-io/nats.go"

	"microservices/pkg/logger"
	"microservices/realtime/internal/config"
	"microservices/realtime/internal/handler"
	"microservices/realtime/internal/hub"
)

func main() {
	logger.Init("development")
	cfg := config.Load()

	logger.Info().Int("port", cfg.Port).Msg("starting realtime websocket service")

	// 1. Initialize Hub
	connectionHub := hub.NewHub()

	// 2. Connect to NATS Backplane
	nc, err := nats.Connect(cfg.NatsURL)
	if err != nil {
		logger.Warn().Err(err).Str("url", cfg.NatsURL).Msg("failed to connect to nats, running in standalone memory mode")
	} else {
		defer nc.Close()
		backplane := hub.NewBackplane(nc, connectionHub)
		if err := backplane.Start(context.Background()); err != nil {
			logger.Error().Err(err).Msg("failed to start distributed backplane")
		} else {
			logger.Info().Msg("distributed nats backplane started")
			defer backplane.Stop()
		}
	}

	// 3. Initialize HTTP & WebSocket Server
	app := fiber.New(fiber.Config{
		DisableStartupMessage: true,
	})

	app.Use(recover.New())
	app.Use(cors.New(cors.Config{
		AllowOrigins: "*",
		AllowHeaders: "*",
	}))

	// Health Check
	healthH := handler.NewHealthHandler(connectionHub)
	app.Get("/health", healthH.Health)

	// WebSocket Route
	wsH := handler.NewWSHandler(connectionHub, cfg.JWTSecret)
	app.Use("/ws", wsH.UpgradeMiddleware())
	app.Get("/ws", wsH.HandleWebSocket())

	// 4. Graceful Shutdown Setup
	shutdownChan := make(chan os.Signal, 1)
	signal.Notify(shutdownChan, os.Interrupt, syscall.SIGTERM)

	go func() {
		addr := fmt.Sprintf(":%d", cfg.Port)
		if err := app.Listen(addr); err != nil {
			logger.Info().Err(err).Msg("http server stopped")
		}
	}()

	sig := <-shutdownChan
	logger.Info().Str("signal", sig.String()).Msg("initiating graceful shutdown")

	_, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := app.Shutdown(); err != nil {
		logger.Error().Err(err).Msg("server forced shutdown error")
	}

	logger.Info().Msg("realtime service exited cleanly")
}
