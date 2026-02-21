package authclient

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
)

// GinMiddleware provides authentication and authorization middleware for Gin
type GinMiddleware struct {
	client *Client
}

// NewGinMiddleware creates a new Gin middleware instance
func NewGinMiddleware(client *Client) *GinMiddleware {
	return &GinMiddleware{client: client}
}

// RequireAuth validates JWT token and stores user info in context
func (m *GinMiddleware) RequireAuth() gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"success": false,
				"code":    "UNAUTHORIZED",
				"message": "Missing authorization header",
			})
			return
		}

		token := strings.TrimPrefix(authHeader, "Bearer ")
		if token == authHeader {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"success": false,
				"code":    "UNAUTHORIZED",
				"message": "Invalid authorization header format",
			})
			return
		}

		resp, err := m.client.ValidateToken(c.Request.Context(), token)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{
				"success": false,
				"code":    "INTERNAL_ERROR",
				"message": "Failed to validate token",
			})
			return
		}

		if !resp.Valid {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"success": false,
				"code":    "UNAUTHORIZED",
				"message": "Invalid or expired token",
			})
			return
		}

		c.Set("userId", resp.UserId)
		c.Set("user_id", resp.UserId)
		c.Set("email", resp.Email)
		c.Set("token", token)

		c.Next()
	}
}

// RequirePermission checks if user has specific permission
func (m *GinMiddleware) RequirePermission(resource, action string) gin.HandlerFunc {
	return func(c *gin.Context) {
		token, exists := c.Get("token")
		if !exists {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"success": false,
				"code":    "UNAUTHORIZED",
				"message": "Authentication required",
			})
			return
		}

		resp, err := m.client.CheckPermission(c.Request.Context(), token.(string), resource, action, nil)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{
				"success": false,
				"code":    "INTERNAL_ERROR",
				"message": "Failed to check permission",
			})
			return
		}

		if !resp.Allowed {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{
				"success": false,
				"code":    "FORBIDDEN",
				"message": "Access denied",
				"reason":  resp.Reason,
			})
			return
		}

		c.Set("userRoles", resp.Roles)
		c.Set("userPermissions", resp.Permissions)

		c.Next()
	}
}

// RequireRole checks if user has specific role
func (m *GinMiddleware) RequireRole(role string) gin.HandlerFunc {
	return func(c *gin.Context) {
		token, exists := c.Get("token")
		if !exists {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"success": false,
				"code":    "UNAUTHORIZED",
				"message": "Authentication required",
			})
			return
		}

		resp, err := m.client.CheckPermission(c.Request.Context(), token.(string), "", "", nil)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{
				"success": false,
				"code":    "INTERNAL_ERROR",
				"message": "Failed to check role",
			})
			return
		}

		hasRole := false
		for _, r := range resp.Roles {
			if r == role {
				hasRole = true
				break
			}
		}

		if !hasRole {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{
				"success": false,
				"code":    "FORBIDDEN",
				"message": "Required role not found: " + role,
			})
			return
		}

		c.Set("userRoles", resp.Roles)
		c.Set("userPermissions", resp.Permissions)

		c.Next()
	}
}

// OptionalAuth validates JWT token if present but doesn't require it
func (m *GinMiddleware) OptionalAuth() gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			c.Next()
			return
		}

		token := strings.TrimPrefix(authHeader, "Bearer ")
		if token == authHeader {
			c.Next()
			return
		}

		resp, err := m.client.ValidateToken(c.Request.Context(), token)
		if err == nil && resp.Valid {
			c.Set("userId", resp.UserId)
			c.Set("user_id", resp.UserId)
			c.Set("email", resp.Email)
			c.Set("token", token)
		}

		c.Next()
	}
}

// GinGetUserID retrieves the user ID from Gin context
func GinGetUserID(c *gin.Context) string {
	if userID, exists := c.Get("userId"); exists {
		return userID.(string)
	}
	return ""
}

// GinGetEmail retrieves the email from Gin context
func GinGetEmail(c *gin.Context) string {
	if email, exists := c.Get("email"); exists {
		return email.(string)
	}
	return ""
}

// GinGetRoles retrieves the roles from Gin context
func GinGetRoles(c *gin.Context) []string {
	if roles, exists := c.Get("userRoles"); exists {
		return roles.([]string)
	}
	return nil
}
