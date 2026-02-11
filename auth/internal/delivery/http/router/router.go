package router

import (
	"strings"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
	"github.com/gofiber/fiber/v2/middleware/helmet"
	swagger "github.com/swaggo/fiber-swagger"

	"microservices/auth/internal/config"
	"microservices/auth/internal/delivery/http/handler"
	"microservices/auth/internal/delivery/http/middleware"
)

type Router struct {
	app                  *fiber.App
	cfg                  *config.Config
	authHandler          *handler.AuthHandler
	healthHandler        *handler.HealthHandler
	jwtMiddleware        *middleware.JWTMiddleware
	permissionMiddleware *middleware.PermissionMiddleware
}

func NewRouter(
	cfg *config.Config,
	authHandler *handler.AuthHandler,
	healthHandler *handler.HealthHandler,
	jwtMiddleware *middleware.JWTMiddleware,
	permissionMiddleware *middleware.PermissionMiddleware,
) *Router {
	app := fiber.New(fiber.Config{
		ErrorHandler: customErrorHandler,
	})

	return &Router{
		app:                  app,
		cfg:                  cfg,
		authHandler:          authHandler,
		healthHandler:        healthHandler,
		jwtMiddleware:        jwtMiddleware,
		permissionMiddleware: permissionMiddleware,
	}
}

func (r *Router) Setup() *fiber.App {
	// Global middleware
	r.app.Use(helmet.New(helmet.Config{
		ContentSecurityPolicy: "default-src 'self'; script-src 'self' 'unsafe-inline' 'unsafe-eval'; style-src 'self' 'unsafe-inline'; img-src 'self' data:;",
	}))
	origins := strings.Join(r.cfg.App.CORSOrigins, ",")
	r.app.Use(cors.New(cors.Config{
		AllowOrigins:     origins,
		AllowMethods:     "GET,POST,PUT,PATCH,DELETE,OPTIONS",
		AllowHeaders:     "Origin,Content-Type,Accept,Authorization,X-Device-ID,X-Request-ID",
		ExposeHeaders:    "Set-Cookie",
		AllowCredentials: origins != "*",
	}))
	r.app.Use(middleware.RequestID())
	r.app.Use(middleware.RequestLogger())
	r.app.Use(middleware.Recovery())

	// API routes
	api := r.app.Group("/" + r.cfg.App.APIPrefix)

	// Swagger
	r.app.Get("/swagger/*", swagger.WrapHandler)

	// Redirect /api/swagger to /swagger/index.html
	r.app.Get("/api/swagger", func(c *fiber.Ctx) error {
		return c.Redirect("/swagger/index.html", fiber.StatusFound)
	})

	// Health routes (public)
	r.app.Get("/health", r.healthHandler.Health)
	r.app.Get("/health/live", r.healthHandler.Liveness)
	r.app.Get("/health/ready", r.healthHandler.Readiness)

	// Auth routes
	auth := api.Group("/auth")

	// Public routes
	auth.Post("/register", r.authHandler.Register)
	auth.Post("/login", r.authHandler.Login)
	auth.Post("/refresh", r.authHandler.RefreshToken)

	// Protected routes
	protected := auth.Group("", r.jwtMiddleware.Authenticate())
	protected.Post("/logout", r.authHandler.Logout)
	protected.Post("/logout-all", r.authHandler.LogoutAll)
	protected.Get("/profile", r.authHandler.GetProfile)
	protected.Get("/sessions", r.authHandler.GetSessions)

	// Permission-protected route example
	protected.Get("/check-permission",
		r.permissionMiddleware.RequirePermissions("product:create"),
		r.authHandler.CheckPermission,
	)

	return r.app
}

func (r *Router) App() *fiber.App {
	return r.app
}

func customErrorHandler(c *fiber.Ctx, err error) error {
	code := fiber.StatusInternalServerError
	message := "Internal Server Error"

	if e, ok := err.(*fiber.Error); ok {
		code = e.Code
		message = e.Message
	}

	return c.Status(code).JSON(fiber.Map{
		"success": false,
		"code":    "ERROR",
		"message": message,
	})
}
