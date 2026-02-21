package router

import (
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
	"github.com/gofiber/fiber/v2/middleware/logger"
	"github.com/gofiber/fiber/v2/middleware/recover"

	"microservices/template-service/internal/config"
	httphandler "microservices/template-service/internal/handler/http"
)

type Router struct {
	app         *fiber.App
	cfg         *config.Config
	itemHandler *httphandler.ItemHandler
}

func NewRouter(cfg *config.Config, itemHandler *httphandler.ItemHandler) *Router {
	app := fiber.New(fiber.Config{
		ErrorHandler: customErrorHandler,
	})

	return &Router{
		app:         app,
		cfg:         cfg,
		itemHandler: itemHandler,
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

	// Health check
	r.app.Get("/health", r.itemHandler.HealthCheck)

	// API routes
	api := r.app.Group("/api/v1")

	// Items routes
	items := api.Group("/items")
	items.Post("/", r.itemHandler.CreateItem)
	items.Get("/", r.itemHandler.ListItems)
	items.Get("/:id", r.itemHandler.GetItem)
	items.Put("/:id", r.itemHandler.UpdateItem)
	items.Delete("/:id", r.itemHandler.DeleteItem)

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
