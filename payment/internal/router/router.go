package router

import (
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
	"github.com/gofiber/fiber/v2/middleware/logger"
	"github.com/gofiber/fiber/v2/middleware/recover"

	"microservices/payment/internal/config"
	httphandler "microservices/payment/internal/handler/http"
	"microservices/pkg/authclient"
)

type Router struct {
	app             *fiber.App
	cfg             *config.Config
	paymentHandler  *httphandler.PaymentHandler
	webhookHandler  *httphandler.WebhookHandler
	healthHandler   *httphandler.HealthHandler
	authMiddleware  *authclient.FiberMiddleware
}

func NewRouter(
	cfg *config.Config,
	paymentHandler *httphandler.PaymentHandler,
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
		paymentHandler:  paymentHandler,
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
	r.app.Get("/ready", r.healthHandler.Ready)

	// API routes
	api := r.app.Group("/api/v1")

	// Payment routes
	payments := api.Group("/payments")

	// Apply auth middleware if available
	if r.authMiddleware != nil {
		payments.Use(r.authMiddleware.RequireAuth())
	}

	payments.Post("/", r.paymentHandler.CreatePayment)
	payments.Get("/:id", r.paymentHandler.GetPayment)
	payments.Get("/order/:orderId", r.paymentHandler.GetPaymentsByOrder)
	payments.Post("/:id/refund", r.paymentHandler.RefundPayment)

	// VNPay return URL (browser redirect - read only, no auth needed)
	payments.Get("/vnpay-return", r.paymentHandler.VNPayReturn)

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
