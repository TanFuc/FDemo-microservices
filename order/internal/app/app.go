package app

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gofiber/fiber/v2"

	"microservices/order/internal/config"
	httphandler "microservices/order/internal/handler/http"
	"microservices/order/internal/infrastructure/database"
	inventorygrpc "microservices/order/internal/infrastructure/grpc"
	"microservices/order/internal/infrastructure/messaging"
	"microservices/order/internal/infrastructure/repository"
	"microservices/order/internal/router"
	"microservices/order/internal/usecase"
	"microservices/pkg/authclient"
	"microservices/pkg/cache"
	"microservices/pkg/cache/redis"
	"microservices/pkg/logger"
)

type App struct {
	cfg             *config.Config
	httpRouter      *router.Router
	db              *database.Database
	cacheClient     cache.Cache
	inventoryClient *inventorygrpc.InventoryClient
	natsPublisher   *messaging.NATSPublisher
	paymentListener *messaging.PaymentEventListener
	authClient      *authclient.Client
	ctx             context.Context
	cancel          context.CancelFunc
}

func New(cfg *config.Config) (*App, error) {
	ctx, cancel := context.WithCancel(context.Background())

	// Initialize database
	logger.Info().Msg("Connecting to PostgreSQL...")
	db, err := database.NewDatabase(&cfg.Database)
	if err != nil {
		cancel()
		return nil, err
	}

	// Run migrations
	if err := db.AutoMigrate(); err != nil {
		db.Close()
		cancel()
		return nil, err
	}
	logger.Info().Msg("Connected to PostgreSQL")

	// Load shared packages configuration
	pkgCfg := config.LoadSharedPackagesConfig()

	// Initialize cache (shared package)
	var cacheClient cache.Cache
	if pkgCfg.Cache.Enabled {
		cacheCfg := pkgCfg.Cache.ToCacheConfig()
		cacheClient, err = redis.New(cacheCfg.Redis, &redis.Options{
			Metrics: true,
			Prefix:  cacheCfg.ServiceName + ":",
		})
		if err != nil {
			logger.Warn().Err(err).Msg("Failed to connect to cache, continuing without cache")
		} else {
			logger.Info().Msg("Connected to Redis cache")
		}
	}

	// Initialize inventory gRPC client
	inventoryClient, err := inventorygrpc.NewInventoryClient(&cfg.Inventory)
	if err != nil {
		logger.Warn().Err(err).Msg("Failed to connect to inventory service")
	} else {
		logger.Info().Str("addr", cfg.Inventory.GRPCAddress).Msg("Connected to Inventory gRPC Service")
	}

	// Initialize NATS publisher
	natsPublisher, err := messaging.NewNATSPublisher(&cfg.NATS)
	if err != nil {
		logger.Warn().Err(err).Msg("Failed to connect to NATS")
	} else {
		logger.Info().Str("url", cfg.NATS.URL).Msg("Connected to NATS")
	}

	// Initialize repository with cache
	orderRepo := repository.NewOrderRepository(db, cacheClient)

	// Create mock implementations for development if services are unavailable
	var stockReserver inventorygrpc.StockReserver = inventoryClient
	if inventoryClient == nil {
		stockReserver = &mockStockReserver{}
		logger.Info().Msg("Using mock stock reserver")
	}

	var eventPublisher messaging.EventPublisher = natsPublisher
	if natsPublisher == nil {
		eventPublisher = &mockEventPublisher{}
		logger.Info().Msg("Using mock event publisher")
	}

	// Initialize use cases
	createOrderUC := usecase.NewCreateOrderUseCase(orderRepo, stockReserver, eventPublisher)
	cancelOrderUC := usecase.NewCancelOrderUseCase(orderRepo, stockReserver, eventPublisher)
	getOrderUC := usecase.NewGetOrderUseCase(orderRepo)
	listOrdersUC := usecase.NewListOrdersUseCase(orderRepo)
	markAsPaidUC := usecase.NewMarkAsPaidUseCase(orderRepo, eventPublisher)
	markAsShippedUC := usecase.NewMarkAsShippedUseCase(orderRepo, eventPublisher)
	markAsCompletedUC := usecase.NewMarkAsCompletedUseCase(orderRepo, eventPublisher)

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
	healthHandler := httphandler.NewHealthHandler(db)
	orderHandler := httphandler.NewOrderHandler(
		createOrderUC,
		cancelOrderUC,
		getOrderUC,
		listOrdersUC,
		markAsPaidUC,
		markAsShippedUC,
		markAsCompletedUC,
		authMiddleware,
	)

	// Initialize HTTP router
	httpRouter := router.NewRouter(cfg, orderHandler, healthHandler, authMiddleware)

	// Initialize structured logger for payment listener
	slogger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))

	// Create stock confirmer for payment listener
	var stockConfirmer inventorygrpc.StockConfirmer = inventoryClient
	if inventoryClient == nil {
		stockConfirmer = &mockStockConfirmer{}
		logger.Info().Msg("Using mock stock confirmer for payment listener")
	}

	// Initialize payment event listener
	var paymentListener *messaging.PaymentEventListener
	paymentListener, err = messaging.NewPaymentEventListener(
		&cfg.NATS,
		orderRepo,
		stockConfirmer,
		eventPublisher,
		slogger,
	)
	if err != nil {
		logger.Warn().Err(err).Msg("Failed to create payment listener")
	}

	// Start payment listener in background
	if paymentListener != nil {
		go func() {
			if err := paymentListener.Start(ctx); err != nil {
				logger.Error().Err(err).Msg("Payment listener error")
			}
		}()
		logger.Info().Msg("Payment event listener started")
	}

	return &App{
		cfg:             cfg,
		httpRouter:      httpRouter,
		db:              db,
		cacheClient:     cacheClient,
		inventoryClient: inventoryClient,
		natsPublisher:   natsPublisher,
		paymentListener: paymentListener,
		authClient:      authClient,
		ctx:             ctx,
		cancel:          cancel,
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
		Msg("Order service started")

	<-quit
	logger.Info().Msg("Shutting down servers...")

	// Cancel context to stop listeners
	a.cancel()

	// Graceful shutdown with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Shutdown HTTP server
	if err := httpApp.ShutdownWithContext(ctx); err != nil {
		logger.Error().Err(err).Msg("HTTP server forced to shutdown")
	}

	// Cleanup
	a.cleanup()

	logger.Info().Msg("Order service exited properly")
	return nil
}

func (a *App) cleanup() {
	if a.paymentListener != nil {
		a.paymentListener.Stop()
	}
	if a.natsPublisher != nil {
		a.natsPublisher.Close()
	}
	if a.inventoryClient != nil {
		a.inventoryClient.Close()
	}
	if a.cacheClient != nil {
		a.cacheClient.Close()
	}
	if a.db != nil {
		a.db.Close()
	}
	if a.authClient != nil {
		a.authClient.Close()
	}
}

func (a *App) HttpApp() *fiber.App {
	return a.httpRouter.Setup()
}

// Mock implementations for development
type mockStockReserver struct{}

func (m *mockStockReserver) ReserveStock(ctx context.Context, skuID string, quantity int, orderID string) (string, error) {
	logger.Debug().
		Str("sku", skuID).
		Int("qty", quantity).
		Str("orderId", orderID).
		Msg("[MOCK] Reserving stock")
	return "mock-reservation-" + skuID, nil
}

func (m *mockStockReserver) ReleaseStock(ctx context.Context, reservationID string) error {
	logger.Debug().
		Str("reservationId", reservationID).
		Msg("[MOCK] Releasing stock")
	return nil
}

type mockEventPublisher struct{}

func (m *mockEventPublisher) PublishOrderCreated(ctx context.Context, event *messaging.OrderCreatedEvent) error {
	logger.Debug().Str("orderId", event.OrderID.String()).Msg("[MOCK] Publishing order.created")
	return nil
}

func (m *mockEventPublisher) PublishOrderCancelled(ctx context.Context, event *messaging.OrderCancelledEvent) error {
	logger.Debug().Str("orderId", event.OrderID.String()).Msg("[MOCK] Publishing order.cancelled")
	return nil
}

func (m *mockEventPublisher) PublishOrderPaid(ctx context.Context, event *messaging.OrderPaidEvent) error {
	logger.Debug().Str("orderId", event.OrderID.String()).Msg("[MOCK] Publishing order.paid")
	return nil
}

func (m *mockEventPublisher) PublishOrderShipped(ctx context.Context, event *messaging.OrderShippedEvent) error {
	logger.Debug().Str("orderId", event.OrderID.String()).Msg("[MOCK] Publishing order.shipped")
	return nil
}

func (m *mockEventPublisher) PublishOrderCompleted(ctx context.Context, event *messaging.OrderCompletedEvent) error {
	logger.Debug().Str("orderId", event.OrderID.String()).Msg("[MOCK] Publishing order.completed")
	return nil
}

// Mock stock confirmer for development
type mockStockConfirmer struct{}

func (m *mockStockConfirmer) ConfirmStock(ctx context.Context, orderID string) error {
	logger.Debug().Str("orderId", orderID).Msg("[MOCK] Confirming stock")
	return nil
}

func (m *mockStockConfirmer) ReleaseStockByOrderID(ctx context.Context, orderID string) error {
	logger.Debug().Str("orderId", orderID).Msg("[MOCK] Releasing stock by order")
	return nil
}
