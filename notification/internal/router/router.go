package router

import (
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
	"github.com/gofiber/fiber/v2/middleware/logger"
	"github.com/gofiber/fiber/v2/middleware/recover"

	"microservices/notification/internal/config"
	httphandler "microservices/notification/internal/handler/http"
	wshandler "microservices/notification/internal/handler/websocket"
	"microservices/pkg/authclient"
)

type Router struct {
	app                 *fiber.App
	cfg                 *config.Config
	notificationHandler *httphandler.NotificationHandler
	wsHandler           *wshandler.Handler
	authMiddleware      *authclient.FiberMiddleware
}

func NewRouter(
	cfg *config.Config,
	notificationHandler *httphandler.NotificationHandler,
	wsHandler *wshandler.Handler,
	authMiddleware *authclient.FiberMiddleware,
) *Router {
	app := fiber.New(fiber.Config{
		ErrorHandler: customErrorHandler,
	})

	return &Router{
		app:                 app,
		cfg:                 cfg,
		notificationHandler: notificationHandler,
		wsHandler:           wsHandler,
		authMiddleware:      authMiddleware,
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
	r.app.Get("/health", r.notificationHandler.HealthCheck)

	// WebSocket endpoint
	if r.cfg.WebSocket.Enabled {
		ws := r.app.Group("/ws")
		ws.Use(r.wsHandler.WebSocketUpgrade())

		// Optional auth for WebSocket (can also use query param)
		if r.authMiddleware != nil {
			ws.Use(r.authMiddleware.OptionalAuth())
		}

		ws.Get("/", r.wsHandler.HandleConnection())
		ws.Get("/stats", r.wsHandler.GetStats)
	}

	// API routes
	api := r.app.Group("/api/v1")

	// Notifications routes
	notifications := api.Group("/notifications")

	// Public endpoints (for internal services)
	notifications.Post("/send", r.notificationHandler.SendNotification)
	notifications.Post("/broadcast", r.notificationHandler.BroadcastNotification)

	// Protected endpoints (require auth)
	if r.authMiddleware != nil {
		protected := notifications.Group("")
		protected.Use(r.authMiddleware.RequireAuth())

		protected.Get("/", r.notificationHandler.GetNotifications)
		protected.Get("/unread-count", r.notificationHandler.GetUnreadCount)
		protected.Post("/:id/read", r.notificationHandler.MarkAsRead)
		protected.Post("/read-all", r.notificationHandler.MarkAllAsRead)
		protected.Delete("/:id", r.notificationHandler.DeleteNotification)
	} else {
		// No auth middleware - expose all endpoints
		notifications.Get("/", r.notificationHandler.GetNotifications)
		notifications.Get("/unread-count", r.notificationHandler.GetUnreadCount)
		notifications.Post("/:id/read", r.notificationHandler.MarkAsRead)
		notifications.Post("/read-all", r.notificationHandler.MarkAllAsRead)
		notifications.Delete("/:id", r.notificationHandler.DeleteNotification)
	}

	// Admin routes
	admin := api.Group("/admin")
	admin.Get("/online-users", r.notificationHandler.GetOnlineUsers)

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
