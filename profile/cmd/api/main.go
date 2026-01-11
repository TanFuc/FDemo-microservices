package main

import (
	"context"
	"os"
	"os/signal"
	"syscall"
	"time"

	"tafu-profile/internal/config"
	"tafu-profile/internal/delivery/event"
	"tafu-profile/internal/delivery/http/handler"
	"tafu-profile/internal/delivery/http/router"
	"tafu-profile/internal/domain/service"
	"tafu-profile/internal/infrastructure/database"
	"tafu-profile/pkg/logger"
)

func main() {
	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		panic("Failed to load configuration: " + err.Error())
	}

	// Initialize logger
	logger.Init(cfg.App.Env)
	logger.Info().
		Str("app", cfg.App.Name).
		Str("env", cfg.App.Env).
		Str("port", cfg.App.Port).
		Msg("Starting application")

	// Initialize MongoDB
	mongodb, err := database.NewMongoDB(cfg)
	if err != nil {
		logger.Fatal().Err(err).Msg("Failed to connect to MongoDB")
	}

	// Initialize repositories
	profileRepo := database.NewProfileRepository(mongodb)
	addressRepo := database.NewAddressRepository(mongodb)

	// Initialize services
	profileService := service.NewProfileService(profileRepo, addressRepo)

	// Initialize event listener
	listener := event.NewUserRegisteredListener(cfg, profileService)
	ctx, cancel := context.WithCancel(context.Background())

	// Start event listener (non-blocking)
	go func() {
		if err := listener.Start(ctx); err != nil {
			logger.Warn().Err(err).Msg("Failed to start event listener, continuing without it")
		}
	}()

	// Initialize handlers
	profileHandler := handler.NewProfileHandler(profileService)
	internalHandler := handler.NewInternalHandler(cfg, profileService)
	healthHandler := handler.NewHealthHandler(mongodb)

	// Initialize router
	r := router.NewRouter(cfg, profileHandler, internalHandler, healthHandler)
	app := r.Setup()

	// Graceful shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		if err := app.Listen(":" + cfg.App.Port); err != nil {
			logger.Fatal().Err(err).Msg("Failed to start server")
		}
	}()

	logger.Info().Str("port", cfg.App.Port).Msg("Server started")

	<-quit
	logger.Info().Msg("Shutting down server...")

	// Cancel context to stop listener
	cancel()

	// Stop event listener
	listener.Stop()

	// Graceful shutdown with timeout
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer shutdownCancel()

	if err := app.ShutdownWithContext(shutdownCtx); err != nil {
		logger.Error().Err(err).Msg("Server forced to shutdown")
	}

	// Close MongoDB connection
	mongodb.Close(shutdownCtx)

	logger.Info().Msg("Server exited properly")
}
