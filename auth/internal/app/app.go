package app

import (
	"context"
	"os"
	"os/signal"
	"syscall"
	"time"

	"gorm.io/gorm"

	"microservices/auth/internal/cache"
	"microservices/auth/internal/config"
	grpchandler "microservices/auth/internal/handler/grpc"
	httphandler "microservices/auth/internal/handler/http"
	"microservices/auth/internal/middleware"
	"microservices/auth/internal/queue"
	"microservices/auth/internal/repository"
	"microservices/auth/internal/repository/postgres"
	"microservices/auth/internal/service"
	"microservices/auth/pkg/logger"
)

type App struct {
	cfg         *config.Config
	db          *gorm.DB
	redisClient *cache.RedisClient
	natsClient  *queue.NATSClient
	httpRouter  *httphandler.Router
	grpcServer  *grpchandler.Server
}

func New(cfg *config.Config) (*App, error) {
	// Initialize database
	db, err := postgres.NewPostgresDB(cfg)
	if err != nil {
		return nil, err
	}

	// Auto migrate
	if err := postgres.AutoMigrate(db); err != nil {
		return nil, err
	}

	// Initialize Redis
	redisClient, err := cache.NewRedisClient(cfg)
	if err != nil {
		logger.Warn().Err(err).Msg("Failed to connect to Redis, continuing without cache")
	}

	// Initialize NATS (optional)
	var natsClient *queue.NATSClient
	natsClient, err = queue.NewNATSClient(cfg)
	if err != nil {
		logger.Warn().Err(err).Msg("Failed to connect to NATS, continuing without event publishing")
		natsClient = nil
	}

	// Initialize repositories
	repos := initRepositories(db)

	// Initialize services
	services := initServices(cfg, repos, redisClient, natsClient)

	// Seed database
	if err := services.seedingService.SeedAll(context.Background()); err != nil {
		logger.Warn().Err(err).Msg("Failed to seed database")
	}

	// Initialize HTTP handlers
	handlers := initHandlers(cfg, services, db, redisClient, natsClient)

	// Initialize middleware
	middlewares := initMiddleware(services)

	// Initialize HTTP router
	httpRouter := httphandler.NewRouter(
		cfg,
		handlers.authHandler,
		handlers.healthHandler,
		middlewares.jwtMiddleware,
		middlewares.permissionMiddleware,
	)

	// Initialize gRPC handler and server
	grpcAuthHandler := grpchandler.NewAuthHandler(services.tokenService, services.userService)
	grpcServer := grpchandler.NewServer(cfg, grpcAuthHandler)

	return &App{
		cfg:         cfg,
		db:          db,
		redisClient: redisClient,
		natsClient:  natsClient,
		httpRouter:  httpRouter,
		grpcServer:  grpcServer,
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
		Msg("Servers started")

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

	// Cleanup
	a.cleanup()

	logger.Info().Msg("Servers exited properly")
	return nil
}

func (a *App) cleanup() {
	if a.natsClient != nil {
		a.natsClient.Close()
	}
	if a.redisClient != nil {
		a.redisClient.Close()
	}
	postgres.Close(a.db)
}

type repositories struct {
	userRepo           repository.UserRepository
	roleRepo           repository.RoleRepository
	permissionRepo     repository.PermissionRepository
	userRoleRepo       repository.UserRoleRepository
	rolePermissionRepo repository.RolePermissionRepository
	refreshTokenRepo   repository.RefreshTokenRepository
	loginHistoryRepo   repository.LoginHistoryRepository
	passwordResetRepo  repository.PasswordResetRepository
}

func initRepositories(db *gorm.DB) *repositories {
	return &repositories{
		userRepo:           postgres.NewUserRepository(db),
		roleRepo:           postgres.NewRoleRepository(db),
		permissionRepo:     postgres.NewPermissionRepository(db),
		userRoleRepo:       postgres.NewUserRoleRepository(db),
		rolePermissionRepo: postgres.NewRolePermissionRepository(db),
		refreshTokenRepo:   postgres.NewRefreshTokenRepository(db),
		loginHistoryRepo:   postgres.NewLoginHistoryRepository(db),
		passwordResetRepo:  postgres.NewPasswordResetRepository(db),
	}
}

type services struct {
	tokenService   *service.TokenService
	userService    *service.UserService
	authService    *service.AuthService
	seedingService *service.SeedingService
}

func initServices(cfg *config.Config, repos *repositories, redisClient *cache.RedisClient, natsClient *queue.NATSClient) *services {
	tokenService := service.NewTokenService(cfg, repos.refreshTokenRepo, redisClient)
	userService := service.NewUserService(
		repos.userRepo,
		repos.roleRepo,
		repos.userRoleRepo,
		repos.rolePermissionRepo,
		repos.loginHistoryRepo,
		redisClient,
	)
	authService := service.NewAuthService(userService, tokenService, redisClient, natsClient)
	seedingService := service.NewSeedingService(repos.roleRepo, repos.permissionRepo, repos.rolePermissionRepo)

	return &services{
		tokenService:   tokenService,
		userService:    userService,
		authService:    authService,
		seedingService: seedingService,
	}
}

type handlers struct {
	authHandler   *httphandler.AuthHandler
	healthHandler *httphandler.HealthHandler
}

func initHandlers(cfg *config.Config, svcs *services, db *gorm.DB, redisClient *cache.RedisClient, natsClient *queue.NATSClient) *handlers {
	return &handlers{
		authHandler:   httphandler.NewAuthHandler(svcs.authService, cfg),
		healthHandler: httphandler.NewHealthHandler(db, redisClient, natsClient),
	}
}

type middlewares struct {
	jwtMiddleware        *middleware.JWTMiddleware
	permissionMiddleware *middleware.PermissionMiddleware
}

func initMiddleware(svcs *services) *middlewares {
	return &middlewares{
		jwtMiddleware:        middleware.NewJWTMiddleware(svcs.tokenService, svcs.authService),
		permissionMiddleware: middleware.NewPermissionMiddleware(svcs.authService),
	}
}
