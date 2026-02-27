package router

import (
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
	"github.com/gofiber/fiber/v2/middleware/logger"
	"github.com/gofiber/fiber/v2/middleware/recover"

	"microservices/pkg/authclient"
	"microservices/wallet/internal/config"
	httphandler "microservices/wallet/internal/handler/http"
)

// Router handles HTTP routing
type Router struct {
	app            *fiber.App
	cfg            *config.Config
	walletHandler  *httphandler.WalletHandler
	healthHandler  *httphandler.HealthHandler
	authMiddleware *authclient.FiberMiddleware
}

// NewRouter creates a new router
func NewRouter(
	cfg *config.Config,
	walletHandler *httphandler.WalletHandler,
	healthHandler *httphandler.HealthHandler,
	authMiddleware *authclient.FiberMiddleware,
) *Router {
	app := fiber.New(fiber.Config{
		ErrorHandler: customErrorHandler,
	})

	return &Router{
		app:            app,
		cfg:            cfg,
		walletHandler:  walletHandler,
		healthHandler:  healthHandler,
		authMiddleware: authMiddleware,
	}
}

// Setup configures all routes
func (r *Router) Setup() *fiber.App {
	// Middleware
	r.app.Use(recover.New())
	r.app.Use(logger.New())
	r.app.Use(cors.New(cors.Config{
		AllowOrigins: "*",
		AllowMethods: "GET,POST,PUT,DELETE,OPTIONS",
		AllowHeaders: "Origin,Content-Type,Accept,Authorization,X-Idempotency-Key",
	}))

	// Health check endpoints
	r.app.Get("/health", r.healthHandler.Health)
	r.app.Get("/ready", r.healthHandler.Ready)

	// API routes
	api := r.app.Group("/api/v1")

	// Wallet routes
	wallet := api.Group("/wallet")

	// Public routes that need auth
	if r.authMiddleware != nil {
		wallet.Use(r.authMiddleware.RequireAuth())
	}

	wallet.Get("/", r.walletHandler.GetWallet)
	wallet.Get("/history", r.walletHandler.GetHistory)
	wallet.Post("/topup", r.walletHandler.InitiateTopUp)
	wallet.Post("/pay", r.walletHandler.PayWithWallet)

	// Internal/admin refund endpoint
	// In production, add additional auth checks (internal service token or admin role)
	refund := api.Group("/wallet")
	if r.authMiddleware != nil {
		refund.Use(r.authMiddleware.RequireAnyRole("admin", "service"))
	}
	refund.Post("/refund", r.walletHandler.RefundToWallet)

	return r.app
}

// App returns the fiber app
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
