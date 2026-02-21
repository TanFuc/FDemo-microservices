package app

import (
	"context"
	"os"
	"os/signal"
	"syscall"
	"time"

	"microservices/inventory/internal/config"
	grpchandler "microservices/inventory/internal/handler/grpc"
	httphandler "microservices/inventory/internal/handler/http"
	"microservices/inventory/internal/middleware"
	"microservices/inventory/internal/model"
	"microservices/inventory/internal/repository/postgres"
	redisrepo "microservices/inventory/internal/repository/redis"
	"microservices/inventory/internal/router"
	"microservices/inventory/internal/service"
	"microservices/inventory/internal/service/impl"
	"microservices/inventory/pkg/logger"

	"github.com/gofiber/fiber/v2"
)

type App struct {
	cfg        *config.Config
	httpServer *fiber.App
	grpcServer *grpchandler.Server
}

func New(cfg *config.Config) (*App, error) {
	// Initialize PostgreSQL
	db, err := postgres.Connect(&cfg.Postgres, cfg.App.Debug)
	if err != nil {
		return nil, err
	}

	// Auto migrate
	if err := postgres.AutoMigrate(db); err != nil {
		return nil, err
	}

	// Initialize Redis
	redisClient, err := redisrepo.Connect(&cfg.Redis)
	if err != nil {
		return nil, err
	}

	// Initialize repositories
	cacheRepo := redisrepo.NewCacheRepository(redisClient)
	inventoryRepo := postgres.NewInventoryRepository(db, cacheRepo)
	reservationRepo := postgres.NewReservationRepository(db)

	// Initialize auth service
	authService, err := impl.NewAuthService(cfg.Auth.GRPCAddr, cfg.Auth.Timeout)
	if err != nil {
		logger.Warn("Failed to connect to auth service, using mock auth")
		authService = newMockAuthService()
	}

	// Initialize services
	inventoryService := impl.NewInventoryService(inventoryRepo, cacheRepo)
	reservationService := impl.NewReservationService(
		inventoryRepo,
		reservationRepo,
		cacheRepo,
		cfg.GetReservationTTL(),
	)

	// Initialize handlers
	inventoryHandler := httphandler.NewInventoryHandler(inventoryService)
	reservationHandler := httphandler.NewReservationHandler(reservationService)

	// Initialize middleware
	authMiddleware := middleware.NewAuthMiddleware(authService)

	// Setup router
	r := router.New(inventoryHandler, reservationHandler, authMiddleware)
	httpServer := r.Setup()

	// Setup gRPC server
	grpcServer, err := grpchandler.NewServer(cfg.App.GRPCPort)
	if err != nil {
		return nil, err
	}

	// Register gRPC handlers
	_ = grpchandler.NewInventoryHandler(inventoryService, reservationService)
	// Note: Register with pb generated service when available
	// pb.RegisterInventoryServiceServer(grpcServer.GetGRPCServer(), grpcHandler)

	return &App{
		cfg:        cfg,
		httpServer: httpServer,
		grpcServer: grpcServer,
	}, nil
}

func (a *App) Run() error {
	// Start gRPC server in goroutine
	go func() {
		if err := a.grpcServer.Start(); err != nil {
			logger.Error("gRPC server error", err)
		}
	}()

	// Start HTTP server in goroutine
	go func() {
		addr := ":" + a.cfg.App.Port
		logger.Infof("Starting HTTP server on %s", addr)
		if err := a.httpServer.Listen(addr); err != nil {
			logger.Error("HTTP server error", err)
		}
	}()

	// Wait for interrupt signal
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	logger.Info("Shutting down servers...")

	// Graceful shutdown
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Shutdown HTTP server
	if err := a.httpServer.ShutdownWithContext(ctx); err != nil {
		logger.Error("HTTP server shutdown error", err)
	}

	// Shutdown gRPC server
	a.grpcServer.Stop()

	logger.Info("Servers stopped")
	return nil
}

// Mock auth service for development/testing
type mockAuthService struct{}

func newMockAuthService() service.AuthService {
	return &mockAuthService{}
}

func (s *mockAuthService) VerifyToken(ctx context.Context, token string) (*service.AuthUser, error) {
	// Allow all tokens in mock mode
	return &service.AuthUser{
		UserID:      "mock-user-id",
		Email:       "mock@example.com",
		Role:        "admin",
		Permissions: []string{"inventory:create", "inventory:read", "inventory:update", "inventory:delete", "inventory:sync", "reservation:create", "reservation:read", "reservation:confirm", "reservation:release"},
	}, nil
}

func (s *mockAuthService) CheckPermission(ctx context.Context, userID string, permission string) (bool, error) {
	return true, nil
}

// Ensure mock implements the interface
var _ service.AuthService = (*mockAuthService)(nil)

// Sync inventory to Redis on startup
func syncInventoryToCache(
	ctx context.Context,
	inventoryRepo *postgres.InventoryRepository,
) {
	items, _, err := inventoryRepo.List(ctx, &model.InventoryFilter{Limit: 1000})
	if err != nil {
		logger.Error("Failed to sync inventory to cache", err)
		return
	}

	for _, item := range items {
		if err := inventoryRepo.SyncFromDB(ctx, item.SkuID); err != nil {
			logger.Errorf("Failed to sync SKU %s to cache: %v", item.SkuID, err)
		}
	}

	logger.Infof("Synced %d inventory items to cache", len(items))
}
