package router

import (
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
	"github.com/gofiber/fiber/v2/middleware/logger"
	"github.com/gofiber/fiber/v2/middleware/recover"

	"microservices/media/internal/config"
	httphandler "microservices/media/internal/handler/http"
	"microservices/pkg/authclient"
)

type Router struct {
	app            *fiber.App
	cfg            *config.Config
	mediaHandler   *httphandler.MediaHandler
	authMiddleware *authclient.FiberMiddleware
}

func NewRouter(cfg *config.Config, mediaHandler *httphandler.MediaHandler, authMiddleware *authclient.FiberMiddleware) *Router {
	app := fiber.New(fiber.Config{
		ErrorHandler: customErrorHandler,
	})

	return &Router{
		app:            app,
		cfg:            cfg,
		mediaHandler:   mediaHandler,
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

	// Health check
	r.app.Get("/health", r.mediaHandler.HealthCheck)

	// API routes
	api := r.app.Group("/api/v1")

	// Media routes
	media := api.Group("/media")

	// Apply auth middleware if available
	if r.authMiddleware != nil {
		media.Use(r.authMiddleware.RequireAuth())
	}

	media.Post("/upload-url", r.mediaHandler.GetUploadURL)
	media.Post("/confirm", r.mediaHandler.ConfirmUpload)
	media.Get("/", r.mediaHandler.ListMedia)
	media.Get("/:id", r.mediaHandler.GetMedia)
	media.Delete("/:id", r.mediaHandler.DeleteMedia)

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
