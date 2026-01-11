package middleware

import (
	"github.com/gofiber/fiber/v2"

	"microservices/auth/internal/domain/service"
	"microservices/auth/pkg/errors"
	"microservices/auth/pkg/response"
)

type PermissionMiddleware struct {
	authService *service.AuthService
}

func NewPermissionMiddleware(authService *service.AuthService) *PermissionMiddleware {
	return &PermissionMiddleware{
		authService: authService,
	}
}

func (m *PermissionMiddleware) RequirePermissions(permissions ...string) fiber.Handler {
	return func(c *fiber.Ctx) error {
		user := GetAuthenticatedUser(c)
		if user == nil {
			return response.Error(c, errors.New("UNAUTHORIZED", "Authentication required", fiber.StatusUnauthorized))
		}

		hasPermission, err := m.authService.CheckAnyPermission(c.Context(), user.ID, permissions)
		if err != nil {
			return response.Error(c, err)
		}

		if !hasPermission {
			return response.Error(c, errors.ErrInsufficientPermissions)
		}

		return c.Next()
	}
}

func (m *PermissionMiddleware) RequireAllPermissions(permissions ...string) fiber.Handler {
	return func(c *fiber.Ctx) error {
		user := GetAuthenticatedUser(c)
		if user == nil {
			return response.Error(c, errors.New("UNAUTHORIZED", "Authentication required", fiber.StatusUnauthorized))
		}

		for _, permission := range permissions {
			hasPermission, err := m.authService.CheckPermission(c.Context(), user.ID, permission)
			if err != nil {
				return response.Error(c, err)
			}
			if !hasPermission {
				return response.Error(c, errors.ErrInsufficientPermissions)
			}
		}

		return c.Next()
	}
}
