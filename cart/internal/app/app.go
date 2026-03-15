package app

import (
	"context"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/redis/go-redis/v9"
	"go.mongodb.org/mongo-driver/mongo"

	"microservices/cart/internal/adapter"
	"microservices/cart/internal/config"
	grpchandler "microservices/cart/internal/handler/grpc"
	httphandler "microservices/cart/internal/handler/http"
	"microservices/cart/internal/middleware"
	mongorepo "microservices/cart/internal/repository/mongo"
	redisrepo "microservices/cart/internal/repository/redis"
	"microservices/cart/internal/router"
	"microservices/cart/internal/service/impl"
	"microservices/cart/pkg/logger"
	"microservices/pkg/authclient"
)

// App represents the application.
type App struct {
	cfg         *config.Config
	redisClient *redis.Client
	mongoClient *mongo.Client
	httpRouter  *router.Router
	grpcServer  *grpchandler.Server
	authClient  *authclient.Client
}

// New creates a new application instance.
func New(cfg *config.Config) (*App, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Initialize Redis
	logger.Info().Msg("Connecting to Redis...")
	redisClient, err := redisrepo.NewClient(&cfg.Redis)
	if err != nil {
		return nil, err
	}
	logger.Info().Msg("Connected to Redis")

	// Initialize MongoDB
	logger.Info().Msg("Connecting to MongoDB...")
	mongoClient, err := mongorepo.NewClient(ctx, &cfg.Mongo)
	if err != nil {
		redisrepo.Close(redisClient)
		return nil, err
	}
	logger.Info().Msg("Connected to MongoDB")

	// Initialize repositories
	redisRepo := redisrepo.NewCartRepository(redisClient)
	mongoDB := mongorepo.GetDatabase(mongoClient, cfg.Mongo.Database)
	mongoRepo := mongorepo.NewCartRepository(mongoDB)

	// Initialize Campaign Client
	var campaignClient adapter.CampaignClient
	if cfg.Campaign.HTTPURL != "" {
		campaignClient = adapter.NewHTTPCampaignClient(adapter.CampaignClientConfig{
			BaseURL:    cfg.Campaign.HTTPURL,
			Timeout:    cfg.Campaign.Timeout,
			ServiceKey: cfg.Campaign.ServiceKey,
		})
		logger.Info().Str("url", cfg.Campaign.HTTPURL).Msg("Campaign client initialized")
	} else {
		campaignClient = &adapter.NoCampaignClient{}
		logger.Warn().Msg("Campaign service URL not configured, voucher features disabled")
	}

	// Initialize Order Client
	var orderClient adapter.OrderClient
	if cfg.Order.HTTPURL != "" {
		orderClient = adapter.NewHTTPOrderClient(adapter.OrderClientConfig{
			BaseURL:    cfg.Order.HTTPURL,
			Timeout:    cfg.Order.Timeout,
			ServiceKey: cfg.Order.ServiceKey,
		})
		logger.Info().Str("url", cfg.Order.HTTPURL).Msg("Order client initialized")
	} else {
		orderClient = &adapter.NoOrderClient{}
		logger.Warn().Msg("Order service URL not configured, checkout feature disabled")
	}

	// Initialize service
	cartService := impl.NewCartService(redisRepo, mongoRepo, campaignClient, orderClient)

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

	// Initialize HTTP handler
	cartHandler := httphandler.NewCartHandler(cartService)

	// Initialize HTTP router
	httpRouter := router.NewRouter(cfg, cartHandler, authMiddleware)

	// Initialize gRPC handler and server
	grpcHandler := grpchandler.NewCartHandler(cartService)
	grpcServer := grpchandler.NewServer(cfg, grpcHandler)

	return &App{
		cfg:         cfg,
		redisClient: redisClient,
		mongoClient: mongoClient,
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
	a.cleanup(ctx)

	logger.Info().Msg("Servers exited properly")
	return nil
}

func (a *App) cleanup(ctx context.Context) {
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

	// Close MongoDB
	if err := mongorepo.Close(ctx, a.mongoClient); err != nil {
		logger.Error().Err(err).Msg("Failed to close MongoDB connection")
	}
}
