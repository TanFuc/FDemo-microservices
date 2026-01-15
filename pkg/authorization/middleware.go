package authorization

import (
	"fmt"
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"
)

// MiddlewareConfig represents the configuration for authorization middleware
type MiddlewareConfig struct {
	// Authorizer is the authorization instance
	Authorizer Authorizer

	// SubjectExtractor extracts the subject (user ID) from the request
	// If nil, uses the default extractor which looks for "user_id" in locals
	SubjectExtractor func(c *fiber.Ctx) string

	// ResourceExtractor extracts the resource from the request
	// If nil, uses the request path
	ResourceExtractor func(c *fiber.Ctx) string

	// ActionExtractor extracts the action from the request
	// If nil, maps HTTP method to action
	ActionExtractor func(c *fiber.Ctx) string

	// DomainExtractor extracts the domain from the request
	// If nil, uses "default" domain
	DomainExtractor func(c *fiber.Ctx) string

	// AttributesExtractor extracts attributes for ABAC from the request
	AttributesExtractor func(c *fiber.Ctx) map[string]interface{}

	// Skipper defines a function to skip middleware
	Skipper func(c *fiber.Ctx) bool

	// ErrorHandler handles authorization errors
	ErrorHandler func(c *fiber.Ctx, err error) error

	// UnauthorizedHandler handles unauthorized requests
	UnauthorizedHandler func(c *fiber.Ctx) error

	// ForbiddenHandler handles forbidden requests
	ForbiddenHandler func(c *fiber.Ctx) error

	// SuccessHandler is called after successful authorization
	SuccessHandler func(c *fiber.Ctx, result *AuthResult)

	// ContextKey is the key used to store the authorization result in context
	ContextKey string

	// EnableLogging enables logging of authorization decisions
	EnableLogging bool

	// LoggingFunc is called for each authorization decision when logging is enabled
	LoggingFunc func(c *fiber.Ctx, result *AuthResult, duration time.Duration)
}

// DefaultMiddlewareConfig returns the default middleware configuration
func DefaultMiddlewareConfig() MiddlewareConfig {
	return MiddlewareConfig{
		SubjectExtractor:  defaultSubjectExtractor,
		ResourceExtractor: defaultResourceExtractor,
		ActionExtractor:   defaultActionExtractor,
		DomainExtractor:   defaultDomainExtractor,
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
				"message": "Access denied",
			})
		},
		ContextKey:    "authz_result",
		EnableLogging: false,
	}
}

// New creates a new authorization middleware with default config
func NewMiddleware(auth Authorizer) fiber.Handler {
	config := DefaultMiddlewareConfig()
	config.Authorizer = auth
	return NewMiddlewareWithConfig(config)
}

// NewMiddlewareWithConfig creates a new authorization middleware with custom config
func NewMiddlewareWithConfig(config MiddlewareConfig) fiber.Handler {
	// Apply defaults for nil fields
	if config.SubjectExtractor == nil {
		config.SubjectExtractor = defaultSubjectExtractor
	}
	if config.ResourceExtractor == nil {
		config.ResourceExtractor = defaultResourceExtractor
	}
	if config.ActionExtractor == nil {
		config.ActionExtractor = defaultActionExtractor
	}
	if config.DomainExtractor == nil {
		config.DomainExtractor = defaultDomainExtractor
	}
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
				"message": "Access denied",
			})
		}
	}
	if config.ContextKey == "" {
		config.ContextKey = "authz_result"
	}

	return func(c *fiber.Ctx) error {
		// Skip if configured
		if config.Skipper(c) {
			return c.Next()
		}

		start := time.Now()

		// Extract subject (user ID)
		subject := config.SubjectExtractor(c)
		if subject == "" {
			return config.UnauthorizedHandler(c)
		}

		// Extract resource, action, and domain
		resource := config.ResourceExtractor(c)
		action := config.ActionExtractor(c)
		domain := config.DomainExtractor(c)

		// Build auth request
		request := &AuthRequest{
			Subject: subject,
			Object:  resource,
			Action:  action,
			Domain:  domain,
		}

		// Extract attributes if configured
		if config.AttributesExtractor != nil {
			request.Attributes = config.AttributesExtractor(c)
		}

		// Perform authorization check
		result, err := config.Authorizer.Enforce(c.Context(), request)
		if err != nil {
			return config.ErrorHandler(c, err)
		}

		// Log if enabled
		if config.EnableLogging && config.LoggingFunc != nil {
			config.LoggingFunc(c, result, time.Since(start))
		}

		// Store result in context
		c.Locals(config.ContextKey, result)

		// Check authorization result
		if !result.Allowed {
			return config.ForbiddenHandler(c)
		}

		// Call success handler if configured
		if config.SuccessHandler != nil {
			config.SuccessHandler(c, result)
		}

		return c.Next()
	}
}

// RequireRole creates middleware that requires the user to have a specific role
func RequireRole(auth Authorizer, roleID string, domain ...string) fiber.Handler {
	return func(c *fiber.Ctx) error {
		subject := defaultSubjectExtractor(c)
		if subject == "" {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"success": false,
				"code":    "UNAUTHORIZED",
				"message": "Authentication required",
			})
		}

		hasRole, err := HasRole(c.Context(), auth, subject, roleID, domain...)
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"success": false,
				"code":    "INTERNAL_ERROR",
				"message": "Failed to check role",
			})
		}

		if !hasRole {
			return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
				"success": false,
				"code":    "FORBIDDEN",
				"message": fmt.Sprintf("Role '%s' required", roleID),
			})
		}

		return c.Next()
	}
}

// RequireAnyRole creates middleware that requires the user to have any of the specified roles
func RequireAnyRole(auth Authorizer, roleIDs []string, domain ...string) fiber.Handler {
	return func(c *fiber.Ctx) error {
		subject := defaultSubjectExtractor(c)
		if subject == "" {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"success": false,
				"code":    "UNAUTHORIZED",
				"message": "Authentication required",
			})
		}

		hasRole, err := HasAnyRole(c.Context(), auth, subject, roleIDs, domain...)
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"success": false,
				"code":    "INTERNAL_ERROR",
				"message": "Failed to check roles",
			})
		}

		if !hasRole {
			return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
				"success": false,
				"code":    "FORBIDDEN",
				"message": "Required role not found",
			})
		}

		return c.Next()
	}
}

// RequireAllRoles creates middleware that requires the user to have all of the specified roles
func RequireAllRoles(auth Authorizer, roleIDs []string, domain ...string) fiber.Handler {
	return func(c *fiber.Ctx) error {
		subject := defaultSubjectExtractor(c)
		if subject == "" {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"success": false,
				"code":    "UNAUTHORIZED",
				"message": "Authentication required",
			})
		}

		hasRoles, err := HasAllRoles(c.Context(), auth, subject, roleIDs, domain...)
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"success": false,
				"code":    "INTERNAL_ERROR",
				"message": "Failed to check roles",
			})
		}

		if !hasRoles {
			return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
				"success": false,
				"code":    "FORBIDDEN",
				"message": "All required roles not found",
			})
		}

		return c.Next()
	}
}

// RequirePermission creates middleware that requires a specific permission
func RequirePermission(auth Authorizer, resource, action string) fiber.Handler {
	return func(c *fiber.Ctx) error {
		subject := defaultSubjectExtractor(c)
		if subject == "" {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"success": false,
				"code":    "UNAUTHORIZED",
				"message": "Authentication required",
			})
		}

		allowed, err := CanExecute(c.Context(), auth, subject, resource, action)
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"success": false,
				"code":    "INTERNAL_ERROR",
				"message": "Failed to check permission",
			})
		}

		if !allowed {
			return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
				"success": false,
				"code":    "FORBIDDEN",
				"message": fmt.Sprintf("Permission '%s:%s' required", resource, action),
			})
		}

		return c.Next()
	}
}

// Default extractors

func defaultSubjectExtractor(c *fiber.Ctx) string {
	// Try to get user_id from locals (set by JWT middleware)
	if userID := c.Locals("user_id"); userID != nil {
		return fmt.Sprintf("%v", userID)
	}
	// Try to get from header
	if userID := c.Get("X-User-ID"); userID != "" {
		return userID
	}
	return ""
}

func defaultResourceExtractor(c *fiber.Ctx) string {
	// Use the base path without parameters
	path := c.Path()
	// Remove leading slash and split
	parts := strings.Split(strings.TrimPrefix(path, "/"), "/")
	if len(parts) > 0 {
		return parts[0]
	}
	return path
}

func defaultActionExtractor(c *fiber.Ctx) string {
	// Map HTTP methods to CRUD actions
	switch c.Method() {
	case fiber.MethodGet:
		return "read"
	case fiber.MethodPost:
		return "create"
	case fiber.MethodPut, fiber.MethodPatch:
		return "update"
	case fiber.MethodDelete:
		return "delete"
	default:
		return strings.ToLower(c.Method())
	}
}

func defaultDomainExtractor(c *fiber.Ctx) string {
	// Try to get domain from header
	if domain := c.Get("X-Domain"); domain != "" {
		return domain
	}
	// Try to get tenant_id from locals
	if tenantID := c.Locals("tenant_id"); tenantID != nil {
		return fmt.Sprintf("%v", tenantID)
	}
	return "default"
}

func defaultSkipper(c *fiber.Ctx) bool {
	// Skip health check and metrics endpoints
	path := c.Path()
	return path == "/health" || path == "/metrics" || path == "/ready" || path == "/live"
}

func defaultErrorHandler(c *fiber.Ctx, err error) error {
	return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
		"success": false,
		"code":    "INTERNAL_ERROR",
		"message": "Authorization check failed",
		"error":   err.Error(),
	})
}

// GetAuthResult retrieves the authorization result from the fiber context
func GetAuthResult(c *fiber.Ctx) *AuthResult {
	result := c.Locals("authz_result")
	if result == nil {
		return nil
	}
	return result.(*AuthResult)
}

// GetAuthResultWithKey retrieves the authorization result using a custom key
func GetAuthResultWithKey(c *fiber.Ctx, key string) *AuthResult {
	result := c.Locals(key)
	if result == nil {
		return nil
	}
	return result.(*AuthResult)
}
