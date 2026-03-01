package http

import (
	"github.com/gin-gonic/gin"

	"microservices/pkg/authclient"
	"microservices/logistic/internal/api/http/handlers"
	"microservices/logistic/internal/api/http/middleware"
	"microservices/logistic/internal/core/services"
)

// Router holds the HTTP router and handlers
type Router struct {
	engine          *gin.Engine
	shippingHandler *handlers.ShippingHandler
	webhookHandler  *handlers.WebhookHandler
	healthHandler   *handlers.HealthHandler
	authMiddleware  *authclient.GinMiddleware
}

// NewRouter creates a new HTTP router
func NewRouter(
	shippingService *services.ShippingService,
	webhookService *services.WebhookService,
) *Router {
	return NewRouterWithHealth(shippingService, webhookService, nil, nil)
}

// NewRouterWithHealth creates a new HTTP router with health service
func NewRouterWithHealth(
	shippingService *services.ShippingService,
	webhookService *services.WebhookService,
	healthService *services.HealthService,
	authMiddleware *authclient.GinMiddleware,
) *Router {
	// Set Gin mode
	gin.SetMode(gin.ReleaseMode)

	engine := gin.New()

	// Apply middleware
	engine.Use(middleware.Recovery())
	engine.Use(middleware.Logger())
	engine.Use(middleware.CORS())

	router := &Router{
		engine:          engine,
		shippingHandler: handlers.NewShippingHandler(shippingService),
		webhookHandler:  handlers.NewWebhookHandler(webhookService),
		healthHandler:   handlers.NewHealthHandler(healthService),
		authMiddleware:  authMiddleware,
	}

	// Setup routes
	router.setupRoutes()

	return router
}

// setupRoutes configures all API routes
func (r *Router) setupRoutes() {
	// Health check endpoints
	r.engine.GET("/health", r.healthHandler.Health)
	r.engine.GET("/health/detailed", r.healthHandler.HealthDetailed)
	r.engine.GET("/ready", r.healthHandler.Ready)
	r.engine.GET("/metrics/db", r.healthHandler.DBStats)

	// API v1 routes
	v1 := r.engine.Group("/api/v1")
	{
		// Shipping routes
		shipping := v1.Group("/shipping")

		// Apply auth middleware if available
		if r.authMiddleware != nil {
			shipping.Use(r.authMiddleware.RequireAuth())
		}

		{
			shipping.POST("/calculate-fee", r.shippingHandler.CalculateFee)
			shipping.POST("/compare-fees", r.shippingHandler.CompareFees)
			shipping.POST("/create", r.shippingHandler.CreateShipment)
			shipping.GET("/providers", r.shippingHandler.ListProviders)
			shipping.GET("/:id", r.shippingHandler.GetShipment)
			shipping.GET("/track/:tracking_code", r.shippingHandler.GetShipmentByTracking)
			shipping.POST("/:id/cancel", r.shippingHandler.CancelShipment)
		}

		// Webhook routes (no auth - external providers need to call these)
		webhooks := v1.Group("/webhooks")
		{
			webhooks.POST("/:provider", r.webhookHandler.HandleWebhook)
		}
	}
}

// Run starts the HTTP server
func (r *Router) Run(addr string) error {
	return r.engine.Run(addr)
}

// Engine returns the underlying Gin engine
func (r *Router) Engine() *gin.Engine {
	return r.engine
}
