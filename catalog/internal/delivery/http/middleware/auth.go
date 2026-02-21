package middleware

import (
	"strings"

	"microservices/catalog/internal/infrastructure/grpc"

	"github.com/gofiber/fiber/v2"
)

// AuthMiddleware handles authentication and authorization
type AuthMiddleware struct {
	authClient *grpc.AuthClient
}

// NewAuthMiddleware creates a new auth middleware
func NewAuthMiddleware(authClient *grpc.AuthClient) *AuthMiddleware {
	return &AuthMiddleware{authClient: authClient}
}

// RequireAuth validates JWT token
func (m *AuthMiddleware) RequireAuth() fiber.Handler {
	return func(c *fiber.Ctx) error {
		// Extract token from Authorization header
		authHeader := c.Get("Authorization")
		if authHeader == "" {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"error": "Missing authorization header",
			})
		}

		// Remove "Bearer " prefix
		token := strings.TrimPrefix(authHeader, "Bearer ")
		if token == authHeader {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"error": "Invalid authorization header format",
			})
		}

		// Validate token via gRPC
		resp, err := m.authClient.ValidateToken(c.Context(), token)
		if err != nil || !resp.Valid {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"error": "Invalid or expired token",
			})
		}

		// Store user info in context
		c.Locals("userId", resp.UserId)
		c.Locals("email", resp.Email)
		c.Locals("token", token)

		return c.Next()
	}
}

// RequirePermission checks if user has specific permission
func (m *AuthMiddleware) RequirePermission(resource, action string) fiber.Handler {
	return func(c *fiber.Ctx) error {
		token, ok := c.Locals("token").(string)
		if !ok || token == "" {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"error": "Authentication required",
			})
		}

		// Check permission via gRPC
		resp, err := m.authClient.CheckPermission(c.Context(), token, resource, action, nil)
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"error": "Failed to check permission",
			})
		}

		if !resp.Allowed {
			return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
				"error":  "Access denied",
				"reason": resp.Reason,
			})
		}

		// Store permission info in context
		c.Locals("userRoles", resp.Roles)
		c.Locals("userPermissions", resp.Permissions)

		return c.Next()
	}
}

// RequireAnyPermission checks if user has any of the specified permissions
func (m *AuthMiddleware) RequireAnyPermission(permissions ...struct{ Resource, Action string }) fiber.Handler {
	return func(c *fiber.Ctx) error {
		token, ok := c.Locals("token").(string)
		if !ok || token == "" {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"error": "Authentication required",
			})
		}

		// Check each permission
		for _, perm := range permissions {
			resp, err := m.authClient.CheckPermission(c.Context(), token, perm.Resource, perm.Action, nil)
			if err == nil && resp.Allowed {
				c.Locals("userRoles", resp.Roles)
				c.Locals("userPermissions", resp.Permissions)
				return c.Next()
			}
		}

		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
			"error": "Access denied",
		})
	}
}

// OptionalAuth validates JWT token if present but doesn't require it
func (m *AuthMiddleware) OptionalAuth() fiber.Handler {
	return func(c *fiber.Ctx) error {
		authHeader := c.Get("Authorization")
		if authHeader == "" {
			return c.Next()
		}

		token := strings.TrimPrefix(authHeader, "Bearer ")
		if token == authHeader {
			return c.Next()
		}

		resp, err := m.authClient.ValidateToken(c.Context(), token)
		if err == nil && resp.Valid {
			c.Locals("userId", resp.UserId)
			c.Locals("email", resp.Email)
			c.Locals("token", token)
		}

		return c.Next()
	}
}
