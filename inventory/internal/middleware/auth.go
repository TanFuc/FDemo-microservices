package middleware

import (
	"strings"

	"microservices/inventory/internal/service"
	"microservices/inventory/pkg/response"

	"github.com/gofiber/fiber/v2"
)

const (
	AuthUserKey = "auth_user"
)

type AuthMiddleware struct {
	authService service.AuthService
}

func NewAuthMiddleware(authService service.AuthService) *AuthMiddleware {
	return &AuthMiddleware{authService: authService}
}

// RequireAuth validates the JWT token and sets the user in context
func (m *AuthMiddleware) RequireAuth() fiber.Handler {
	return func(c *fiber.Ctx) error {
		token := extractToken(c)
		if token == "" {
			return response.Unauthorized(c, "missing authorization token")
		}

		user, err := m.authService.VerifyToken(c.Context(), token)
		if err != nil {
			return response.Unauthorized(c, "invalid token")
		}
		if user == nil {
			return response.Unauthorized(c, "invalid token")
		}

		c.Locals(AuthUserKey, user)
		return c.Next()
	}
}

// RequirePermission checks if the user has the specified permission
func (m *AuthMiddleware) RequirePermission(permission string) fiber.Handler {
	return func(c *fiber.Ctx) error {
		user, ok := c.Locals(AuthUserKey).(*service.AuthUser)
		if !ok || user == nil {
			return response.Unauthorized(c, "unauthorized")
		}

		// Check if user has permission locally first
		for _, p := range user.Permissions {
			if p == permission {
				return c.Next()
			}
		}

		// Check via auth service
		hasPermission, err := m.authService.CheckPermission(c.Context(), user.UserID, permission)
		if err != nil {
			return response.InternalError(c, err)
		}

		if !hasPermission {
			return response.Forbidden(c, "insufficient permissions")
		}

		return c.Next()
	}
}

// RequireAnyPermission checks if the user has any of the specified permissions
func (m *AuthMiddleware) RequireAnyPermission(permissions ...string) fiber.Handler {
	return func(c *fiber.Ctx) error {
		user, ok := c.Locals(AuthUserKey).(*service.AuthUser)
		if !ok || user == nil {
			return response.Unauthorized(c, "unauthorized")
		}

		// Check local permissions first
		for _, p := range user.Permissions {
			for _, required := range permissions {
				if p == required {
					return c.Next()
				}
			}
		}

		// Check via auth service
		for _, permission := range permissions {
			hasPermission, err := m.authService.CheckPermission(c.Context(), user.UserID, permission)
			if err != nil {
				continue
			}
			if hasPermission {
				return c.Next()
			}
		}

		return response.Forbidden(c, "insufficient permissions")
	}
}

// OptionalAuth extracts user info if token present, but doesn't require it
func (m *AuthMiddleware) OptionalAuth() fiber.Handler {
	return func(c *fiber.Ctx) error {
		token := extractToken(c)
		if token == "" {
			return c.Next()
		}

		user, err := m.authService.VerifyToken(c.Context(), token)
		if err == nil && user != nil {
			c.Locals(AuthUserKey, user)
		}

		return c.Next()
	}
}

func extractToken(c *fiber.Ctx) string {
	authHeader := c.Get("Authorization")
	if authHeader == "" {
		return ""
	}

	parts := strings.Split(authHeader, " ")
	if len(parts) != 2 || strings.ToLower(parts[0]) != "bearer" {
		return ""
	}

	return parts[1]
}

// GetAuthUser retrieves the authenticated user from context
func GetAuthUser(c *fiber.Ctx) *service.AuthUser {
	user, ok := c.Locals(AuthUserKey).(*service.AuthUser)
	if !ok {
		return nil
	}
	return user
}
