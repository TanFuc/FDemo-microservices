package app

import (
	"context"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gofiber/fiber/v2"

	"microservices/media/internal/config"
	httphandler "microservices/media/internal/handler/http"
	"microservices/media/internal/infrastructure/queue"
	"microservices/media/internal/infrastructure/storage"
	"microservices/media/internal/router"
	"microservices/media/internal/service/impl"
	"microservices/media/internal/worker"
	"microservices/pkg/authclient"
	"microservices/pkg/logger"
)

type App struct {
	cfg         *config.Config
	httpRouter  *router.Router
	minioClient *storage.MinIOClient
	natsClient  *queue.NATSClient
	processor   *worker.ImageProcessor
	authClient  *authclient.Client
	workerCtx   context.Context
	workerCancel context.CancelFunc
}

func New(cfg *config.Config) (*App, error) {
	// Initialize MinIO client
	minioClient, err := storage.NewMinIOClient(cfg.MinIO)
	if err != nil {
		return nil, err
	}

	// Ensure bucket exists with proper configuration
	ctx := context.Background()
	if err := minioClient.EnsureBucket(ctx); err != nil {
		return nil, err
	}
	logger.Info().Msg("MinIO bucket ready")

	// Initialize NATS client
	natsClient, err := queue.NewNATSClient(cfg.NATS)
	if err != nil {
		return nil, err
	}
	logger.Info().Msg("NATS JetStream ready")

	// Initialize service
	mediaService := impl.NewMediaService(minioClient, natsClient, cfg.Media, cfg.NATS.Subject)

	// Initialize and start image processor worker
	processor := worker.NewImageProcessor(minioClient, natsClient, cfg.Media, cfg.NATS.Subject, 4)

	workerCtx, workerCancel := context.WithCancel(context.Background())
	if err := processor.Start(workerCtx); err != nil {
		workerCancel()
		natsClient.Close()
		return nil, err
	}
	logger.Info().Msg("Image processor worker started")

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
	mediaHandler := httphandler.NewMediaHandler(mediaService)

	// Initialize HTTP router
	httpRouter := router.NewRouter(cfg, mediaHandler, authMiddleware)

	return &App{
		cfg:          cfg,
		httpRouter:   httpRouter,
		minioClient:  minioClient,
		natsClient:   natsClient,
		processor:    processor,
		authClient:   authClient,
		workerCtx:    workerCtx,
		workerCancel: workerCancel,
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
		Str("grpcPort", a.cfg.App.GRPCPort).
		Msg("Media service started")

	<-quit
	logger.Info().Msg("Shutting down servers...")

	// Graceful shutdown with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Stop worker
	a.workerCancel()
	a.processor.Stop()
	logger.Info().Msg("Worker stopped")

	// Shutdown HTTP server
	if err := httpApp.ShutdownWithContext(ctx); err != nil {
		logger.Error().Err(err).Msg("HTTP server forced to shutdown")
	}

	// Cleanup
	a.cleanup()

	logger.Info().Msg("Media service exited properly")
	return nil
}

func (a *App) cleanup() {
	if a.natsClient != nil {
		a.natsClient.Close()
	}
	if a.authClient != nil {
		a.authClient.Close()
	}
}

func (a *App) HttpApp() *fiber.App {
	return a.httpRouter.Setup()
}
