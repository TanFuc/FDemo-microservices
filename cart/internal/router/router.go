package router

import (
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
	"github.com/gofiber/fiber/v2/middleware/logger"
	"github.com/gofiber/fiber/v2/middleware/recover"

	"microservices/cart/internal/config"
	httphandler "microservices/cart/internal/handler/http"
	"microservices/cart/internal/middleware"
)

// Router handles HTTP routing.
type Router struct {
	app            *fiber.App
	cfg            *config.Config
	cartHandler    *httphandler.CartHandler
	authMiddleware *middleware.AuthMiddleware
}

// NewRouter creates a new router.
func NewRouter(
	cfg *config.Config,
	cartHandler *httphandler.CartHandler,
	authMiddleware *middleware.AuthMiddleware,
) *Router {
	app := fiber.New(fiber.Config{
		AppName:      cfg.App.Name,
		ErrorHandler: customErrorHandler,
	})

	return &Router{
		app:            app,
		cfg:            cfg,
		cartHandler:    cartHandler,
		authMiddleware: authMiddleware,
	}
}

// Setup configures all routes and middleware.
func (r *Router) Setup() *fiber.App {
	// Global middleware
	r.app.Use(recover.New())
	r.app.Use(logger.New(logger.Config{
		Format:     "${time} | ${status} | ${latency} | ${method} | ${path}\n",
		TimeFormat: "2006-01-02 15:04:05",
	}))
	r.app.Use(cors.New(cors.Config{
		AllowOrigins: "*",
		AllowMethods: "GET,POST,PUT,DELETE,OPTIONS",
		AllowHeaders: "Origin,Content-Type,Accept,Authorization",
	}))

	// Health check
	r.app.Get("/health", r.cartHandler.HealthCheck)

	// API routes
	api := r.app.Group("/api/v1")

	// Cart routes - protected by auth middleware
	cart := api.Group("/cart")

	// Apply auth middleware if available
	if r.authMiddleware != nil {
		cart.Use(r.authMiddleware.RequireAuth())
	}

	// Cart operations with user ownership validation
	cartUser := cart.Group("/:userId")
	if r.authMiddleware != nil {
		cartUser.Use(r.authMiddleware.ValidateUserOwnership())
	}

	cartUser.Get("", r.cartHandler.GetCart)
	cartUser.Get("/summary", r.cartHandler.GetCartSummary)
	cartUser.Delete("", r.cartHandler.ClearCart)

	// Item operations
	cartUser.Post("/items", r.cartHandler.AddToCart)
	cartUser.Delete("/items/:skuId", r.cartHandler.RemoveItem)
	cartUser.Put("/items/:skuId/quantity", r.cartHandler.UpdateQuantity)
	cartUser.Put("/items/:skuId/selection", r.cartHandler.UpdateSelection)

	return r.app
}

// App returns the fiber app instance.
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
