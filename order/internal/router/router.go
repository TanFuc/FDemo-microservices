package router

import (
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
	"github.com/gofiber/fiber/v2/middleware/logger"
	"github.com/gofiber/fiber/v2/middleware/recover"

	"microservices/order/internal/config"
	httphandler "microservices/order/internal/handler/http"
	"microservices/pkg/authclient"
)

type Router struct {
	app            *fiber.App
	cfg            *config.Config
	orderHandler   *httphandler.OrderHandler
	returnHandler  *httphandler.ReturnHandler
	healthHandler  *httphandler.HealthHandler
	authMiddleware *authclient.FiberMiddleware
}

func NewRouter(
	cfg *config.Config,
	orderHandler *httphandler.OrderHandler,
	returnHandler *httphandler.ReturnHandler,
	healthHandler *httphandler.HealthHandler,
	authMiddleware *authclient.FiberMiddleware,
) *Router {
	app := fiber.New(fiber.Config{
		ErrorHandler: customErrorHandler,
	})

	return &Router{
		app:            app,
		cfg:            cfg,
		orderHandler:   orderHandler,
		returnHandler:  returnHandler,
		healthHandler:  healthHandler,
		authMiddleware: authMiddleware,
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

	// Order routes
	orders := api.Group("/orders")

	// Apply auth middleware if available
	if r.authMiddleware != nil {
		orders.Use(r.authMiddleware.RequireAuth())
	}

	orders.Post("/", r.orderHandler.CreateOrder)
	orders.Get("/:id", r.orderHandler.GetOrder)
	orders.Post("/:id/cancel", r.orderHandler.CancelOrder)
	orders.Post("/:id/pay", r.orderHandler.MarkAsPaid)
	orders.Post("/:id/ship", r.orderHandler.MarkAsShipped)
	orders.Post("/:id/complete", r.orderHandler.MarkAsCompleted)
	orders.Get("/user/:userId", r.orderHandler.ListOrders)

	// Return request routes
	returns := api.Group("/returns")
	if r.authMiddleware != nil {
		returns.Use(r.authMiddleware.RequireAuth())
	}

	returns.Post("/", r.returnHandler.CreateReturn)
	returns.Get("/", r.returnHandler.ListReturns)
	returns.Get("/:id", r.returnHandler.GetReturn)
	returns.Post("/:id/approve", r.returnHandler.ApproveReturn)
	returns.Post("/:id/reject", r.returnHandler.RejectReturn)
	returns.Post("/:id/complete", r.returnHandler.CompleteReturn)

	// Get returns for a specific order
	orders.Get("/:id/returns", r.returnHandler.GetReturnsByOrder)

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
