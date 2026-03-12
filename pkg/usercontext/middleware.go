package usercontext

import (
	"context"
	"fmt"

	"github.com/gofiber/fiber/v2"
)

// Context keys for storing user context
type contextKey string

const (
	UserContextKey contextKey = "user_context"
	UserIDKey      contextKey = "user_id"
	UserRoleKey    contextKey = "user_role"
	ShopIDKey      contextKey = "shop_id"
)

// Header names for user context propagation
const (
	HeaderUserID    = "X-User-ID"
	HeaderUserRole  = "X-User-Role"
	HeaderUserEmail = "X-User-Email"
	HeaderShopID    = "X-Shop-ID"
)

// UserContextRepository is the interface for fetching/storing user context
type UserContextRepository interface {
	// FindByUserID retrieves user context from local database
	FindByUserID(ctx context.Context, userID string) (*UserContext, error)
	// Upsert inserts or updates user context
	Upsert(ctx context.Context, user *UserContext) error
}

// UserContextFallback is the interface for fallback when user not in local DB
type UserContextFallback interface {
	// FetchUserContext fetches user context from Auth/Profile service
	FetchUserContext(ctx context.Context, userID string) (*UserContext, error)
}

// MiddlewareConfig represents configuration for user context middleware
type MiddlewareConfig struct {
	// Repository for local user context lookup
	Repository UserContextRepository

	// Fallback for fetching user context when not found locally
	Fallback UserContextFallback

	// RequireAuth requires authentication (returns 401 if no user ID)
	RequireAuth bool

	// RequireActiveUser requires the user to be active
	RequireActiveUser bool

	// Skipper defines a function to skip middleware
	Skipper func(c *fiber.Ctx) bool

	// ErrorHandler handles errors
	ErrorHandler func(c *fiber.Ctx, err error) error

	// UnauthorizedHandler handles unauthorized requests
	UnauthorizedHandler func(c *fiber.Ctx) error

	// ForbiddenHandler handles forbidden requests (inactive user)
	ForbiddenHandler func(c *fiber.Ctx) error
}

// DefaultMiddlewareConfig returns the default configuration
func DefaultMiddlewareConfig() MiddlewareConfig {
	return MiddlewareConfig{
		RequireAuth:       false,
		RequireActiveUser: false,
		Skipper:           defaultSkipper,
		ErrorHandler:      defaultErrorHandler,
		UnauthorizedHandler: func(c *fiber.Ctx) error {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"success": false,
				"code":    "UNAUTHORIZED",
				"message": "Authentication required",
			})
		},
		ForbiddenHandler: func(c *fiber.Ctx) error {
			return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
				"success": false,
				"code":    "FORBIDDEN",
				"message": "User account is not active",
			})
		},
	}
}

// NewMiddleware creates a new user context middleware with default config
func NewMiddleware() fiber.Handler {
	return NewMiddlewareWithConfig(DefaultMiddlewareConfig())
}

// NewMiddlewareWithConfig creates a new user context middleware with custom config
func NewMiddlewareWithConfig(config MiddlewareConfig) fiber.Handler {
	// Apply defaults
	if config.Skipper == nil {
		config.Skipper = defaultSkipper
	}
	if config.ErrorHandler == nil {
		config.ErrorHandler = defaultErrorHandler
	}
	if config.UnauthorizedHandler == nil {
		config.UnauthorizedHandler = func(c *fiber.Ctx) error {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"success": false,
				"code":    "UNAUTHORIZED",
				"message": "Authentication required",
			})
		}
	}
	if config.ForbiddenHandler == nil {
		config.ForbiddenHandler = func(c *fiber.Ctx) error {
			return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
				"success": false,
				"code":    "FORBIDDEN",
				"message": "User account is not active",
			})
		}
	}

	return func(c *fiber.Ctx) error {
		// Skip if configured
		if config.Skipper(c) {
			return c.Next()
		}

		// Extract user ID from header
		userID := c.Get(HeaderUserID)
		if userID == "" {
			if config.RequireAuth {
				return config.UnauthorizedHandler(c)
			}
			return c.Next()
		}

		// Store basic info in locals
		c.Locals("user_id", userID)

		// Extract other headers
		userRole := c.Get(HeaderUserRole)
		if userRole != "" {
			c.Locals("user_role", userRole)
		}

		shopID := c.Get(HeaderShopID)
		if shopID != "" {
			c.Locals("shop_id", shopID)
		}

		// If repository is configured, try to load full user context
		if config.Repository != nil {
			userCtx, err := config.Repository.FindByUserID(c.Context(), userID)
			if err != nil {
				// Not found in local DB - try fallback
				if config.Fallback != nil {
					userCtx, err = config.Fallback.FetchUserContext(c.Context(), userID)
					if err == nil && userCtx != nil {
						// Self-healing: save to local DB
						_ = config.Repository.Upsert(c.Context(), userCtx)
					}
				}
			}

			if userCtx != nil {
				// Store full context
				c.Locals("user_context", userCtx)
				c.Locals("user_role", string(userCtx.Role))
				if userCtx.ShopID != "" {
					c.Locals("shop_id", userCtx.ShopID)
				}

				// Check if user is active
				if config.RequireActiveUser && !userCtx.IsActive() {
					return config.ForbiddenHandler(c)
				}
			}
		}

		return c.Next()
	}
}

// ExtractUserMiddleware is a simpler middleware that only extracts headers to locals
func ExtractUserMiddleware() fiber.Handler {
	return func(c *fiber.Ctx) error {
		// Extract user ID from header
		userID := c.Get(HeaderUserID)
		if userID != "" {
			c.Locals("user_id", userID)
		}

		// Extract role from header
		userRole := c.Get(HeaderUserRole)
		if userRole != "" {
			c.Locals("user_role", userRole)
		}

		// Extract email from header
		userEmail := c.Get(HeaderUserEmail)
		if userEmail != "" {
			c.Locals("user_email", userEmail)
		}

		// Extract shop ID from header
		shopID := c.Get(HeaderShopID)
		if shopID != "" {
			c.Locals("shop_id", shopID)
		}

		return c.Next()
	}
}

// RequireAuth creates middleware that requires authentication
func RequireAuth() fiber.Handler {
	return func(c *fiber.Ctx) error {
		userID := c.Locals("user_id")
		if userID == nil || userID == "" {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"success": false,
				"code":    "UNAUTHORIZED",
				"message": "Authentication required",
			})
		}
		return c.Next()
	}
}

// RequireRole creates middleware that requires a specific role
func RequireRole(roles ...UserRole) fiber.Handler {
	return func(c *fiber.Ctx) error {
		userRole := c.Locals("user_role")
		if userRole == nil {
			return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
				"success": false,
				"code":    "FORBIDDEN",
				"message": "Role not found",
			})
		}

		roleStr, ok := userRole.(string)
		if !ok {
			return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
				"success": false,
				"code":    "FORBIDDEN",
				"message": "Invalid role",
			})
		}

		for _, r := range roles {
			if string(r) == roleStr {
				return c.Next()
			}
		}

		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
			"success": false,
			"code":    "FORBIDDEN",
			"message": "Insufficient permissions",
		})
	}
}

// RequireSeller creates middleware that requires seller role
func RequireSeller() fiber.Handler {
	return RequireRole(RoleSeller, RoleAdmin)
}

// RequireAdmin creates middleware that requires admin role
func RequireAdmin() fiber.Handler {
	return RequireRole(RoleAdmin)
}

// GetUserContext retrieves the full user context from fiber context
func GetUserContext(c *fiber.Ctx) *UserContext {
	ctx := c.Locals("user_context")
	if ctx == nil {
		return nil
	}
	if uc, ok := ctx.(*UserContext); ok {
		return uc
	}
	return nil
}

// GetUserID retrieves the user ID from fiber context
func GetUserID(c *fiber.Ctx) string {
	id := c.Locals("user_id")
	if id == nil {
		return ""
	}
	return fmt.Sprintf("%v", id)
}

// GetUserRole retrieves the user role from fiber context
func GetUserRole(c *fiber.Ctx) string {
	role := c.Locals("user_role")
	if role == nil {
		return ""
	}
	return fmt.Sprintf("%v", role)
}

// GetShopID retrieves the shop ID from fiber context
func GetShopID(c *fiber.Ctx) string {
	id := c.Locals("shop_id")
	if id == nil {
		return ""
	}
	return fmt.Sprintf("%v", id)
}

// Helper functions

func defaultSkipper(c *fiber.Ctx) bool {
	path := c.Path()
	return path == "/health" || path == "/health/live" || path == "/health/ready" || path == "/metrics"
}

func defaultErrorHandler(c *fiber.Ctx, err error) error {
	return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
		"success": false,
		"code":    "INTERNAL_ERROR",
		"message": "Failed to load user context",
		"error":   err.Error(),
	})
}
