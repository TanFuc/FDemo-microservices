package router

import (
	"microservices/inventory/internal/handler/http"
	"microservices/inventory/internal/middleware"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
	"github.com/gofiber/fiber/v2/middleware/logger"
	"github.com/gofiber/fiber/v2/middleware/recover"
)

type Router struct {
	app                *fiber.App
	inventoryHandler   *http.InventoryHandler
	reservationHandler *http.ReservationHandler
	authMiddleware     *middleware.AuthMiddleware
}

func New(
	inventoryHandler *http.InventoryHandler,
	reservationHandler *http.ReservationHandler,
	authMiddleware *middleware.AuthMiddleware,
) *Router {
	return &Router{
		inventoryHandler:   inventoryHandler,
		reservationHandler: reservationHandler,
		authMiddleware:     authMiddleware,
	}
}

func (r *Router) Setup() *fiber.App {
	r.app = fiber.New(fiber.Config{
		ErrorHandler: customErrorHandler,
	})

	// Global middleware
	r.app.Use(recover.New())
	r.app.Use(logger.New())
	r.app.Use(cors.New())

	// Health check
	r.app.Get("/health", func(c *fiber.Ctx) error {
		return c.JSON(fiber.Map{
			"status":  "healthy",
			"service": "inventory-service",
		})
	})

	// API routes
	api := r.app.Group("/api/v1")
	r.setupInventoryRoutes(api)
	r.setupReservationRoutes(api)

	return r.app
}

func (r *Router) setupInventoryRoutes(api fiber.Router) {
	inventory := api.Group("/inventory")

	// Public routes (with optional auth for read operations)
	inventory.Get("", r.authMiddleware.OptionalAuth(), r.inventoryHandler.ListInventory)
	inventory.Get("/:sku_id", r.authMiddleware.OptionalAuth(), r.inventoryHandler.GetInventory)

	// Protected routes (require authentication and permissions)
	inventory.Post("",
		r.authMiddleware.RequireAuth(),
		r.authMiddleware.RequirePermission("inventory:create"),
		r.inventoryHandler.CreateInventory,
	)

	inventory.Put("/:sku_id",
		r.authMiddleware.RequireAuth(),
		r.authMiddleware.RequirePermission("inventory:update"),
		r.inventoryHandler.UpdateInventory,
	)

	inventory.Delete("/:sku_id",
		r.authMiddleware.RequireAuth(),
		r.authMiddleware.RequirePermission("inventory:delete"),
		r.inventoryHandler.DeleteInventory,
	)

	inventory.Post("/sync",
		r.authMiddleware.RequireAuth(),
		r.authMiddleware.RequirePermission("inventory:sync"),
		r.inventoryHandler.SyncInventory,
	)
}

func (r *Router) setupReservationRoutes(api fiber.Router) {
	reservations := api.Group("/reservations")

	// Protected routes - require authentication for all reservation operations
	reservations.Use(r.authMiddleware.RequireAuth())

	// List/Get reservations
	reservations.Get("",
		r.authMiddleware.RequireAnyPermission("reservation:read", "inventory:read"),
		r.reservationHandler.ListReservations,
	)

	reservations.Get("/:order_id",
		r.authMiddleware.RequireAnyPermission("reservation:read", "inventory:read"),
		r.reservationHandler.GetReservation,
	)

	// Stock operations
	reservations.Post("/reserve",
		r.authMiddleware.RequirePermission("reservation:create"),
		r.reservationHandler.ReserveStock,
	)

	reservations.Post("/confirm",
		r.authMiddleware.RequirePermission("reservation:confirm"),
		r.reservationHandler.ConfirmStock,
	)

	reservations.Post("/release",
		r.authMiddleware.RequirePermission("reservation:release"),
		r.reservationHandler.ReleaseStock,
	)
}

func customErrorHandler(c *fiber.Ctx, err error) error {
	code := fiber.StatusInternalServerError
	message := "internal server error"

	if e, ok := err.(*fiber.Error); ok {
		code = e.Code
		message = e.Message
	}

	return c.Status(code).JSON(fiber.Map{
		"success": false,
		"error":   message,
	})
}
