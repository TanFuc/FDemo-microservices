package router

import (
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
	"github.com/gofiber/fiber/v2/middleware/logger"
	"github.com/gofiber/fiber/v2/middleware/recover"

	"microservices/catalog/internal/config"
	httphandler "microservices/catalog/internal/handler/http"
	"microservices/catalog/internal/middleware"
)

type Router struct {
	app             *fiber.App
	cfg             *config.Config
	productHandler  *httphandler.ProductHandler
	categoryHandler *httphandler.CategoryHandler
	brandHandler    *httphandler.BrandHandler
	authMiddleware  *middleware.AuthMiddleware
}

func NewRouter(
	cfg *config.Config,
	productHandler *httphandler.ProductHandler,
	categoryHandler *httphandler.CategoryHandler,
	brandHandler *httphandler.BrandHandler,
	authMiddleware *middleware.AuthMiddleware,
) *Router {
	app := fiber.New(fiber.Config{
		ErrorHandler:          customErrorHandler,
		ReadTimeout:           cfg.App.ReadTimeout,
		WriteTimeout:          cfg.App.WriteTimeout,
		DisableStartupMessage: false,
	})

	return &Router{
		app:             app,
		cfg:             cfg,
		productHandler:  productHandler,
		categoryHandler: categoryHandler,
		brandHandler:    brandHandler,
		authMiddleware:  authMiddleware,
	}
}

func (r *Router) Setup() *fiber.App {
	// Middleware
	r.app.Use(recover.New())
	r.app.Use(logger.New())
	r.app.Use(cors.New(cors.Config{
		AllowOrigins: "*",
		AllowMethods: "GET,POST,PUT,PATCH,DELETE,OPTIONS",
		AllowHeaders: "Origin,Content-Type,Accept,Authorization",
	}))

	// Health check
	r.app.Get("/health", func(c *fiber.Ctx) error {
		return c.JSON(fiber.Map{
			"status": "ok",
		})
	})

	// Internal API routes (for service-to-service communication)
	internal := r.app.Group("/internal")
	internalProducts := internal.Group("/products")
	internalProducts.Get("/:productId/seller", r.productHandler.GetProductSeller)
	internalProducts.Put("/:productId/rating", r.productHandler.UpdateProductRating)

	// API routes
	api := r.app.Group("/api/v1")

	// Public routes (no auth required)
	// Products
	products := api.Group("/products")
	products.Get("/", r.productHandler.ListProducts)
	products.Get("/:id", r.productHandler.GetProduct)
	products.Get("/slug/:slug", r.productHandler.GetProductBySlug)

	// Categories
	categories := api.Group("/categories")
	categories.Get("/", r.categoryHandler.ListCategories)
	categories.Get("/:id", r.categoryHandler.GetCategory)
	categories.Get("/slug/:slug", r.categoryHandler.GetCategoryBySlug)

	// Brands
	brands := api.Group("/brands")
	brands.Get("/", r.brandHandler.ListBrands)
	brands.Get("/:id", r.brandHandler.GetBrand)
	brands.Get("/slug/:slug", r.brandHandler.GetBrandBySlug)

	// Protected routes (auth required)
	if r.authMiddleware != nil {
		// Product management
		products.Post("/", r.authMiddleware.RequireAuth(), r.authMiddleware.RequirePermission("product:create"), r.productHandler.CreateProduct)
		products.Put("/:id", r.authMiddleware.RequireAuth(), r.authMiddleware.RequirePermission("product:update"), r.productHandler.UpdateProduct)
		products.Patch("/:id/metadata", r.authMiddleware.RequireAuth(), r.authMiddleware.RequirePermission("product:update"), r.productHandler.UpdateProductMetadata)
		products.Delete("/:id", r.authMiddleware.RequireAuth(), r.authMiddleware.RequirePermission("product:delete"), r.productHandler.DeleteProduct)

		// Category management
		categories.Post("/", r.authMiddleware.RequireAuth(), r.authMiddleware.RequirePermission("category:create"), r.categoryHandler.CreateCategory)
		categories.Put("/:id", r.authMiddleware.RequireAuth(), r.authMiddleware.RequirePermission("category:update"), r.categoryHandler.UpdateCategory)
		categories.Delete("/:id", r.authMiddleware.RequireAuth(), r.authMiddleware.RequirePermission("category:delete"), r.categoryHandler.DeleteCategory)

		// Brand management
		brands.Post("/", r.authMiddleware.RequireAuth(), r.authMiddleware.RequirePermission("brand:create"), r.brandHandler.CreateBrand)
		brands.Put("/:id", r.authMiddleware.RequireAuth(), r.authMiddleware.RequirePermission("brand:update"), r.brandHandler.UpdateBrand)
		brands.Delete("/:id", r.authMiddleware.RequireAuth(), r.authMiddleware.RequirePermission("brand:delete"), r.brandHandler.DeleteBrand)
	} else {
		// No auth middleware - allow all operations (development mode)
		products.Post("/", r.productHandler.CreateProduct)
		products.Put("/:id", r.productHandler.UpdateProduct)
		products.Patch("/:id/metadata", r.productHandler.UpdateProductMetadata)
		products.Delete("/:id", r.productHandler.DeleteProduct)

		categories.Post("/", r.categoryHandler.CreateCategory)
		categories.Put("/:id", r.categoryHandler.UpdateCategory)
		categories.Delete("/:id", r.categoryHandler.DeleteCategory)

		brands.Post("/", r.brandHandler.CreateBrand)
		brands.Put("/:id", r.brandHandler.UpdateBrand)
		brands.Delete("/:id", r.brandHandler.DeleteBrand)
	}

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
