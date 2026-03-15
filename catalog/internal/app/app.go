package app

import (
	"context"
	"os"
	"os/signal"
	"syscall"
	"time"

	"microservices/catalog/internal/config"
	grpchandler "microservices/catalog/internal/handler/grpc"
	httphandler "microservices/catalog/internal/handler/http"
	"microservices/catalog/internal/infrastructure/nats"
	"microservices/catalog/internal/middleware"
	"microservices/catalog/internal/repository"
	"microservices/catalog/internal/repository/mongo"
	"microservices/catalog/internal/repository/redis"
	"microservices/catalog/internal/router"
	"microservices/catalog/internal/service/impl"
	"microservices/catalog/internal/worker"
	"microservices/catalog/pkg/logger"
)

type App struct {
	cfg              *config.Config
	httpRouter       *router.Router
	grpcServer       *grpchandler.Server
	eventPublisher   repository.EventPublisher
	ratingSyncWorker *worker.RatingSyncWorker
	ctx              context.Context
	cancel           context.CancelFunc
}

func New(cfg *config.Config) (*App, error) {
	// Create app-level context for lifecycle management
	appCtx, appCancel := context.WithCancel(context.Background())

	// Initialize MongoDB
	ctx, cancel := context.WithTimeout(context.Background(), cfg.MongoDB.Timeout)
	defer cancel()

	db, err := mongo.Connect(ctx, cfg.MongoDB.URI, cfg.MongoDB.Database)
	if err != nil {
		return nil, err
	}
	logger.Info().Msg("Connected to MongoDB")

	// Initialize repositories
	categoryRepo, err := mongo.NewCategoryRepository(db)
	if err != nil {
		return nil, err
	}

	brandRepo, err := mongo.NewBrandRepository(db)
	if err != nil {
		return nil, err
	}

	productRepo, err := mongo.NewProductRepository(db)
	if err != nil {
		return nil, err
	}

	// Initialize Redis cache (optional)
	var cacheRepo *redis.CacheRepository
	redisClient, err := redis.Connect(cfg.Redis.Addr, cfg.Redis.Password, cfg.Redis.DB)
	if err != nil {
		logger.Warn().Err(err).Msg("Failed to connect to Redis, continuing without cache")
	} else {
		logger.Info().Msg("Connected to Redis")
		cacheRepo = redis.NewCacheRepository(redisClient)
	}

	// Initialize Auth service (optional)
	var authService *impl.AuthServiceCloser
	authServiceImpl, err := impl.NewAuthService(cfg.Auth.GRPCAddr)
	if err != nil {
		logger.Warn().Err(err).Msg("Failed to connect to Auth service, authorization will fail")
	} else {
		logger.Info().Str("addr", cfg.Auth.GRPCAddr).Msg("Connected to Auth gRPC Service")
		authService = &impl.AuthServiceCloser{AuthService: authServiceImpl}
	}

	// Initialize NATS publisher for event streaming (optional)
	var eventPublisher repository.EventPublisher
	var natsPublisher *nats.Publisher
	natsPublisher, err = nats.NewPublisher(cfg.NATS.URL, cfg.NATS.StreamName)
	if err != nil {
		logger.Warn().Err(err).Msg("Failed to connect to NATS, product events will be disabled")
	} else {
		logger.Info().
			Str("url", cfg.NATS.URL).
			Str("stream", cfg.NATS.StreamName).
			Msg("Connected to NATS JetStream for event publishing")
		eventPublisher = natsPublisher
	}

	// Initialize services
	productService := impl.NewProductService(productRepo, categoryRepo, brandRepo, cacheRepo, eventPublisher)

	// Initialize RatingSyncWorker for consuming rating events from NATS
	var ratingSyncWorker *worker.RatingSyncWorker
	if natsPublisher != nil {
		ratingSyncWorker = worker.NewRatingSyncWorker(natsPublisher.JetStream(), productService)
		if err := ratingSyncWorker.Start(appCtx); err != nil {
			logger.Warn().Err(err).Msg("Failed to start RatingSyncWorker, rating sync will be disabled")
			ratingSyncWorker = nil
		} else {
			logger.Info().Msg("RatingSyncWorker started: listening for rating.updated events")
		}
	}
	categoryService := impl.NewCategoryService(categoryRepo, cacheRepo)
	brandService := impl.NewBrandService(brandRepo, cacheRepo)

	// Initialize HTTP handlers
	productHandler := httphandler.NewProductHandler(productService)
	categoryHandler := httphandler.NewCategoryHandler(categoryService)
	brandHandler := httphandler.NewBrandHandler(brandService)

	// Initialize auth middleware
	var authMiddleware *middleware.AuthMiddleware
	if authService != nil {
		authMiddleware = middleware.NewAuthMiddleware(authService.AuthService)
	}

	// Initialize HTTP router
	httpRouter := router.NewRouter(cfg, productHandler, categoryHandler, brandHandler, authMiddleware)

	// Initialize gRPC handler and server
	var grpcAuthService impl.AuthService
	if authService != nil {
		grpcAuthService = authService.AuthService
	}
	grpcHandler := grpchandler.NewCatalogHandler(productService, categoryService, brandService, grpcAuthService)
	grpcServer := grpchandler.NewServer(cfg, grpcHandler)

	return &App{
		cfg:              cfg,
		httpRouter:       httpRouter,
		grpcServer:       grpcServer,
		eventPublisher:   eventPublisher,
		ratingSyncWorker: ratingSyncWorker,
		ctx:              appCtx,
		cancel:           appCancel,
	}, nil
}

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

	// Cancel app context to signal workers to stop
	a.cancel()

	// Graceful shutdown with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Stop RatingSyncWorker
	if a.ratingSyncWorker != nil {
		a.ratingSyncWorker.Stop()
		logger.Info().Msg("RatingSyncWorker stopped")
	}

	// Shutdown gRPC server
	a.grpcServer.GracefulStop()

	// Shutdown HTTP server
	if err := httpApp.ShutdownWithContext(ctx); err != nil {
		logger.Error().Err(err).Msg("HTTP server forced to shutdown")
	}

	// Close NATS publisher
	if a.eventPublisher != nil {
		if err := a.eventPublisher.Close(); err != nil {
			logger.Error().Err(err).Msg("Failed to close NATS publisher")
		} else {
			logger.Info().Msg("NATS publisher closed")
		}
	}

	logger.Info().Msg("Servers exited properly")
	return nil
}
