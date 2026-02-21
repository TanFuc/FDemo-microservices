package main

import (
	"microservices/auth/internal/app"
	"microservices/auth/internal/config"
	"microservices/auth/pkg/logger"
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
		Msg("Starting application")

	// Create and run application
	application, err := app.New(cfg)
	if err != nil {
		logger.Fatal().Err(err).Msg("Failed to initialize application")
	}

	if err := application.Run(); err != nil {
		logger.Fatal().Err(err).Msg("Application error")
	}
}
