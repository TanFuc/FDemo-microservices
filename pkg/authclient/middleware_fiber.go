package authclient

import (
	"strings"

	"github.com/gofiber/fiber/v2"
)

// FiberMiddleware provides authentication and authorization middleware for Fiber
type FiberMiddleware struct {
	client *Client
}

// NewFiberMiddleware creates a new Fiber middleware instance
func NewFiberMiddleware(client *Client) *FiberMiddleware {
	return &FiberMiddleware{client: client}
}

// RequireAuth validates JWT token and stores user info in context
func (m *FiberMiddleware) RequireAuth() fiber.Handler {
	return func(c *fiber.Ctx) error {
		// Extract token from Authorization header
		authHeader := c.Get("Authorization")
		if authHeader == "" {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"success": false,
				"code":    "UNAUTHORIZED",
				"message": "Missing authorization header",
			})
		}

		// Remove "Bearer " prefix
		token := strings.TrimPrefix(authHeader, "Bearer ")
		if token == authHeader {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"success": false,
				"code":    "UNAUTHORIZED",
				"message": "Invalid authorization header format",
			})
		}

		// Validate token via gRPC
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

		// Store user info in context
		c.Locals("userId", resp.UserId)
		c.Locals("user_id", resp.UserId) // Also store with underscore for compatibility
		c.Locals("email", resp.Email)
		c.Locals("token", token)

		return c.Next()
	}
}

// RequirePermission checks if user has specific permission
func (m *FiberMiddleware) RequirePermission(resource, action string) fiber.Handler {
	return func(c *fiber.Ctx) error {
		token, ok := c.Locals("token").(string)
		if !ok || token == "" {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"success": false,
				"code":    "UNAUTHORIZED",
				"message": "Authentication required",
			})
		}

		// Check permission via gRPC
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

		// Store roles and permissions in context
		c.Locals("userRoles", resp.Roles)
		c.Locals("userPermissions", resp.Permissions)

		return c.Next()
	}
}

// RequireRole checks if user has specific role
func (m *FiberMiddleware) RequireRole(role string) fiber.Handler {
	return func(c *fiber.Ctx) error {
		token, ok := c.Locals("token").(string)
		if !ok || token == "" {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"success": false,
				"code":    "UNAUTHORIZED",
				"message": "Authentication required",
			})
		}

		// Check permission via gRPC (will return roles)
		resp, err := m.client.CheckPermission(c.Context(), token, "", "", nil)
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"success": false,
				"code":    "INTERNAL_ERROR",
				"message": "Failed to check role",
			})
		}

		// Check if user has the required role
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

// RequireAnyRole checks if user has any of the specified roles
func (m *FiberMiddleware) RequireAnyRole(roles ...string) fiber.Handler {
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
				"message": "Failed to check roles",
			})
		}

		roleSet := make(map[string]bool)
		for _, r := range resp.Roles {
			roleSet[r] = true
		}

		hasRole := false
		for _, r := range roles {
			if roleSet[r] {
				hasRole = true
				break
			}
		}

		if !hasRole {
			return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
				"success": false,
				"code":    "FORBIDDEN",
				"message": "Required role not found",
			})
		}

		c.Locals("userRoles", resp.Roles)
		c.Locals("userPermissions", resp.Permissions)

		return c.Next()
	}
}

// OptionalAuth validates JWT token if present but doesn't require it
func (m *FiberMiddleware) OptionalAuth() fiber.Handler {
	return func(c *fiber.Ctx) error {
		authHeader := c.Get("Authorization")
		if authHeader == "" {
			return c.Next()
		}

		token := strings.TrimPrefix(authHeader, "Bearer ")
		if token == authHeader {
			return c.Next()
		}

		resp, err := m.client.ValidateToken(c.Context(), token)
		if err == nil && resp.Valid {
			c.Locals("userId", resp.UserId)
			c.Locals("user_id", resp.UserId)
			c.Locals("email", resp.Email)
			c.Locals("token", token)
		}

		return c.Next()
	}
}

// GetUserID retrieves the user ID from Fiber context
func GetUserID(c *fiber.Ctx) string {
	if userID := c.Locals("userId"); userID != nil {
		return userID.(string)
	}
	return ""
}

// GetEmail retrieves the email from Fiber context
func GetEmail(c *fiber.Ctx) string {
	if email := c.Locals("email"); email != nil {
		return email.(string)
	}
	return ""
}

// GetRoles retrieves the roles from Fiber context
func GetRoles(c *fiber.Ctx) []string {
	if roles := c.Locals("userRoles"); roles != nil {
		return roles.([]string)
	}
	return nil
}

// GetPermissions retrieves the permissions from Fiber context
func GetPermissions(c *fiber.Ctx) []string {
	if perms := c.Locals("userPermissions"); perms != nil {
		return perms.([]string)
	}
	return nil
}
