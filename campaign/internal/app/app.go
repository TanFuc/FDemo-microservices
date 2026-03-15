package app

import (
	"context"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"

	"microservices/campaign/internal/config"
	grpchandler "microservices/campaign/internal/handler/grpc"
	httphandler "microservices/campaign/internal/handler/http"
	"microservices/campaign/internal/middleware"
	"microservices/campaign/internal/repository/postgres"
	redisrepo "microservices/campaign/internal/repository/redis"
	"microservices/campaign/internal/router"
	"microservices/campaign/internal/service/impl"
	"microservices/campaign/pkg/logger"
	"microservices/pkg/authclient"
)

// App represents the application.
type App struct {
	cfg         *config.Config
	pgPool      *pgxpool.Pool
	redisClient *redis.Client
	httpRouter  *router.Router
	grpcServer  *grpchandler.Server
	authClient  *authclient.Client
}

// New creates a new application instance.
func New(cfg *config.Config) (*App, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Initialize PostgreSQL
	logger.Info().Msg("Connecting to PostgreSQL...")
	pgPool, err := postgres.NewPool(ctx, &cfg.Postgres)
	if err != nil {
		return nil, err
	}
	logger.Info().Msg("Connected to PostgreSQL")

	// Initialize Redis
	logger.Info().Msg("Connecting to Redis...")
	redisClient, err := redisrepo.NewClient(ctx, &cfg.Redis)
	if err != nil {
		postgres.Close(pgPool)
		return nil, err
	}
	logger.Info().Msg("Connected to Redis")

	// Initialize repositories
	campaignRepo := postgres.NewCampaignRepository(pgPool)
	voucherRepo := postgres.NewVoucherRepository(pgPool)
	userVoucherRepo := postgres.NewUserVoucherRepository(pgPool)
	voucherCacheRepo := redisrepo.NewVoucherCacheRepository(redisClient)

	// Initialize services
	campaignService := impl.NewCampaignService(campaignRepo)
	voucherService := impl.NewVoucherService(voucherRepo, userVoucherRepo, voucherCacheRepo)

	// Initialize Auth gRPC Client
	var authMiddleware *middleware.AuthMiddleware
	authClient, err := authclient.NewClient(&authclient.Config{
		GRPCAddr: cfg.Auth.GRPCAddr,
		Timeout:  cfg.Auth.Timeout,
	})
	if err != nil {
		logger.Warn().Err(err).Msg("Failed to connect to Auth service (authorization will fail)")
	} else {
		logger.Info().Str("addr", cfg.Auth.GRPCAddr).Msg("Connected to Auth gRPC Service")
		authMiddleware = middleware.NewAuthMiddleware(authClient)
	}

	// Initialize HTTP handlers
	campaignHandler := httphandler.NewCampaignHandler(campaignService)
	voucherHandler := httphandler.NewVoucherHandler(voucherService)

	// Initialize HTTP router
	httpRouter := router.NewRouter(cfg, campaignHandler, voucherHandler, authMiddleware)

	// Initialize gRPC handler and server
	grpcHandler := grpchandler.NewCampaignHandler(campaignService, voucherService)
	grpcServer := grpchandler.NewServer(cfg, grpcHandler)

	return &App{
		cfg:         cfg,
		pgPool:      pgPool,
		redisClient: redisClient,
		httpRouter:  httpRouter,
		grpcServer:  grpcServer,
		authClient:  authClient,
	}, nil
}

// Run starts the application.
func (a *App) Run() error {
	// Setup HTTP app
	httpApp := a.httpRouter.Setup()

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

	// Start gRPC server
	safego.Go(func() {
		if err := a.grpcServer.Run(); err != nil {
			logger.Fatal().Err(err).Msg("Failed to start gRPC server")
		}
	})

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
	// Close Auth client
	if a.authClient != nil {
		if err := a.authClient.Close(); err != nil {
			logger.Error().Err(err).Msg("Failed to close Auth client")
		}
	}

	// Close Redis
	if err := redisrepo.Close(a.redisClient); err != nil {
		logger.Error().Err(err).Msg("Failed to close Redis connection")
	}

	// Close PostgreSQL
	postgres.Close(a.pgPool)
}
