package middleware

import (
	"strings"

	"github.com/gofiber/fiber/v2"

	"microservices/catalog/internal/service"
	"microservices/catalog/pkg/response"
)

// AuthMiddleware handles authentication and authorization
type AuthMiddleware struct {
	authService service.AuthService
}

// NewAuthMiddleware creates a new auth middleware
func NewAuthMiddleware(authService service.AuthService) *AuthMiddleware {
	return &AuthMiddleware{
		authService: authService,
	}
}

// RequireAuth validates the JWT token and stores user info in context
func (m *AuthMiddleware) RequireAuth() fiber.Handler {
	return func(c *fiber.Ctx) error {
		// Get token from Authorization header
		authHeader := c.Get("Authorization")
		if authHeader == "" {
			return response.Unauthorized(c, "Missing authorization header")
		}

		// Extract Bearer token
		parts := strings.Split(authHeader, " ")
		if len(parts) != 2 || strings.ToLower(parts[0]) != "bearer" {
			return response.Unauthorized(c, "Invalid authorization header format")
		}

		token := parts[1]

		// Verify token
		user, err := m.authService.VerifyToken(c.Context(), token)
		if err != nil {
			return response.Unauthorized(c, "Failed to verify token")
		}
		if user == nil {
			return response.Unauthorized(c, "Invalid or expired token")
		}

		// Store user info in context
		c.Locals("userId", user.UserID)
		c.Locals("email", user.Email)
		c.Locals("role", user.Role)
		c.Locals("permissions", user.Permissions)
		c.Locals("token", token)

		return c.Next()
	}
}

// RequirePermission checks if the user has the specified permission
func (m *AuthMiddleware) RequirePermission(permission string) fiber.Handler {
	return func(c *fiber.Ctx) error {
		userID, ok := c.Locals("userId").(string)
		if !ok || userID == "" {
			return response.Unauthorized(c, "User not authenticated")
		}

		// Check permission via gRPC
		allowed, err := m.authService.CheckPermission(c.Context(), userID, permission)
		if err != nil {
			return response.InternalError(c, "Failed to check permission")
		}
		if !allowed {
			return response.Forbidden(c, "Permission denied: "+permission)
		}

		return c.Next()
	}
}

// RequireAnyPermission checks if the user has any of the specified permissions
func (m *AuthMiddleware) RequireAnyPermission(permissions ...string) fiber.Handler {
	return func(c *fiber.Ctx) error {
		userID, ok := c.Locals("userId").(string)
		if !ok || userID == "" {
			return response.Unauthorized(c, "User not authenticated")
		}

		for _, permission := range permissions {
			allowed, err := m.authService.CheckPermission(c.Context(), userID, permission)
			if err != nil {
				continue
			}
			if allowed {
				return c.Next()
			}
		}

		return response.Forbidden(c, "Permission denied")
	}
}

// OptionalAuth validates token if present but doesn't require it
func (m *AuthMiddleware) OptionalAuth() fiber.Handler {
	return func(c *fiber.Ctx) error {
		// Get token from Authorization header
		authHeader := c.Get("Authorization")
		if authHeader == "" {
			return c.Next()
		}

		// Extract Bearer token
		parts := strings.Split(authHeader, " ")
		if len(parts) != 2 || strings.ToLower(parts[0]) != "bearer" {
			return c.Next()
		}

		token := parts[1]

		// Verify token
		user, err := m.authService.VerifyToken(c.Context(), token)
		if err == nil && user != nil {
			// Store user info in context
			c.Locals("userId", user.UserID)
			c.Locals("email", user.Email)
			c.Locals("role", user.Role)
			c.Locals("permissions", user.Permissions)
			c.Locals("token", token)
		}

		return c.Next()
	}
}

// GetUserID returns the user ID from context
func GetUserID(c *fiber.Ctx) string {
	if userID, ok := c.Locals("userId").(string); ok {
		return userID
	}
	return ""
}

// GetEmail returns the user email from context
func GetEmail(c *fiber.Ctx) string {
	if email, ok := c.Locals("email").(string); ok {
		return email
	}
	return ""
}

// GetRole returns the user role from context
func GetRole(c *fiber.Ctx) string {
	if role, ok := c.Locals("role").(string); ok {
		return role
	}
	return ""
}

// GetPermissions returns the user permissions from context
func GetPermissions(c *fiber.Ctx) []string {
	if permissions, ok := c.Locals("permissions").([]string); ok {
		return permissions
	}
	return nil
}
