package app

import (
	"context"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/redis/go-redis/v9"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"

	grpcclient "microservices/review/internal/adapter/grpc"
	"microservices/review/internal/adapter/mongodb"
	rediscache "microservices/review/internal/adapter/redis"
	"microservices/review/internal/config"
	"microservices/review/internal/core/service"
	httphandler "microservices/review/internal/handler/http"
	"microservices/review/internal/router"
	"microservices/pkg/authclient"
	"microservices/pkg/logger"
)

type App struct {
	cfg           *config.Config
	httpRouter    *router.Router
	mongoClient   *mongo.Client
	redisClient   *redis.Client
	orderClient   grpcclient.OrderServiceClient
	authClient    *authclient.Client
	ctx           context.Context
	cancel        context.CancelFunc
}

func New(cfg *config.Config) (*App, error) {
	ctx, cancel := context.WithCancel(context.Background())

	// Initialize MongoDB
	logger.Info().Msg("Connecting to MongoDB...")
	mongoClient, err := connectMongoDB(ctx, cfg.MongoDB.URI)
	if err != nil {
		cancel()
		return nil, err
	}
	mongoDB := mongoClient.Database(cfg.MongoDB.Database)
	logger.Info().Msg("Connected to MongoDB")

	// Initialize Redis
	logger.Info().Msg("Connecting to Redis...")
	redisClient := redis.NewClient(&redis.Options{
		Addr:     cfg.Redis.Addr,
		Password: cfg.Redis.Password,
		DB:       cfg.Redis.DB,
	})

	// Verify Redis connection
	pingCtx, pingCancel := context.WithTimeout(ctx, 5*time.Second)
	defer pingCancel()
	if err := redisClient.Ping(pingCtx).Err(); err != nil {
		logger.Warn().Err(err).Msg("Failed to connect to Redis, continuing without cache")
	} else {
		logger.Info().Msg("Connected to Redis")
	}

	// Initialize repositories
	reviewRepo := mongodb.NewReviewRepository(mongoDB)
	productRatingRepo := mongodb.NewProductRatingRepository(mongoDB)
	cacheRepo := rediscache.NewCacheRepository(redisClient)

	// Initialize gRPC client for Order Service
	orderClient, err := grpcclient.NewOrderServiceClient(cfg.GRPC.OrderServiceAddr)
	if err != nil {
		logger.Warn().Err(err).Msg("Failed to connect to Order Service, using mock client")
		orderClient = grpcclient.NewMockOrderServiceClient()
	} else {
		logger.Info().Str("addr", cfg.GRPC.OrderServiceAddr).Msg("Connected to Order Service")
	}

	// Initialize service
	reviewService := service.NewReviewService(reviewRepo, productRatingRepo, cacheRepo, orderClient)

	// Initialize Auth gRPC Client
	var authClient *authclient.Client
	authClient, err = authclient.NewClient(&authclient.Config{
		GRPCAddr: cfg.App.AuthGRPCAddr,
	})
	if err != nil {
		logger.Warn().Err(err).Msg("Failed to connect to Auth service (authorization will fail)")
	} else {
		logger.Info().Str("addr", cfg.App.AuthGRPCAddr).Msg("Connected to Auth gRPC Service")
	}

	// Create auth middleware for Fiber
	var authMiddleware *authclient.FiberMiddleware
	if authClient != nil {
		authMiddleware = authclient.NewFiberMiddleware(authClient)
	}

	// Initialize HTTP handlers
	reviewHandler := httphandler.NewReviewHandler(reviewService)
	healthHandler := httphandler.NewHealthHandler(mongoDB)

	// Initialize HTTP router
	httpRouter := router.NewRouter(cfg, reviewHandler, healthHandler, authMiddleware)

	return &App{
		cfg:           cfg,
		httpRouter:    httpRouter,
		mongoClient:   mongoClient,
		redisClient:   redisClient,
		orderClient:   orderClient,
		authClient:    authClient,
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
		Msg("Review service started")

	<-quit
	logger.Info().Msg("Shutting down servers...")

	// Cancel context
	a.cancel()

	// Shutdown HTTP server
	if err := httpApp.Shutdown(); err != nil {
		logger.Error().Err(err).Msg("HTTP server forced to shutdown")
	}

	// Cleanup
	a.cleanup()

	logger.Info().Msg("Review service exited properly")
	return nil
}

func (a *App) cleanup() {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if a.orderClient != nil {
		a.orderClient.Close()
	}
	if a.redisClient != nil {
		a.redisClient.Close()
	}
	if a.mongoClient != nil {
		if err := a.mongoClient.Disconnect(ctx); err != nil {
			logger.Error().Err(err).Msg("Error disconnecting from MongoDB")
		}
	}
	if a.authClient != nil {
		a.authClient.Close()
	}
}

func (a *App) HttpApp() *fiber.App {
	return a.httpRouter.Setup()
}

func connectMongoDB(ctx context.Context, uri string) (*mongo.Client, error) {
	connectCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	clientOptions := options.Client().ApplyURI(uri)
	client, err := mongo.Connect(connectCtx, clientOptions)
	if err != nil {
		return nil, err
	}

	// Verify connection
	if err := client.Ping(connectCtx, nil); err != nil {
		return nil, err
	}

	return client, nil
}
