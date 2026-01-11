package main

import (
	"context"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/jackc/pgx/v5/pgxpool"
	"microservices/payment/internal/adapter/gateway"
	"microservices/payment/internal/adapter/publisher"
	"microservices/payment/internal/adapter/repository"
	"microservices/payment/internal/handler"
	"microservices/payment/internal/job"
	"microservices/payment/internal/port"
	"microservices/payment/internal/usecase"
	"microservices/payment/pkg/config"
)

func main() {
	// Initialize logger
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))
	slog.SetDefault(logger)

	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		logger.Error("failed to load configuration", "error", err)
		os.Exit(1)
	}

	// Create context for graceful shutdown
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Connect to PostgreSQL
	pool, err := pgxpool.New(ctx, cfg.Database.URL)
	if err != nil {
		logger.Error("failed to connect to database", "error", err)
		os.Exit(1)
	}
	defer pool.Close()

	// Verify database connection
	if err := pool.Ping(ctx); err != nil {
		logger.Error("failed to ping database", "error", err)
		os.Exit(1)
	}
	logger.Info("connected to database")

	// Connect to NATS
	natsPublisher, err := publisher.NewNATSPublisher(ctx, cfg.NATS.URL)
	if err != nil {
		logger.Error("failed to connect to NATS", "error", err)
		os.Exit(1)
	}
	defer natsPublisher.Close()
	logger.Info("connected to NATS")

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

	// Create use case
	paymentUC := usecase.NewPaymentUseCase(repo, natsPublisher, gateways, logger)

	// Create HTTP handlers
	paymentHandler := handler.NewPaymentHandler(paymentUC, logger)
	webhookHandler := handler.NewWebhookHandler(paymentUC, logger)

	// Setup router
	r := chi.NewRouter()

	// Middleware
	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	r.Use(middleware.Timeout(60 * time.Second))

	// Health check
	r.Get("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	})

	// Register routes
	paymentHandler.RegisterRoutes(r)
	webhookHandler.RegisterRoutes(r)

	// Create and start reconciliation job
	reconciliationJob := job.NewReconciliationJob(
		paymentUC,
		job.ReconciliationConfig{
			Schedule:  cfg.Job.ReconciliationSchedule,
			Timeout:   cfg.Job.ReconciliationTimeout,
			OlderThan: cfg.Job.ReconciliationOlderThan,
		},
		logger,
	)
	if err := reconciliationJob.Start(); err != nil {
		logger.Error("failed to start reconciliation job", "error", err)
		os.Exit(1)
	}

	// Create HTTP server
	server := &http.Server{
		Addr:         cfg.Server.Address(),
		Handler:      r,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	// Start server in goroutine
	go func() {
		logger.Info("starting server", "address", cfg.Server.Address())
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Error("server error", "error", err)
			cancel()
		}
	}()

	// Wait for shutdown signal
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	logger.Info("shutting down server...")

	// Stop reconciliation job
	jobCtx := reconciliationJob.Stop()
	<-jobCtx.Done()
	logger.Info("reconciliation job stopped")

	// Graceful shutdown with timeout
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer shutdownCancel()

	if err := server.Shutdown(shutdownCtx); err != nil {
		logger.Error("server shutdown error", "error", err)
	}

	logger.Info("server stopped")
}
