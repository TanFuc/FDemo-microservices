package routes

import (
	"strings"

	"microservices/api-gateway/internal/config"
	"microservices/api-gateway/internal/middleware"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/proxy"
)

type Router struct {
	app     *fiber.App
	cfg     *config.Config
	storage fiber.Storage
}

func NewRouter(app *fiber.App, cfg *config.Config, storage fiber.Storage) *Router {
	return &Router{
		app:     app,
		cfg:     cfg,
		storage: storage,
	}
}

func (r *Router) Setup() {
	r.setupHealthCheck()
	r.setupIdentityRoutes()
	r.setupProfileRoutes()
	r.setupCatalogRoutes()
	r.setupCartRoutes()
	r.setupOrderRoutes()
	r.setupPaymentRoutes()
	r.setupLogisticRoutes()
	r.setupMediaRoutes()
	r.setupReviewRoutes()
	r.setupSearchRoutes()
	r.setupCampaignRoutes()
	r.setupAnalyticRoutes()
	r.setupInventoryRoutes()
	r.setupNotificationRoutes()
	r.setupRealtimeRoutes()
}

func (r *Router) setupHealthCheck() {
	r.app.Get("/health", func(c *fiber.Ctx) error {
		return c.JSON(fiber.Map{
			"status":  "healthy",
			"service": "api-gateway",
		})
	})
}

func (r *Router) setupIdentityRoutes() {
	// Support /api/v1/identity/auth/* with rewrite to /api/v1/auth/*
	identity := r.app.Group("/api/v1/identity")
	identity.All("/auth/*", r.createProxyWithRewrite(r.cfg.Services.IdentityURL, "/api/v1/identity/auth", "/api/v1/auth"))

	// Also support direct /api/v1/auth/* for seamless microservice contract parity
	authDirect := r.app.Group("/api/v1/auth")
	authDirect.All("/*", r.createProxy(r.cfg.Services.IdentityURL))

	protected := identity.Group("",
		middleware.JWTAuth(&r.cfg.JWT),
		middleware.RateLimiter(r.storage, &r.cfg.RateLimit),
	)
	protected.All("/users/*", r.createProxyWithRewrite(r.cfg.Services.IdentityURL, "/api/v1/identity/users", "/api/v1/users"))
	protected.All("/profile/*", r.createProxyWithRewrite(r.cfg.Services.IdentityURL, "/api/v1/identity/profile", "/api/v1/profile"))
}

func (r *Router) setupCatalogRoutes() {
	catalog := r.app.Group("/api/v1/catalog")

	catalog.Get("/products", r.createProxy(r.cfg.Services.CatalogURL))
	catalog.Get("/products/*", r.createProxy(r.cfg.Services.CatalogURL))
	catalog.Get("/categories", r.createProxy(r.cfg.Services.CatalogURL))
	catalog.Get("/categories/*", r.createProxy(r.cfg.Services.CatalogURL))

	protected := catalog.Group("",
		middleware.JWTAuth(&r.cfg.JWT),
		middleware.RateLimiter(r.storage, &r.cfg.RateLimit),
	)
	protected.Post("/products", r.createProxy(r.cfg.Services.CatalogURL))
	protected.Put("/products/*", r.createProxy(r.cfg.Services.CatalogURL))
	protected.Delete("/products/*", r.createProxy(r.cfg.Services.CatalogURL))
	protected.Post("/categories", r.createProxy(r.cfg.Services.CatalogURL))
	protected.Put("/categories/*", r.createProxy(r.cfg.Services.CatalogURL))
	protected.Delete("/categories/*", r.createProxy(r.cfg.Services.CatalogURL))
}

func (r *Router) setupCartRoutes() {
	cart := r.app.Group("/api/v1/cart",
		middleware.JWTAuth(&r.cfg.JWT),
		middleware.RateLimiter(r.storage, &r.cfg.RateLimit),
	)

	cart.All("/*", r.createProxy(r.cfg.Services.CartURL))
}

func (r *Router) setupOrderRoutes() {
	order := r.app.Group("/api/v1/orders",
		middleware.JWTAuth(&r.cfg.JWT),
		middleware.RateLimiter(r.storage, &r.cfg.RateLimit),
	)
	order.All("/*", r.createProxy(r.cfg.Services.OrderURL))

	// Singular route alias for client contract compatibility
	orderSingular := r.app.Group("/api/v1/order",
		middleware.JWTAuth(&r.cfg.JWT),
		middleware.RateLimiter(r.storage, &r.cfg.RateLimit),
	)
	orderSingular.All("/*", r.createProxyWithRewrite(r.cfg.Services.OrderURL, "/api/v1/order", "/api/v1/orders"))
}

func (r *Router) createProxy(targetURL string) fiber.Handler {
	return func(c *fiber.Ctx) error {
		url := targetURL + c.OriginalURL()
		return proxy.Do(c, url)
	}
}

func (r *Router) createProxyWithRewrite(targetURL, stripPrefix, addPrefix string) fiber.Handler {
	return func(c *fiber.Ctx) error {
		orig := c.OriginalURL()
		rewritten := orig
		if stripPrefix != "" && strings.HasPrefix(orig, stripPrefix) {
			rewritten = strings.TrimPrefix(orig, stripPrefix)
			if addPrefix != "" {
				rewritten = addPrefix + rewritten
			}
		}
		url := targetURL + rewritten
		return proxy.Do(c, url)
	}
}

func (r *Router) setupProfileRoutes() {
	profile := r.app.Group("/api/v1/profile",
		middleware.JWTAuth(&r.cfg.JWT),
		middleware.RateLimiter(r.storage, &r.cfg.RateLimit),
	)

	profile.Get("", r.createProxy(r.cfg.Services.ProfileURL))
	profile.Put("", r.createProxy(r.cfg.Services.ProfileURL))
	profile.Get("/addresses", r.createProxy(r.cfg.Services.ProfileURL))
	profile.Post("/addresses", r.createProxy(r.cfg.Services.ProfileURL))
	profile.Put("/addresses/:id", r.createProxy(r.cfg.Services.ProfileURL))
	profile.Delete("/addresses/:id", r.createProxy(r.cfg.Services.ProfileURL))
	profile.Post("/shop", r.createProxy(r.cfg.Services.ProfileURL))
}

func (r *Router) setupPaymentRoutes() {
	payment := r.app.Group("/api/v1/payments")

	payment.Post("/webhook/:provider", r.createProxy(r.cfg.Services.PaymentURL))

	protected := payment.Group("",
		middleware.JWTAuth(&r.cfg.JWT),
		middleware.RateLimiter(r.storage, &r.cfg.RateLimit),
	)
	protected.Post("", r.createProxy(r.cfg.Services.PaymentURL))
	protected.Get("/:id", r.createProxy(r.cfg.Services.PaymentURL))
	protected.Get("/order/:orderId", r.createProxy(r.cfg.Services.PaymentURL))
}

func (r *Router) setupLogisticRoutes() {
	logistics := r.app.Group("/api/v1/logistics")

	logistics.Post("/webhook/:provider", r.createProxy(r.cfg.Services.LogisticURL))

	protected := logistics.Group("",
		middleware.JWTAuth(&r.cfg.JWT),
		middleware.RateLimiter(r.storage, &r.cfg.RateLimit),
	)
	protected.Post("/calculate-fee", r.createProxy(r.cfg.Services.LogisticURL))
	protected.Post("/shipments", r.createProxy(r.cfg.Services.LogisticURL))
	protected.Get("/shipments/:id", r.createProxy(r.cfg.Services.LogisticURL))
	protected.Get("/shipments/order/:orderId", r.createProxy(r.cfg.Services.LogisticURL))
	protected.Get("/shipments/:id/tracking", r.createProxy(r.cfg.Services.LogisticURL))
}

func (r *Router) setupMediaRoutes() {
	media := r.app.Group("/api/v1/media",
		middleware.JWTAuth(&r.cfg.JWT),
		middleware.RateLimiter(r.storage, &r.cfg.RateLimit),
	)

	media.Post("/presigned-url", r.createProxy(r.cfg.Services.MediaURL))
	media.Post("/confirm", r.createProxy(r.cfg.Services.MediaURL))
	media.Delete("/:id", r.createProxy(r.cfg.Services.MediaURL))
	media.Get("/:id", r.createProxy(r.cfg.Services.MediaURL))
}

func (r *Router) setupReviewRoutes() {
	review := r.app.Group("/api/v1/reviews")

	review.Get("/products/:productId", r.createProxy(r.cfg.Services.ReviewURL))
	review.Get("/products/:productId/rating", r.createProxy(r.cfg.Services.ReviewURL))

	protected := review.Group("",
		middleware.JWTAuth(&r.cfg.JWT),
		middleware.RateLimiter(r.storage, &r.cfg.RateLimit),
	)
	protected.Post("", r.createProxy(r.cfg.Services.ReviewURL))
	protected.Put("/:id", r.createProxy(r.cfg.Services.ReviewURL))
	protected.Delete("/:id", r.createProxy(r.cfg.Services.ReviewURL))
	protected.Post("/:id/reply", r.createProxy(r.cfg.Services.ReviewURL))
	protected.Get("/my", r.createProxy(r.cfg.Services.ReviewURL))
}

func (r *Router) setupSearchRoutes() {
	search := r.app.Group("/api/v1/search")

	search.Get("/products", r.createProxy(r.cfg.Services.SearchURL))
	search.Get("/suggest", r.createProxy(r.cfg.Services.SearchURL))
	search.Get("/categories/:categoryId/products", r.createProxy(r.cfg.Services.SearchURL))
}

func (r *Router) setupCampaignRoutes() {
	campaign := r.app.Group("/api/v1/campaigns")

	campaign.Get("", r.createProxy(r.cfg.Services.CampaignURL))
	campaign.Get("/:id", r.createProxy(r.cfg.Services.CampaignURL))
	campaign.Get("/vouchers/public", r.createProxy(r.cfg.Services.CampaignURL))

	protected := campaign.Group("",
		middleware.JWTAuth(&r.cfg.JWT),
		middleware.RateLimiter(r.storage, &r.cfg.RateLimit),
	)
	protected.Post("/vouchers/claim", r.createProxy(r.cfg.Services.CampaignURL))
	protected.Get("/vouchers/my", r.createProxy(r.cfg.Services.CampaignURL))
	protected.Post("/vouchers/apply", r.createProxy(r.cfg.Services.CampaignURL))
	protected.Post("", r.createProxy(r.cfg.Services.CampaignURL))
	protected.Put("/:id", r.createProxy(r.cfg.Services.CampaignURL))
	protected.Delete("/:id", r.createProxy(r.cfg.Services.CampaignURL))
	protected.Post("/:id/vouchers", r.createProxy(r.cfg.Services.CampaignURL))
}

func (r *Router) setupAnalyticRoutes() {
	analytic := r.app.Group("/api/v1/analytics",
		middleware.JWTAuth(&r.cfg.JWT),
		middleware.RateLimiter(r.storage, &r.cfg.RateLimit),
	)

	analytic.Get("/dashboard", r.createProxy(r.cfg.Services.AnalyticURL))
	analytic.Get("/sales", r.createProxy(r.cfg.Services.AnalyticURL))
	analytic.Get("/products/:productId", r.createProxy(r.cfg.Services.AnalyticURL))
	analytic.Get("/users/:userId", r.createProxy(r.cfg.Services.AnalyticURL))
	analytic.Post("/events", r.createProxy(r.cfg.Services.AnalyticURL))
}

func (r *Router) setupInventoryRoutes() {
	inventory := r.app.Group("/api/v1/inventory",
		middleware.JWTAuth(&r.cfg.JWT),
		middleware.RateLimiter(r.storage, &r.cfg.RateLimit),
	)

	inventory.Get("/products/:skuId", r.createProxy(r.cfg.Services.InventoryURL))
	inventory.Put("/products/:skuId", r.createProxy(r.cfg.Services.InventoryURL))
	inventory.Get("/products/:skuId/history", r.createProxy(r.cfg.Services.InventoryURL))
}

func (r *Router) setupNotificationRoutes() {
	notification := r.app.Group("/api/v1/notifications",
		middleware.JWTAuth(&r.cfg.JWT),
		middleware.RateLimiter(r.storage, &r.cfg.RateLimit),
	)

	notification.Get("", r.createProxy(r.cfg.Services.NotificationURL))
	notification.Get("/unread-count", r.createProxy(r.cfg.Services.NotificationURL))
	notification.Put("/:id/read", r.createProxy(r.cfg.Services.NotificationURL))
	notification.Put("/read-all", r.createProxy(r.cfg.Services.NotificationURL))
	notification.Get("/preferences", r.createProxy(r.cfg.Services.NotificationURL))
	notification.Put("/preferences", r.createProxy(r.cfg.Services.NotificationURL))
}

func (r *Router) setupRealtimeRoutes() {
	wsHandler := func(c *fiber.Ctx) error {
		if r.cfg.Services.RealtimeURL == "" {
			return c.Status(fiber.StatusServiceUnavailable).JSON(fiber.Map{
				"error": "Realtime WebSocket service is currently unavailable",
			})
		}
		target := r.cfg.Services.RealtimeURL + "/ws"
		if qs := c.Request().URI().QueryString(); len(qs) > 0 {
			target += "?" + string(qs)
		}
		return proxy.Do(c, target)
	}

	r.app.Get("/api/v1/ws", wsHandler)
	r.app.Get("/ws", wsHandler)
}
