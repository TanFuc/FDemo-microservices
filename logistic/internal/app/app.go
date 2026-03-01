package app

import (
	"context"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/jackc/pgx/v5/pgxpool"
	goredis "github.com/redis/go-redis/v9"

	rediscache "microservices/logistic/internal/adapters/cache/redis"
	"microservices/logistic/internal/adapters/messaging/nats"
	"microservices/logistic/internal/adapters/providers"
	"microservices/logistic/internal/adapters/providers/ghn"
	"microservices/logistic/internal/adapters/providers/ghtk"
	"microservices/logistic/internal/adapters/repository/postgres"
	"microservices/logistic/internal/config"
	httphandler "microservices/logistic/internal/handler/http"
	"microservices/logistic/internal/router"
	"microservices/logistic/internal/core/services"
	"microservices/pkg/authclient"
	"microservices/pkg/logger"
)

type App struct {
	cfg             *config.Config
	httpRouter      *router.Router
	dbPool          *pgxpool.Pool
	redisClient     *goredis.Client
	natsPublisher   *nats.Publisher
	natsConsumer    *nats.Consumer
	paymentConsumer *nats.PaymentConsumer
	authClient      *authclient.Client
	ctx             context.Context
	cancel          context.CancelFunc
}

func New(cfg *config.Config) (*App, error) {
	ctx, cancel := context.WithCancel(context.Background())

	// Initialize PostgreSQL connection pool
	logger.Info().Msg("Connecting to PostgreSQL...")
	dbPool, err := pgxpool.New(ctx, cfg.Database.URL)
	if err != nil {
		cancel()
		return nil, err
	}

	// Test database connection
	if err := dbPool.Ping(ctx); err != nil {
		dbPool.Close()
		cancel()
		return nil, err
	}
	logger.Info().Msg("Connected to PostgreSQL")

	// Initialize Redis client
	logger.Info().Msg("Connecting to Redis...")
	redisClient, err := rediscache.NewRedisClient(cfg.Redis.URL)
	if err != nil {
		dbPool.Close()
		cancel()
		return nil, err
	}
	logger.Info().Msg("Connected to Redis")

	// Initialize NATS publisher
	logger.Info().Msg("Connecting to NATS...")
	natsPublisher, err := nats.NewPublisher(nats.Config{
		URL:        cfg.NATS.URL,
		StreamName: cfg.NATS.StreamName,
	})
	if err != nil {
		redisClient.Close()
		dbPool.Close()
		cancel()
		return nil, err
	}
	logger.Info().Msg("Connected to NATS")

	// Initialize adapters
	shippingRepo := postgres.NewShippingRepository(dbPool)
	webhookLogRepo := postgres.NewWebhookLogRepository(dbPool)
	cache := rediscache.NewCache(redisClient)

	// Initialize provider factory
	providerCfg := providers.ProviderConfig{}
	if cfg.HasGHNCredentials() {
		providerCfg.GHN = ghn.Config{
			APIURL: cfg.GHN.APIURL,
			Token:  cfg.GHN.Token,
			ShopID: cfg.GHN.ShopID,
		}
		logger.Info().Msg("GHN provider configured")
	}
	if cfg.HasGHTKCredentials() {
		providerCfg.GHTK = ghtk.Config{
			APIURL: cfg.GHTK.APIURL,
			Token:  cfg.GHTK.Token,
		}
		logger.Info().Msg("GHTK provider configured")
	}

	providerFactory, err := providers.NewFactory(providerCfg)
	if err != nil {
		natsPublisher.Close()
		redisClient.Close()
		dbPool.Close()
		cancel()
		return nil, err
	}
	logger.Info().Msgf("Available providers: %v", providerFactory.ListProviders())

	// Initialize services
	shippingService := services.NewShippingService(
		providerFactory,
		shippingRepo,
		cache,
		natsPublisher,
	)

	webhookService := services.NewWebhookServiceWithLogging(
		providerFactory,
		shippingRepo,
		webhookLogRepo,
		natsPublisher,
	)

	healthService := services.NewHealthService(dbPool, redisClient)

	// Initialize Auth gRPC Client
	var authClient *authclient.Client
	authGRPCAddr := os.Getenv("AUTH_GRPC_ADDR")
	if authGRPCAddr == "" {
		authGRPCAddr = cfg.App.AuthGRPCAddr
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

	// Initialize HTTP handlers
	shippingHandler := httphandler.NewShippingHandler(shippingService)
	webhookHandler := httphandler.NewWebhookHandler(webhookService)
	healthHandler := httphandler.NewHealthHandler(healthService)

	// Initialize HTTP router
	httpRouter := router.NewRouter(cfg, shippingHandler, webhookHandler, healthHandler, authMiddleware)

	// Initialize NATS consumers
	var natsConsumer *nats.Consumer
	var paymentConsumer *nats.PaymentConsumer

	// Order packed event consumer
	natsConsumer, err = nats.NewConsumer(nats.ConsumerConfig{
		URL:          cfg.NATS.URL,
		StreamName:   "ORDERS",
		ConsumerName: "logistics-service",
		Subject:      "order.packed",
	}, shippingService)
	if err != nil {
		logger.Warn().Err(err).Msg("Failed to create NATS consumer")
	} else {
		if err := natsConsumer.Start(ctx, nats.ConsumerConfig{
			URL:          cfg.NATS.URL,
			ConsumerName: "logistics-service",
		}); err != nil {
			logger.Warn().Err(err).Msg("Failed to start NATS consumer")
		} else {
			logger.Info().Msg("NATS consumer started for order.packed events")
		}
	}

	// Payment event consumer
	orderFetcher := &nats.MockOrderFetcher{}
	paymentConsumer, err = nats.NewPaymentConsumer(nats.PaymentConsumerConfig{
		URL:          cfg.NATS.URL,
		ConsumerName: "logistics-payment-consumer",
	}, shippingService, orderFetcher)
	if err != nil {
		logger.Warn().Err(err).Msg("Failed to create payment consumer")
	} else {
		if err := paymentConsumer.Start(ctx, nats.PaymentConsumerConfig{
			URL:          cfg.NATS.URL,
			ConsumerName: "logistics-payment-consumer",
		}); err != nil {
			logger.Warn().Err(err).Msg("Failed to start payment consumer")
		} else {
			logger.Info().Msg("Payment consumer started for auto-shipment creation")
		}
	}

	return &App{
		cfg:             cfg,
		httpRouter:      httpRouter,
		dbPool:          dbPool,
		redisClient:     redisClient,
		natsPublisher:   natsPublisher,
		natsConsumer:    natsConsumer,
		paymentConsumer: paymentConsumer,
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
		logger.Info().
			Str("port", a.cfg.App.Port).
			Msg("HTTP server started")
		if err := httpApp.Listen(":" + a.cfg.App.Port); err != nil {
			logger.Fatal().Err(err).Msg("Failed to start HTTP server")
		}
	}()

	logger.Info().
		Str("httpPort", a.cfg.App.Port).
		Msg("Logistic service started")

	<-quit
	logger.Info().Msg("Shutting down servers...")

	// Cancel context to stop consumers
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

	logger.Info().Msg("Logistic service exited properly")
	return nil
}

func (a *App) cleanup() {
	if a.natsConsumer != nil {
		a.natsConsumer.Close()
	}
	if a.paymentConsumer != nil {
		a.paymentConsumer.Close()
	}
	if a.natsPublisher != nil {
		a.natsPublisher.Close()
	}
	if a.redisClient != nil {
		a.redisClient.Close()
	}
	if a.dbPool != nil {
		a.dbPool.Close()
	}
	if a.authClient != nil {
		a.authClient.Close()
	}
}

func (a *App) HttpApp() *fiber.App {
	return a.httpRouter.Setup()
}
