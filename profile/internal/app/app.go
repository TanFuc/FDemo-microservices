package app

import (
	"context"
	"os"
	"os/signal"
	"syscall"
	"time"

	"microservices/profile/internal/config"
	"microservices/profile/internal/event"
	httphandler "microservices/profile/internal/handler/http"
	"microservices/profile/internal/repository/mongodb"
	"microservices/profile/internal/router"
	"microservices/profile/internal/service/impl"
	"microservices/profile/pkg/logger"
)

type App struct {
	cfg        *config.Config
	db         *mongodb.MongoDB
	httpRouter *router.Router
	listener   *event.UserRegisteredListener
}

func New(cfg *config.Config) (*App, error) {
	// Initialize MongoDB
	db, err := mongodb.NewMongoDB(cfg)
	if err != nil {
		return nil, err
	}

	// Initialize repositories
	profileRepo := mongodb.NewProfileRepository(db)
	addressRepo := mongodb.NewAddressRepository(db)

	// Initialize services
	profileService := impl.NewProfileService(profileRepo, addressRepo, db.Client())

	// Initialize event listener
	listener := event.NewUserRegisteredListener(cfg, profileService)

	// Initialize HTTP handlers
	profileHandler := httphandler.NewProfileHandler(profileService)
	internalHandler := httphandler.NewInternalHandler(cfg, profileService)
	healthHandler := httphandler.NewHealthHandler(db)

	// Initialize HTTP router
	httpRouter := router.NewRouter(cfg, profileHandler, internalHandler, healthHandler)

	return &App{
		cfg:        cfg,
		db:         db,
		httpRouter: httpRouter,
		listener:   listener,
	}, nil
}

func (a *App) Run() error {
	// Setup HTTP app
	httpApp := a.httpRouter.Setup()

	// Create context for event listener
	ctx, cancel := context.WithCancel(context.Background())

	// Start event listener (non-blocking)
	safego.Go(func() {
		if err := a.listener.Start(ctx); err != nil {
			logger.Warn().Err(err).Msg("Failed to start event listener, continuing without it")
		}
	})

	// Graceful shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	// Start HTTP server
	safego.Go(func() {
		logger.Info().
			Str("port", a.cfg.App.Port).
			Msg("HTTP server started")
		if err := httpApp.Listen(":" + a.cfg.App.Port); err != nil {
			logger.Fatal().Err(err).Msg("Failed to start HTTP server")
		}
	})

	logger.Info().
		Str("httpPort", a.cfg.App.Port).
		Msg("Server started")

	<-quit
	logger.Info().Msg("Shutting down server...")

	// Cancel context to stop listener
	cancel()

	// Stop event listener
	a.listener.Stop()

	// Graceful shutdown with timeout
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer shutdownCancel()

	// Shutdown HTTP server
	if err := httpApp.ShutdownWithContext(shutdownCtx); err != nil {
		logger.Error().Err(err).Msg("HTTP server forced to shutdown")
	}

	// Cleanup
	a.cleanup(shutdownCtx)

	logger.Info().Msg("Server exited properly")
	return nil
}

func (a *App) cleanup(ctx context.Context) {
	if err := a.db.Close(ctx); err != nil {
		logger.Error().Err(err).Msg("Failed to close database connection")
	}
}
