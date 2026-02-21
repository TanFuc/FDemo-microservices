package router

import (
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
	"github.com/gofiber/fiber/v2/middleware/logger"
	"github.com/gofiber/fiber/v2/middleware/recover"

	"microservices/review/internal/config"
	httphandler "microservices/review/internal/handler/http"
	"microservices/pkg/authclient"
)

type Router struct {
	app            *fiber.App
	cfg            *config.Config
	reviewHandler  *httphandler.ReviewHandler
	healthHandler  *httphandler.HealthHandler
	authMiddleware *authclient.FiberMiddleware
}

func NewRouter(
	cfg *config.Config,
	reviewHandler *httphandler.ReviewHandler,
	healthHandler *httphandler.HealthHandler,
	authMiddleware *authclient.FiberMiddleware,
) *Router {
	app := fiber.New(fiber.Config{
		ErrorHandler: customErrorHandler,
	})

	return &Router{
		app:            app,
		cfg:            cfg,
		reviewHandler:  reviewHandler,
		healthHandler:  healthHandler,
		authMiddleware: authMiddleware,
	}
}

func (r *Router) Setup() *fiber.App {
	// Middleware
	r.app.Use(recover.New())
	r.app.Use(logger.New(logger.Config{
		Format: "[${time}] ${status} - ${method} ${path} ${latency}\n",
	}))
	r.app.Use(cors.New(cors.Config{
		AllowOrigins: "*",
		AllowMethods: "GET,POST,PUT,PATCH,DELETE,OPTIONS",
		AllowHeaders: "Origin,Content-Type,Accept,Authorization",
	}))

	// Health check endpoints
	r.app.Get("/health", r.healthHandler.Health)
	r.app.Get("/ready", r.healthHandler.Ready)

	// API routes
	api := r.app.Group("/api")

	// Review endpoints
	reviews := api.Group("/reviews")
	reviews.Get("/:reviewId", r.reviewHandler.GetReviewByID)

	// Protected review endpoints (require auth)
	if r.authMiddleware != nil {
		protectedReviews := reviews.Group("", r.authMiddleware.RequireAuth())
		protectedReviews.Post("/", r.reviewHandler.CreateReview)
		protectedReviews.Post("/:reviewId/reply", r.reviewHandler.ReplyReview)
		protectedReviews.Patch("/:reviewId/status", r.reviewHandler.UpdateReviewStatus)
	} else {
		reviews.Post("/", r.reviewHandler.CreateReview)
		reviews.Post("/:reviewId/reply", r.reviewHandler.ReplyReview)
		reviews.Patch("/:reviewId/status", r.reviewHandler.UpdateReviewStatus)
	}

	// Product endpoints (public)
	products := api.Group("/products")
	products.Get("/:productId/reviews", r.reviewHandler.GetProductReviews)
	products.Get("/:productId/rating", r.reviewHandler.GetRatingSummary)

	// User endpoints (protected)
	users := api.Group("/users")
	if r.authMiddleware != nil {
		users.Use(r.authMiddleware.RequireAuth())
	}
	users.Get("/:userId/reviews", r.reviewHandler.GetUserReviews)

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
		"error":   "INTERNAL_ERROR",
		"message": message,
	})
}
