package main

import (
	"context"
	"os"
	"os/signal"
	"syscall"
	"time"

	"gorm.io/gorm"

	"microservices/auth/internal/config"
	"microservices/auth/internal/delivery/http/handler"
	"microservices/auth/internal/delivery/http/middleware"
	"microservices/auth/internal/delivery/http/router"
	"microservices/auth/internal/domain/repository"
	"microservices/auth/internal/domain/service"
	"microservices/auth/internal/infrastructure/cache"
	"microservices/auth/internal/infrastructure/database"
	"microservices/auth/internal/infrastructure/queue"
	"microservices/auth/pkg/logger"

	_ "microservices/auth/docs"
)

// @title Tafu Auth Service API
// @version 1.0
// @description Authentication and authorization service for Tafu e-commerce platform
// @termsOfService http://swagger.io/terms/

// @contact.name API Support
// @contact.email support@tafu.vn

// @license.name Apache 2.0
// @license.url http://www.apache.org/licenses/LICENSE-2.0.html

// @host localhost:3001
// @BasePath /api/v1

// @securityDefinitions.apikey BearerAuth
// @in header
// @name Authorization
// @description Enter your bearer token in the format: Bearer {token}

func main() {
	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		panic("Failed to load configuration: " + err.Error())
	}

	// Initialize logger
	logger.Init(cfg.App.Env)
	logger.Info().
		Str("app", cfg.App.Name).
		Str("env", cfg.App.Env).
		Str("port", cfg.App.Port).
		Msg("Starting application")

	// Initialize database
	db, err := database.NewPostgresDB(cfg)
	if err != nil {
		logger.Fatal().Err(err).Msg("Failed to connect to database")
	}

	// Auto migrate
	if err := database.AutoMigrate(db); err != nil {
		logger.Fatal().Err(err).Msg("Failed to run migrations")
	}

	// Initialize Redis
	redisClient, err := cache.NewRedisClient(cfg)
	if err != nil {
		logger.Fatal().Err(err).Msg("Failed to connect to Redis")
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

	// Initialize handlers
	handlers := initHandlers(cfg, services, db, redisClient, natsClient)

	// Initialize middleware
	middlewares := initMiddleware(services)

	// Initialize router
	r := router.NewRouter(
		cfg,
		handlers.authHandler,
		handlers.healthHandler,
		middlewares.jwtMiddleware,
		middlewares.permissionMiddleware,
	)

	app := r.Setup()

	// Graceful shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		if err := app.Listen(":" + cfg.App.Port); err != nil {
			logger.Fatal().Err(err).Msg("Failed to start server")
		}
	}()

	logger.Info().Str("port", cfg.App.Port).Msg("Server started")

	<-quit
	logger.Info().Msg("Shutting down server...")

	// Graceful shutdown with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := app.ShutdownWithContext(ctx); err != nil {
		logger.Error().Err(err).Msg("Server forced to shutdown")
	}

	// Cleanup
	if natsClient != nil {
		natsClient.Close()
	}
	redisClient.Close()
	database.Close(db)

	logger.Info().Msg("Server exited properly")
}

type Repositories struct {
	userRepo           repository.UserRepository
	roleRepo           repository.RoleRepository
	permissionRepo     repository.PermissionRepository
	userRoleRepo       repository.UserRoleRepository
	rolePermissionRepo repository.RolePermissionRepository
	refreshTokenRepo   repository.RefreshTokenRepository
	loginHistoryRepo   repository.LoginHistoryRepository
	passwordResetRepo  repository.PasswordResetRepository
}

func initRepositories(db *gorm.DB) *Repositories {
	return &Repositories{
		userRepo:           database.NewUserRepository(db),
		roleRepo:           database.NewRoleRepository(db),
		permissionRepo:     database.NewPermissionRepository(db),
		userRoleRepo:       database.NewUserRoleRepository(db),
		rolePermissionRepo: database.NewRolePermissionRepository(db),
		refreshTokenRepo:   database.NewRefreshTokenRepository(db),
		loginHistoryRepo:   database.NewLoginHistoryRepository(db),
		passwordResetRepo:  database.NewPasswordResetRepository(db),
	}
}

type Services struct {
	tokenService   *service.TokenService
	userService    *service.UserService
	authService    *service.AuthService
	seedingService *service.SeedingService
}

func initServices(cfg *config.Config, repos *Repositories, redisClient *cache.RedisClient, natsClient *queue.NATSClient) *Services {
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

	return &Services{
		tokenService:   tokenService,
		userService:    userService,
		authService:    authService,
		seedingService: seedingService,
	}
}

type Handlers struct {
	authHandler   *handler.AuthHandler
	healthHandler *handler.HealthHandler
}

func initHandlers(cfg *config.Config, services *Services, db *gorm.DB, redisClient *cache.RedisClient, natsClient *queue.NATSClient) *Handlers {
	return &Handlers{
		authHandler:   handler.NewAuthHandler(services.authService, cfg),
		healthHandler: handler.NewHealthHandler(db, redisClient, natsClient),
	}
}

type Middleware struct {
	jwtMiddleware        *middleware.JWTMiddleware
	permissionMiddleware *middleware.PermissionMiddleware
}

func initMiddleware(services *Services) *Middleware {
	return &Middleware{
		jwtMiddleware:        middleware.NewJWTMiddleware(services.tokenService, services.authService),
		permissionMiddleware: middleware.NewPermissionMiddleware(services.authService),
	}
}
