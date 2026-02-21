package main

import (
	"microservices/auth/internal/app"
	"microservices/auth/internal/config"
	"microservices/auth/pkg/logger"

	_ "microservices/auth/docs"
)

// @title Tafu Auth Service API
// @version 1.0
// @description Authentication and authorization service for Tafu e-commerce platform
// @termsOfService http://swagger.io/terms/

// @contact.name API Support
// @contact.email support@tafu.vn

// @license.name Apache 2.0
// @license.url http://www.apache.org/licenses/LICENSE-2.0.html

// @host localhost:3001
// @BasePath /api/v1

// @securityDefinitions.apikey BearerAuth
// @in header
// @name Authorization
// @description Enter your bearer token in the format: Bearer {token}

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

	// Create and run application
	application, err := app.New(cfg)
	if err != nil {
		logger.Fatal().Err(err).Msg("Failed to initialize application")
	}

	if err := application.Run(); err != nil {
		logger.Fatal().Err(err).Msg("Application error")
	}
}
