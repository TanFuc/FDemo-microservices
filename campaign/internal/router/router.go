package router

import (
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
	"github.com/gofiber/fiber/v2/middleware/logger"
	"github.com/gofiber/fiber/v2/middleware/recover"

	"microservices/campaign/internal/config"
	httphandler "microservices/campaign/internal/handler/http"
	"microservices/campaign/internal/middleware"
)

// Router handles HTTP routing.
type Router struct {
	app              *fiber.App
	cfg              *config.Config
	campaignHandler  *httphandler.CampaignHandler
	voucherHandler   *httphandler.VoucherHandler
	authMiddleware   *middleware.AuthMiddleware
}

// NewRouter creates a new router.
func NewRouter(
	cfg *config.Config,
	campaignHandler *httphandler.CampaignHandler,
	voucherHandler *httphandler.VoucherHandler,
	authMiddleware *middleware.AuthMiddleware,
) *Router {
	app := fiber.New(fiber.Config{
		AppName:      cfg.App.Name,
		ErrorHandler: customErrorHandler,
	})

	return &Router{
		app:              app,
		cfg:              cfg,
		campaignHandler:  campaignHandler,
		voucherHandler:   voucherHandler,
		authMiddleware:   authMiddleware,
	}
}

// Setup configures all routes and middleware.
func (r *Router) Setup() *fiber.App {
	// Global middleware
	r.app.Use(recover.New())
	r.app.Use(logger.New(logger.Config{
		Format:     "${time} | ${status} | ${latency} | ${method} | ${path}\n",
		TimeFormat: "2006-01-02 15:04:05",
	}))
	r.app.Use(cors.New(cors.Config{
		AllowOrigins: "*",
		AllowMethods: "GET,POST,PUT,DELETE,OPTIONS",
		AllowHeaders: "Origin,Content-Type,Accept,Authorization",
	}))

	// Health check
	r.app.Get("/health", r.campaignHandler.HealthCheck)

	// API routes
	api := r.app.Group("/api/v1")

	// Campaign routes - public read, auth required for write
	campaigns := api.Group("/campaigns")
	campaigns.Get("/", r.campaignHandler.ListCampaigns)
	campaigns.Get("/active", r.campaignHandler.ListActiveCampaigns)
	campaigns.Get("/:id", r.campaignHandler.GetCampaign)

	// Protected campaign routes (admin only)
	if r.authMiddleware != nil {
		protectedCampaigns := campaigns.Group("")
		protectedCampaigns.Use(r.authMiddleware.RequireAuth())
		protectedCampaigns.Use(r.authMiddleware.RequireRole("admin"))
		protectedCampaigns.Post("/", r.campaignHandler.CreateCampaign)
	}

	// Voucher routes
	vouchers := api.Group("/vouchers")
	vouchers.Get("/code/:code", r.voucherHandler.GetVoucherByCode)
	vouchers.Get("/:id", r.voucherHandler.GetVoucher)

	// Calculate cart - public endpoint
	vouchers.Post("/calculate", r.voucherHandler.CalculateCart)

	// Protected voucher routes
	if r.authMiddleware != nil {
		protectedVouchers := vouchers.Group("")
		protectedVouchers.Use(r.authMiddleware.RequireAuth())

		// User voucher operations
		protectedVouchers.Post("/claim", r.voucherHandler.ClaimVoucher)
		protectedVouchers.Get("/user", r.voucherHandler.GetUserVouchers)

		// Admin voucher operations
		adminVouchers := protectedVouchers.Group("")
		adminVouchers.Use(r.authMiddleware.RequireRole("admin"))
		adminVouchers.Post("/", r.voucherHandler.CreateVoucher)
		adminVouchers.Post("/:code/initialize", r.voucherHandler.InitializeVoucherStock)
	}

	return r.app
}

// App returns the fiber app instance.
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
