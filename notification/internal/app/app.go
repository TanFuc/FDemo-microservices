package app

import (
	"context"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/redis/go-redis/v9"

	"microservices/notification/internal/adapter"
	"microservices/notification/internal/bridge"
	"microservices/notification/internal/config"
	httphandler "microservices/notification/internal/handler/http"
	wshandler "microservices/notification/internal/handler/websocket"
	"microservices/notification/internal/infrastructure"
	"microservices/notification/internal/provider"
	"microservices/notification/internal/router"
	"microservices/notification/internal/service/impl"
	"microservices/notification/internal/websocket"
	"microservices/notification/internal/worker"
	"microservices/pkg/authclient"
	"microservices/pkg/logger"
t"microservices/pkg/safego"
)

type App struct {
	cfg           *config.Config
	httpRouter    *router.Router
	wsManager     *websocket.Manager
	rabbitmq      *infrastructure.RabbitMQ
	natsConn      *infrastructure.NATS
	mongodb       *infrastructure.MongoDB
	redisClient   *redis.Client
	orderListener *bridge.OrderEventListener
	emailConsumer *worker.EmailConsumer
	reviewWorker  *worker.ReviewWorker
	authClient    *authclient.Client
	ctx           context.Context
	cancel        context.CancelFunc
}

func New(cfg *config.Config) (*App, error) {
	ctx, cancel := context.WithCancel(context.Background())

	// Initialize RabbitMQ
	rabbitmq, err := infrastructure.NewRabbitMQ(cfg.RabbitMQ.URL)
	if err != nil {
		cancel()
		return nil, err
	}
	logger.Info().Msg("RabbitMQ connected")

	// Initialize NATS
	natsConn, err := infrastructure.NewNATS(cfg.NATS.URL)
	if err != nil {
		rabbitmq.Close()
		cancel()
		return nil, err
	}
	logger.Info().Msg("NATS connected")

	// Initialize MongoDB
	mongodb, err := infrastructure.NewMongoDB(cfg.MongoDB.URI, cfg.MongoDB.Database)
	if err != nil {
		natsConn.Close()
		rabbitmq.Close()
		cancel()
		return nil, err
	}
	logger.Info().Msg("MongoDB connected")

	// Get templates directory path
	execPath, err := os.Executable()
	if err != nil {
		mongodb.Close(context.Background())
		natsConn.Close()
		rabbitmq.Close()
		cancel()
		return nil, err
	}
	templatesDir := filepath.Join(filepath.Dir(execPath), "..", "templates")

	// Try current working directory if templates not found
	if _, err := os.Stat(templatesDir); os.IsNotExist(err) {
		cwd, _ := os.Getwd()
		templatesDir = filepath.Join(cwd, "templates")
	}

	// Initialize template engine
	templateEngine, err := provider.NewTemplateEngine(templatesDir)
	if err != nil {
		logger.Warn().Err(err).Msg("Failed to initialize template engine")
	} else {
		// Preload templates
		if err := templateEngine.PreloadTemplates(); err != nil {
			logger.Warn().Err(err).Msg("Failed to preload templates")
		}
	}

	// Initialize email provider
	var emailProvider provider.EmailProvider
	if cfg.SMTP.User == "" || cfg.SMTP.Pass == "" {
		logger.Info().Msg("SMTP credentials not configured, using mock email provider")
		emailProvider = provider.NewMockEmailProvider()
	} else {
		emailProvider = provider.NewSMTPEmailProvider(cfg.SMTP)
	}

	// Initialize WebSocket manager
	var wsManager *websocket.Manager
	if cfg.WebSocket.Enabled {
		wsManager = websocket.NewManager(cfg.WebSocket)
		go wsManager.Start()
		logger.Info().Msg("WebSocket manager started")
	}

	// Initialize and start the bridge (NATS -> RabbitMQ)
	orderListener := bridge.NewOrderEventListener(natsConn, rabbitmq)
	if err := orderListener.Start(ctx); err != nil {
		if wsManager != nil {
			wsManager.Stop()
		}
		mongodb.Close(context.Background())
		natsConn.Close()
		rabbitmq.Close()
		cancel()
		return nil, err
	}
	logger.Info().Msg("Order event listener started")

	// Initialize and start the email consumer (RabbitMQ -> Email)
	emailConsumer := worker.NewEmailConsumer(rabbitmq, mongodb, emailProvider, templateEngine)
	if err := emailConsumer.Start(ctx); err != nil {
		orderListener.Stop()
		if wsManager != nil {
			wsManager.Stop()
		}
		mongodb.Close(context.Background())
		natsConn.Close()
		rabbitmq.Close()
		cancel()
		return nil, err
	}
	logger.Info().Msg("Email consumer started")

	// Initialize Redis for idempotency
	redisClient := redis.NewClient(&redis.Options{
		Addr:     cfg.Redis.Addr,
		Password: cfg.Redis.Password,
		DB:       cfg.Redis.DB,
	})
	// Test Redis connection
	if err := redisClient.Ping(ctx).Err(); err != nil {
		logger.Warn().Err(err).Msg("Failed to connect to Redis, idempotency checks will be disabled")
		redisClient = nil
	} else {
		logger.Info().Msg("Redis connected for idempotency")
	}

	// Initialize Catalog client
	catalogClient := adapter.NewHTTPCatalogClient(cfg.Catalog.URL, cfg.Catalog.ServiceKey)

	// Initialize notification service (needed for review worker)
	notificationService := impl.NewNotificationService(wsManager)

	// Initialize idempotency store
	var idempotencyStore *worker.IdempotencyStore
	if redisClient != nil {
		idempotencyStore = worker.NewIdempotencyStore(redisClient)
	}

	// Initialize and start the review worker (NATS -> Notification)
	reviewWorker := worker.NewReviewWorker(natsConn, notificationService, catalogClient, idempotencyStore)
	if err := reviewWorker.Start(ctx); err != nil {
		logger.Warn().Err(err).Msg("Failed to start ReviewWorker, review notifications will be disabled")
	} else {
		logger.Info().Msg("Review worker started")
	}

	// Initialize Auth gRPC Client
	var authClient *authclient.Client
	authGRPCAddr := os.Getenv("AUTH_GRPC_ADDR")
	if authGRPCAddr == "" {
		authGRPCAddr = cfg.App.AuthGRPCAddr
	}
	if authGRPCAddr == "" {
		authGRPCAddr = "localhost:50051"
	}

	authClient, err = authclient.NewClient(&authclient.Config{
		GRPCAddr: authGRPCAddr,
	})
	if err != nil {
		logger.Warn().Err(err).Msg("Failed to connect to Auth service (authorization will fail)")
	} else {
		logger.Info().Str("addr", authGRPCAddr).Msg("Connected to Auth gRPC Service")
	}

	// Create auth middleware for Fiber
	var authMiddleware *authclient.FiberMiddleware
	if authClient != nil {
		authMiddleware = authclient.NewFiberMiddleware(authClient)
	}

	// Initialize HTTP handler
	notificationHandler := httphandler.NewNotificationHandler(notificationService)

	// Initialize WebSocket handler
	var wsHandler *wshandler.Handler
	if wsManager != nil {
		wsHandler = wshandler.NewHandler(wsManager)
	}

	// Initialize HTTP router
	httpRouter := router.NewRouter(cfg, notificationHandler, wsHandler, authMiddleware)

	return &App{
		cfg:           cfg,
		httpRouter:    httpRouter,
		wsManager:     wsManager,
		rabbitmq:      rabbitmq,
		natsConn:      natsConn,
		mongodb:       mongodb,
		redisClient:   redisClient,
		orderListener: orderListener,
		emailConsumer: emailConsumer,
		reviewWorker:  reviewWorker,
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
		Bool("websocketEnabled", a.cfg.WebSocket.Enabled).
		Msg("Notification service started")

	<-quit
	logger.Info().Msg("Shutting down servers...")

	// Cancel context to stop workers
	a.cancel()

	// Graceful shutdown with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Stop workers
	if a.reviewWorker != nil {
		a.reviewWorker.Stop()
		logger.Info().Msg("Review worker stopped")
	}

	if a.emailConsumer != nil {
		a.emailConsumer.Stop()
		logger.Info().Msg("Email consumer stopped")
	}

	if a.orderListener != nil {
		a.orderListener.Stop()
		logger.Info().Msg("Order listener stopped")
	}

	// Stop WebSocket manager
	if a.wsManager != nil {
		a.wsManager.Stop()
		logger.Info().Msg("WebSocket manager stopped")
	}

	// Shutdown HTTP server
	if err := httpApp.ShutdownWithContext(ctx); err != nil {
		logger.Error().Err(err).Msg("HTTP server forced to shutdown")
	}

	// Cleanup
	a.cleanup()

	logger.Info().Msg("Notification service exited properly")
	return nil
}

func (a *App) cleanup() {
	if a.redisClient != nil {
		a.redisClient.Close()
	}
	if a.mongodb != nil {
		a.mongodb.Close(context.Background())
	}
	if a.natsConn != nil {
		a.natsConn.Close()
	}
	if a.rabbitmq != nil {
		a.rabbitmq.Close()
	}
	if a.authClient != nil {
		a.authClient.Close()
	}
}

func (a *App) HttpApp() *fiber.App {
	return a.httpRouter.Setup()
}

func (a *App) WebSocketManager() *websocket.Manager {
	return a.wsManager
}
