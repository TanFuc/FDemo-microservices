package router

import (
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
	"github.com/gofiber/fiber/v2/middleware/helmet"
	"github.com/gofiber/fiber/v2/middleware/logger"
	"github.com/gofiber/fiber/v2/middleware/recover"

	"microservices/profile/internal/config"
	httphandler "microservices/profile/internal/handler/http"
)

type Router struct {
	app             *fiber.App
	cfg             *config.Config
	profileHandler  *httphandler.ProfileHandler
	internalHandler *httphandler.InternalHandler
	healthHandler   *httphandler.HealthHandler
}

func NewRouter(
	cfg *config.Config,
	profileHandler *httphandler.ProfileHandler,
	internalHandler *httphandler.InternalHandler,
	healthHandler *httphandler.HealthHandler,
) *Router {
	app := fiber.New(fiber.Config{
		ErrorHandler: customErrorHandler,
	})

	return &Router{
		app:             app,
		cfg:             cfg,
		profileHandler:  profileHandler,
		internalHandler: internalHandler,
		healthHandler:   healthHandler,
	}
}

func (r *Router) Setup() *fiber.App {
	// Global middleware
	r.app.Use(helmet.New())
	r.app.Use(cors.New(cors.Config{
		AllowOrigins:     "*",
		AllowMethods:     "GET,POST,PUT,PATCH,DELETE,OPTIONS",
		AllowHeaders:     "Origin,Content-Type,Accept,Authorization,X-User-ID,X-Internal-Service-Key,X-Request-ID",
		AllowCredentials: true,
	}))
	r.app.Use(logger.New())
	r.app.Use(recover.New())

	// Health routes (public)
	r.app.Get("/health", r.healthHandler.Health)
	r.app.Get("/health/live", r.healthHandler.Liveness)
	r.app.Get("/health/ready", r.healthHandler.Readiness)

	// API routes
	api := r.app.Group("/" + r.cfg.App.APIPrefix)

	// Profile routes
	profiles := api.Group("/profiles")
	profiles.Get("/me", r.profileHandler.GetMyProfile)
	profiles.Patch("/me", r.profileHandler.UpdateMyProfile)
	profiles.Post("/me/shop", r.profileHandler.RegisterShop)
	profiles.Patch("/me/shop", r.profileHandler.UpdateShop)
	profiles.Get("/me/addresses", r.profileHandler.GetAddresses)
	profiles.Post("/me/addresses", r.profileHandler.CreateAddress)
	profiles.Get("/me/addresses/:id", r.profileHandler.GetAddress)
	profiles.Patch("/me/addresses/:id/set-default", r.profileHandler.SetDefaultAddress)
	profiles.Delete("/me/addresses/:id", r.profileHandler.DeleteAddress)
	profiles.Post("/me/affiliate", r.profileHandler.RegisterAffiliate)

	// Internal routes
	internal := api.Group("/internal")
	internal.Get("/users/:userId", r.internalHandler.GetUserInfo)
	internal.Get("/health", r.internalHandler.Health)

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
