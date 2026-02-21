package router

import (
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
	"github.com/gofiber/fiber/v2/middleware/logger"
	"github.com/gofiber/fiber/v2/middleware/recover"

	"microservices/logistic/internal/config"
	httphandler "microservices/logistic/internal/handler/http"
	"microservices/pkg/authclient"
)

type Router struct {
	app             *fiber.App
	cfg             *config.Config
	shippingHandler *httphandler.ShippingHandler
	webhookHandler  *httphandler.WebhookHandler
	healthHandler   *httphandler.HealthHandler
	authMiddleware  *authclient.FiberMiddleware
}

func NewRouter(
	cfg *config.Config,
	shippingHandler *httphandler.ShippingHandler,
	webhookHandler *httphandler.WebhookHandler,
	healthHandler *httphandler.HealthHandler,
	authMiddleware *authclient.FiberMiddleware,
) *Router {
	app := fiber.New(fiber.Config{
		ErrorHandler: customErrorHandler,
	})

	return &Router{
		app:             app,
		cfg:             cfg,
		shippingHandler: shippingHandler,
		webhookHandler:  webhookHandler,
		healthHandler:   healthHandler,
		authMiddleware:  authMiddleware,
	}
}

func (r *Router) Setup() *fiber.App {
	// Middleware
	r.app.Use(recover.New())
	r.app.Use(logger.New())
	r.app.Use(cors.New(cors.Config{
		AllowOrigins: "*",
		AllowMethods: "GET,POST,PUT,DELETE,OPTIONS",
		AllowHeaders: "Origin,Content-Type,Accept,Authorization",
	}))

	// Health check endpoints
	r.app.Get("/health", r.healthHandler.Health)
	r.app.Get("/health/detailed", r.healthHandler.HealthDetailed)
	r.app.Get("/ready", r.healthHandler.Ready)
	r.app.Get("/metrics/db", r.healthHandler.DBStats)

	// API routes
	api := r.app.Group("/api/v1")

	// Shipping routes
	shipping := api.Group("/shipping")

	// Apply auth middleware if available
	if r.authMiddleware != nil {
		shipping.Use(r.authMiddleware.RequireAuth())
	}

	shipping.Post("/calculate-fee", r.shippingHandler.CalculateFee)
	shipping.Post("/create", r.shippingHandler.CreateShipment)
	shipping.Get("/providers", r.shippingHandler.ListProviders)
	shipping.Get("/:id", r.shippingHandler.GetShipment)
	shipping.Get("/track/:tracking_code", r.shippingHandler.GetShipmentByTracking)

	// Webhook routes (no auth - external providers need to call these)
	webhooks := api.Group("/webhooks")
	webhooks.Post("/:provider", r.webhookHandler.HandleWebhook)

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
