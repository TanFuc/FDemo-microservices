package main

import (
	"microservices/campaign/internal/app"
	"microservices/campaign/internal/config"
	"microservices/campaign/pkg/logger"
)

// @title Campaign Service API
// @version 1.0
// @description Campaign and voucher management microservice with HTTP and gRPC support
// @termsOfService http://swagger.io/terms/

// @contact.name API Support
// @contact.email support@example.com

// @license.name Apache 2.0
// @license.url http://www.apache.org/licenses/LICENSE-2.0.html

// @host localhost:8083
// @BasePath /api/v1

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
