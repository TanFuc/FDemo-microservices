package app

import (
	"context"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/jackc/pgx/v5/pgxpool"

	"microservices/payment/internal/adapter/gateway"
	"microservices/payment/internal/adapter/publisher"
	"microservices/payment/internal/adapter/repository"
	"microservices/payment/internal/config"
	httphandler "microservices/payment/internal/handler/http"
	"microservices/payment/internal/job"
	"microservices/payment/internal/port"
	"microservices/payment/internal/router"
	"microservices/payment/internal/usecase"
	"microservices/pkg/authclient"
	"microservices/pkg/logger"
)

type App struct {
	cfg               *config.Config
	httpRouter        *router.Router
	pool              *pgxpool.Pool
	natsPublisher     *publisher.NATSPublisher
	reconciliationJob *job.ReconciliationJob
	authClient        *authclient.Client
	ctx               context.Context
	cancel            context.CancelFunc
}

func New(cfg *config.Config) (*App, error) {
	ctx, cancel := context.WithCancel(context.Background())

	// Connect to PostgreSQL
	logger.Info().Msg("Connecting to PostgreSQL...")
	pool, err := pgxpool.New(ctx, cfg.Database.URL)
	if err != nil {
		cancel()
		return nil, err
	}

	// Verify database connection
	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		cancel()
		return nil, err
	}
	logger.Info().Msg("Connected to PostgreSQL")

	// Connect to NATS
	logger.Info().Msg("Connecting to NATS...")
	natsPublisher, err := publisher.NewNATSPublisher(ctx, cfg.NATS.URL)
	if err != nil {
		pool.Close()
		cancel()
		return nil, err
	}
	logger.Info().Str("url", cfg.NATS.URL).Msg("Connected to NATS")

	// Create repository
	repo := repository.NewPostgresRepository(pool)

	// Create payment gateways
	gateways := []port.PaymentGateway{
		gateway.NewStripeAdapter(gateway.StripeConfig{
			SecretKey:     cfg.Stripe.SecretKey,
			WebhookSecret: cfg.Stripe.WebhookSecret,
		}),
		gateway.NewMoMoAdapter(gateway.MoMoConfig{
			PartnerCode: cfg.MoMo.PartnerCode,
			AccessKey:   cfg.MoMo.AccessKey,
			SecretKey:   cfg.MoMo.SecretKey,
			Endpoint:    cfg.MoMo.Endpoint,
		}),
		gateway.NewCODAdapter(),
	}

	// Add VNPay gateway if configured
	if cfg.VNPay.TmnCode != "" {
		gateways = append(gateways, gateway.NewVNPayAdapter(gateway.VNPayConfig{
			TmnCode:    cfg.VNPay.TmnCode,
			HashSecret: cfg.VNPay.HashSecret,
			PayURL:     cfg.VNPay.PayURL,
			APIURL:     cfg.VNPay.APIURL,
			ReturnURL:  cfg.VNPay.ReturnURL,
		}))
		logger.Info().Msg("VNPay gateway registered")
	}

	// Add ZaloPay gateway if configured
	if cfg.ZaloPay.AppID > 0 {
		gateways = append(gateways, gateway.NewZaloPayAdapter(gateway.ZaloPayConfig{
			AppID:    cfg.ZaloPay.AppID,
			Key1:     cfg.ZaloPay.Key1,
			Key2:     cfg.ZaloPay.Key2,
			Endpoint: cfg.ZaloPay.Endpoint,
		}))
		logger.Info().Msg("ZaloPay gateway registered")
	}

	// Create use case
	paymentUC := usecase.NewPaymentUseCase(repo, natsPublisher, gateways, nil)

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
	paymentHandler := httphandler.NewPaymentHandler(paymentUC)
	webhookHandler := httphandler.NewWebhookHandler(paymentUC)
	healthHandler := httphandler.NewHealthHandler(pool)

	// Initialize HTTP router
	httpRouter := router.NewRouter(cfg, paymentHandler, webhookHandler, healthHandler, authMiddleware)

	// Create and start reconciliation job
	reconciliationJob := job.NewReconciliationJob(
		paymentUC,
		job.ReconciliationConfig{
			Schedule:  cfg.Job.ReconciliationSchedule,
			Timeout:   cfg.Job.ReconciliationTimeout,
			OlderThan: cfg.Job.ReconciliationOlderThan,
		},
		nil,
	)
	if err := reconciliationJob.Start(); err != nil {
		natsPublisher.Close()
		pool.Close()
		cancel()
		return nil, err
	}
	logger.Info().Msg("Reconciliation job started")

	return &App{
		cfg:               cfg,
		httpRouter:        httpRouter,
		pool:              pool,
		natsPublisher:     natsPublisher,
		reconciliationJob: reconciliationJob,
		authClient:        authClient,
		ctx:               ctx,
		cancel:            cancel,
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
		addr := a.cfg.App.Address()
		logger.Info().
			Str("port", a.cfg.App.Port).
			Msg("HTTP server started")
		if err := httpApp.Listen(addr); err != nil {
			logger.Fatal().Err(err).Msg("Failed to start HTTP server")
		}
	}()

	logger.Info().
		Str("httpPort", a.cfg.App.Port).
		Msg("Payment service started")

	<-quit
	logger.Info().Msg("Shutting down servers...")

	// Cancel context
	a.cancel()

	// Stop reconciliation job
	if a.reconciliationJob != nil {
		jobCtx := a.reconciliationJob.Stop()
		<-jobCtx.Done()
		logger.Info().Msg("Reconciliation job stopped")
	}

	// Graceful shutdown with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Shutdown HTTP server
	if err := httpApp.ShutdownWithContext(ctx); err != nil {
		logger.Error().Err(err).Msg("HTTP server forced to shutdown")
	}

	// Cleanup
	a.cleanup()

	logger.Info().Msg("Payment service exited properly")
	return nil
}

func (a *App) cleanup() {
	if a.natsPublisher != nil {
		a.natsPublisher.Close()
	}
	if a.pool != nil {
		a.pool.Close()
	}
	if a.authClient != nil {
		a.authClient.Close()
	}
}

func (a *App) HttpApp() *fiber.App {
	return a.httpRouter.Setup()
}
