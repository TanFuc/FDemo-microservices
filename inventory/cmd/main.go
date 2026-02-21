package main

import (
	"flag"
	"os"

	"microservices/inventory/internal/app"
	"microservices/inventory/internal/config"
	"microservices/inventory/pkg/logger"
)

// @title Inventory Service API
// @version 1.0
// @description Inventory management microservice with stock reservation capabilities
// @termsOfService http://swagger.io/terms/

// @contact.name API Support
// @contact.url http://www.swagger.io/support
// @contact.email support@swagger.io

// @license.name Apache 2.0
// @license.url http://www.apache.org/licenses/LICENSE-2.0.html

// @host localhost:8083
// @BasePath /api/v1

// @securityDefinitions.apikey BearerAuth
// @in header
// @name Authorization
// @description Type "Bearer" followed by a space and JWT token.
func main() {
	configPath := flag.String("config", "config.yaml", "path to config file")
	flag.Parse()

	// Load configuration
	cfg, err := config.Load(*configPath)
	if err != nil {
		// Try environment-based config path
		if envPath := os.Getenv("CONFIG_PATH"); envPath != "" {
			cfg, err = config.Load(envPath)
		}
		if err != nil {
			logger.Fatalf("Failed to load config: %v", err)
		}
	}

	// Initialize logger
	logger.Init(cfg.App.Debug)

	logger.Infof("Starting %s in %s mode", cfg.App.Name, cfg.App.Env)

	// Create and run application
	application, err := app.New(cfg)
	if err != nil {
		logger.Fatal("Failed to create application", err)
	}

	if err := application.Run(); err != nil {
		logger.Fatal("Application error", err)
	}
}
