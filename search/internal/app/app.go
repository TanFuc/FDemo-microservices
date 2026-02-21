package app

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gofiber/fiber/v2"

	"microservices/search/internal/config"
	httphandler "microservices/search/internal/handler/http"
	"microservices/search/internal/infrastructure/cache"
	"microservices/search/internal/infrastructure/elastic"
	"microservices/search/internal/router"
	"microservices/search/internal/usecase"
	"microservices/pkg/logger"
)

type App struct {
	cfg           *config.Config
	httpRouter    *router.Router
	elasticClient *elastic.Client
	redisClient   *cache.RedisClient
	ctx           context.Context
	cancel        context.CancelFunc
}

func New(cfg *config.Config) (*App, error) {
	ctx, cancel := context.WithCancel(context.Background())

	// Initialize slog for internal components
	slogger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelDebug,
	}))

	// Initialize Elasticsearch client
	logger.Info().Msg("Connecting to Elasticsearch...")
	elasticClient, err := elastic.NewClient(cfg.Elasticsearch.Addresses, slogger)
	if err != nil {
		cancel()
		return nil, err
	}

	// Ensure index exists with proper mapping
	if err := elasticClient.EnsureIndex(ctx); err != nil {
		elasticClient.Close()
		cancel()
		return nil, err
	}
	logger.Info().Msg("Connected to Elasticsearch")

	// Initialize Redis client
	logger.Info().Msg("Connecting to Redis...")
	redisClient, err := cache.NewRedisClient(cfg.Redis.Addr, cfg.Redis.Password, cfg.Redis.DB, slogger)
	if err != nil {
		elasticClient.Close()
		cancel()
		return nil, err
	}
	logger.Info().Msg("Connected to Redis")

	// Initialize usecase
	searchUsecase := usecase.NewSearchUsecase(elasticClient, redisClient, slogger)

	// Initialize HTTP handlers
	searchHandler := httphandler.NewSearchHandler(searchUsecase)
	healthHandler := httphandler.NewHealthHandler()

	// Initialize HTTP router
	httpRouter := router.NewRouter(cfg, searchHandler, healthHandler)

	return &App{
		cfg:           cfg,
		httpRouter:    httpRouter,
		elasticClient: elasticClient,
		redisClient:   redisClient,
		ctx:           ctx,
		cancel:        cancel,
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
		addr := a.cfg.App.Host + ":" + a.cfg.App.Port
		logger.Info().
			Str("port", a.cfg.App.Port).
			Msg("HTTP server started")
		if err := httpApp.Listen(addr); err != nil {
			logger.Fatal().Err(err).Msg("Failed to start HTTP server")
		}
	}()

	logger.Info().
		Str("httpPort", a.cfg.App.Port).
		Msg("Search service started")

	<-quit
	logger.Info().Msg("Shutting down servers...")

	// Cancel context
	a.cancel()

	// Graceful shutdown with timeout
	if err := httpApp.ShutdownWithTimeout(30 * time.Second); err != nil {
		logger.Error().Err(err).Msg("HTTP server forced to shutdown")
	}

	// Cleanup
	a.cleanup()

	logger.Info().Msg("Search service exited properly")
	return nil
}

func (a *App) cleanup() {
	if a.redisClient != nil {
		a.redisClient.Close()
	}
	if a.elasticClient != nil {
		a.elasticClient.Close()
	}
}

func (a *App) HttpApp() *fiber.App {
	return a.httpRouter.Setup()
}
