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
	"microservices/catalog/internal/middleware"
	"microservices/catalog/internal/repository/mongo"
	"microservices/catalog/internal/repository/redis"
	"microservices/catalog/internal/router"
	"microservices/catalog/internal/service/impl"
	"microservices/catalog/pkg/logger"
)

type App struct {
	cfg        *config.Config
	httpRouter *router.Router
	grpcServer *grpchandler.Server
}

func New(cfg *config.Config) (*App, error) {
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

	// Initialize services
	productService := impl.NewProductService(productRepo, categoryRepo, brandRepo, cacheRepo, nil)
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
		cfg:        cfg,
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

	logger.Info().Msg("Servers exited properly")
	return nil
}
