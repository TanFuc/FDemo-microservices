package router

import (
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
	"github.com/gofiber/fiber/v2/middleware/logger"
	"github.com/gofiber/fiber/v2/middleware/recover"

	"microservices/search/internal/config"
	httphandler "microservices/search/internal/handler/http"
)

type Router struct {
	app           *fiber.App
	cfg           *config.Config
	searchHandler *httphandler.SearchHandler
	healthHandler *httphandler.HealthHandler
}

func NewRouter(
	cfg *config.Config,
	searchHandler *httphandler.SearchHandler,
	healthHandler *httphandler.HealthHandler,
) *Router {
	app := fiber.New(fiber.Config{
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
		IdleTimeout:  120 * time.Second,
		ErrorHandler: customErrorHandler,
	})

	return &Router{
		app:           app,
		cfg:           cfg,
		searchHandler: searchHandler,
		healthHandler: healthHandler,
	}
}

func (r *Router) Setup() *fiber.App {
	// Middleware
	r.app.Use(recover.New())
	r.app.Use(cors.New(cors.Config{
		AllowOrigins: "*",
		AllowMethods: "GET,POST,PUT,DELETE,OPTIONS",
		AllowHeaders: "Origin,Content-Type,Accept,Authorization",
	}))
	r.app.Use(logger.New(logger.Config{
		Format: "[${time}] ${status} - ${latency} ${method} ${path}\n",
	}))

	// Health check endpoints
	r.app.Get("/health", r.healthHandler.Health)
	r.app.Get("/ready", r.healthHandler.Ready)

	// API routes
	api := r.app.Group("/api")

	// Search routes (public)
	api.Get("/search", r.searchHandler.SearchGet)
	api.Post("/search", r.searchHandler.Search)

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
		"error":   message,
	})
}
