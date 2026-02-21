package app

import (
	"context"
	"os"
	"os/signal"
	"syscall"
	"time"

	"gorm.io/gorm"

	"microservices/template-service/internal/config"
	grpchandler "microservices/template-service/internal/handler/grpc"
	httphandler "microservices/template-service/internal/handler/http"
	"microservices/template-service/internal/repository/postgres"
	"microservices/template-service/internal/router"
	"microservices/template-service/internal/service/impl"
	"microservices/template-service/pkg/logger"
)

type App struct {
	cfg        *config.Config
	db         *gorm.DB
	httpRouter *router.Router
	grpcServer *grpchandler.Server
}

func New(cfg *config.Config) (*App, error) {
	// Initialize database
	db, err := postgres.NewPostgresDB(cfg)
	if err != nil {
		return nil, err
	}

	// Auto migrate
	if err := postgres.AutoMigrate(db); err != nil {
		return nil, err
	}

	// Initialize repository
	itemRepo := postgres.NewItemRepository(db)

	// Initialize service
	itemService := impl.NewItemService(itemRepo)

	// Initialize HTTP handler
	itemHandler := httphandler.NewItemHandler(itemService)

	// Initialize HTTP router
	httpRouter := router.NewRouter(cfg, itemHandler)

	// Initialize gRPC handler and server
	grpcHandler := grpchandler.NewItemHandler(itemService)
	grpcServer := grpchandler.NewServer(cfg, grpcHandler)

	return &App{
		cfg:        cfg,
		db:         db,
		httpRouter: httpRouter,
		grpcServer: grpcServer,
	}, nil
}

func (a *App) Run() error {
	// Setup HTTP app
	httpApp := a.httpRouter.Setup()

	// Graceful shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	// Start HTTP server
	go func() {
		logger.Info().
			Str("port", a.cfg.App.Port).
			Msg("HTTP server started")
		if err := httpApp.Listen(":" + a.cfg.App.Port); err != nil {
			logger.Fatal().Err(err).Msg("Failed to start HTTP server")
		}
	}()

	// Start gRPC server
	go func() {
		if err := a.grpcServer.Run(); err != nil {
			logger.Fatal().Err(err).Msg("Failed to start gRPC server")
		}
	}()

	logger.Info().
		Str("httpPort", a.cfg.App.Port).
		Str("grpcPort", a.cfg.App.GRPCPort).
		Msg("All servers started")

	<-quit
	logger.Info().Msg("Shutting down servers...")

	// Graceful shutdown with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Shutdown gRPC server
	a.grpcServer.GracefulStop()

	// Shutdown HTTP server
	if err := httpApp.ShutdownWithContext(ctx); err != nil {
		logger.Error().Err(err).Msg("HTTP server forced to shutdown")
	}

	// Cleanup
	a.cleanup()

	logger.Info().Msg("Servers exited properly")
	return nil
}

func (a *App) cleanup() {
	if err := postgres.Close(a.db); err != nil {
		logger.Error().Err(err).Msg("Failed to close database connection")
	}
}
