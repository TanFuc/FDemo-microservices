package app

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"microservices/pkg/authclient"
	"microservices/wallet/internal/config"
	httphandler "microservices/wallet/internal/handler/http"
	"microservices/wallet/internal/infrastructure/database"
	"microservices/wallet/internal/infrastructure/messaging"
	"microservices/wallet/internal/infrastructure/outbox"
	"microservices/wallet/internal/infrastructure/repository"
	"microservices/wallet/internal/router"
	"microservices/wallet/internal/usecase"
)

// App represents the wallet application
type App struct {
	cfg             *config.Config
	pool            *pgxpool.Pool
	router          *router.Router
	publisher       *messaging.NATSPublisher
	userListener    *messaging.UserRegisteredListener
	paymentListener *messaging.PaymentEventListener
	outboxWorker    *outbox.OutboxWorker
	logger          *slog.Logger
}

// New creates a new App instance
func New(cfg *config.Config) (*App, error) {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))

	// Connect to database
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	pool, err := database.NewPool(ctx, &database.Config{
		URL: cfg.DB.URL,
	})
	if err != nil {
		return nil, fmt.Errorf("connect to database: %w", err)
	}
	logger.Info("Connected to database")

	// Create repository
	repo := repository.NewPostgresWalletRepository(pool)

	// Create NATS publisher
	publisher, err := messaging.NewNATSPublisher(cfg.NATS.URL)
	if err != nil {
		logger.Warn("Failed to create NATS publisher, continuing without event publishing", "error", err)
		publisher = nil
	} else {
		logger.Info("Connected to NATS")
	}

	// Create use cases
	provisionUC := usecase.NewProvisionWalletUseCase(repo, logger)
	getUC := usecase.NewGetWalletUseCase(repo, logger)
	topupUC := usecase.NewTopUpUseCase(repo, publisher, usecase.TopUpConfig{
		PaymentServiceURL: cfg.Payment.BaseURL,
		MinTopUpVND:       cfg.Wallet.MinTopUpVND,
		MaxTopUpVND:       cfg.Wallet.MaxTopUpVND,
		MaxBalanceVND:     cfg.Wallet.MaxBalanceVND,
	}, logger)
	payUC := usecase.NewPayUseCase(repo, publisher, logger)
	refundUC := usecase.NewRefundUseCase(repo, publisher, logger)

	// Create user listener
	var userListener *messaging.UserRegisteredListener
	userListener, err = messaging.NewUserRegisteredListener(cfg.NATS.URL, provisionUC, logger)
	if err != nil {
		logger.Warn("Failed to create user listener", "error", err)
	}

	// Create payment listener
	var paymentListener *messaging.PaymentEventListener
	paymentListener, err = messaging.NewPaymentEventListener(cfg.NATS.URL, topupUC, logger)
	if err != nil {
		logger.Warn("Failed to create payment listener", "error", err)
	}

	// Create outbox worker
	var outboxWorker *outbox.OutboxWorker
	if publisher != nil {
		outboxWorker = outbox.NewOutboxWorker(repo, publisher, logger, outbox.DefaultConfig())
	}

	// Create auth client
	var authMiddleware *authclient.FiberMiddleware
	authClient, err := authclient.NewClient(&authclient.Config{
		GRPCAddr: cfg.App.AuthGRPCAddr,
	})
	if err != nil {
		logger.Warn("Failed to create auth client, continuing without auth", "error", err)
	} else {
		authMiddleware = authclient.NewFiberMiddleware(authClient)
		logger.Info("Connected to auth service")
	}

	// Create handlers
	walletHandler := httphandler.NewWalletHandler(getUC, topupUC, payUC, refundUC)
	healthHandler := httphandler.NewHealthHandler(pool)

	// Create router
	r := router.NewRouter(cfg, walletHandler, healthHandler, authMiddleware)
	r.Setup()

	return &App{
		cfg:             cfg,
		pool:            pool,
		router:          r,
		publisher:       publisher,
		userListener:    userListener,
		paymentListener: paymentListener,
		outboxWorker:    outboxWorker,
		logger:          logger,
	}, nil
}

// Run starts the application
func (a *App) Run() error {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Start outbox worker
	if a.outboxWorker != nil {
		a.outboxWorker.Start(ctx)
	}

	// Start user listener
	if a.userListener != nil {
		safego.Go(func() {
			if err := a.userListener.Start(ctx); err != nil {
				a.logger.Error("User listener error", "error", err)
			}
		})
	}

	// Start payment listener
	if a.paymentListener != nil {
		safego.Go(func() {
			if err := a.paymentListener.Start(ctx); err != nil {
				a.logger.Error("Payment listener error", "error", err)
			}
		})
	}

	// Start HTTP server in goroutine
	addr := fmt.Sprintf("%s:%s", a.cfg.App.Host, a.cfg.App.Port)
	safego.Go(func() {
		a.logger.Info("Starting HTTP server", "address", addr)
		if err := a.router.App().Listen(addr); err != nil {
			a.logger.Error("HTTP server error", "error", err)
		}
	})

	// Wait for shutdown signal
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	a.logger.Info("Shutting down...")
	cancel()

	// Graceful shutdown
	a.cleanup()

	return nil
}

func (a *App) cleanup() {
	// Stop outbox worker
	if a.outboxWorker != nil {
		a.outboxWorker.Stop()
	}

	// Stop listeners
	if a.userListener != nil {
		a.userListener.Stop()
	}
	if a.paymentListener != nil {
		a.paymentListener.Stop()
	}

	// Close publisher
	if a.publisher != nil {
		a.publisher.Close()
	}

	// Close database
	if a.pool != nil {
		a.pool.Close()
	}

	// Shutdown HTTP server
	if err := a.router.App().Shutdown(); err != nil {
		a.logger.Error("Error shutting down HTTP server", "error", err)
	}

	a.logger.Info("Shutdown complete")
}
