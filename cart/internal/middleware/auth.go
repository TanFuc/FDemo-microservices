package middleware

import (
	"strings"

	"github.com/gofiber/fiber/v2"

	"microservices/pkg/authclient"
)

// AuthMiddleware provides authentication and authorization middleware.
type AuthMiddleware struct {
	client *authclient.Client
}

// NewAuthMiddleware creates a new auth middleware instance.
func NewAuthMiddleware(client *authclient.Client) *AuthMiddleware {
	return &AuthMiddleware{client: client}
}

// RequireAuth validates JWT token and stores user info in context.
func (m *AuthMiddleware) RequireAuth() fiber.Handler {
	return func(c *fiber.Ctx) error {
		authHeader := c.Get("Authorization")
		if authHeader == "" {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"success": false,
				"code":    "UNAUTHORIZED",
				"message": "Missing authorization header",
			})
		}

		token := strings.TrimPrefix(authHeader, "Bearer ")
		if token == authHeader {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"success": false,
				"code":    "UNAUTHORIZED",
				"message": "Invalid authorization header format",
			})
		}

		resp, err := m.client.ValidateToken(c.Context(), token)
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"success": false,
				"code":    "INTERNAL_ERROR",
				"message": "Failed to validate token",
			})
		}

		if !resp.Valid {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"success": false,
				"code":    "UNAUTHORIZED",
				"message": "Invalid or expired token",
			})
		}

		c.Locals("userId", resp.UserId)
		c.Locals("user_id", resp.UserId)
		c.Locals("email", resp.Email)
		c.Locals("token", token)

		return c.Next()
	}
}

// RequirePermission checks if user has specific permission.
func (m *AuthMiddleware) RequirePermission(resource, action string) fiber.Handler {
	return func(c *fiber.Ctx) error {
		token, ok := c.Locals("token").(string)
		if !ok || token == "" {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"success": false,
				"code":    "UNAUTHORIZED",
				"message": "Authentication required",
			})
		}

		resp, err := m.client.CheckPermission(c.Context(), token, resource, action, nil)
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"success": false,
				"code":    "INTERNAL_ERROR",
				"message": "Failed to check permission",
			})
		}

		if !resp.Allowed {
			return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
				"success": false,
				"code":    "FORBIDDEN",
				"message": "Access denied",
				"reason":  resp.Reason,
			})
		}

		c.Locals("userRoles", resp.Roles)
		c.Locals("userPermissions", resp.Permissions)

		return c.Next()
	}
}

// RequireRole checks if user has specific role.
func (m *AuthMiddleware) RequireRole(role string) fiber.Handler {
	return func(c *fiber.Ctx) error {
		token, ok := c.Locals("token").(string)
		if !ok || token == "" {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"success": false,
				"code":    "UNAUTHORIZED",
				"message": "Authentication required",
			})
		}

		resp, err := m.client.CheckPermission(c.Context(), token, "", "", nil)
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"success": false,
				"code":    "INTERNAL_ERROR",
				"message": "Failed to check role",
			})
		}

		hasRole := false
		for _, r := range resp.Roles {
			if r == role {
				hasRole = true
				break
			}
		}

		if !hasRole {
			return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
				"success": false,
				"code":    "FORBIDDEN",
				"message": "Required role not found: " + role,
			})
		}

		c.Locals("userRoles", resp.Roles)
		c.Locals("userPermissions", resp.Permissions)

		return c.Next()
	}
}

// ValidateUserOwnership validates that the authenticated user owns the resource.
func (m *AuthMiddleware) ValidateUserOwnership() fiber.Handler {
	return func(c *fiber.Ctx) error {
		authUserID, ok := c.Locals("userId").(string)
		if !ok || authUserID == "" {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"success": false,
				"code":    "UNAUTHORIZED",
				"message": "Authentication required",
			})
		}

		pathUserID := c.Params("userId")
		if pathUserID == "" {
			return c.Next()
		}

		if authUserID != pathUserID {
			return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
				"success": false,
				"code":    "FORBIDDEN",
				"message": "Access denied: you can only access your own cart",
			})
		}

		return c.Next()
	}
}

// GetUserID retrieves the user ID from Fiber context.
func GetUserID(c *fiber.Ctx) string {
	if userID := c.Locals("userId"); userID != nil {
		return userID.(string)
	}
	return ""
}

// GetEmail retrieves the email from Fiber context.
func GetEmail(c *fiber.Ctx) string {
	if email := c.Locals("email"); email != nil {
		return email.(string)
	}
	return ""
}

// GetRoles retrieves the roles from Fiber context.
func GetRoles(c *fiber.Ctx) []string {
	if roles := c.Locals("userRoles"); roles != nil {
		return roles.([]string)
	}
	return nil
}
