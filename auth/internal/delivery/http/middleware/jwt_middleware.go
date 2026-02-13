package middleware

import (
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"

	"microservices/auth/internal/domain/service"
	"microservices/auth/pkg/errors"
	"microservices/auth/pkg/response"
)

const (
	AuthorizationHeader = "Authorization"
	BearerPrefix        = "Bearer "
	UserContextKey      = "user"
	TokenClaimsKey      = "tokenClaims"
)

type AuthenticatedUser struct {
	ID       uuid.UUID
	Email    string
	JTI      string
	IssuedAt time.Time
	ExpireAt time.Time
}

type JWTMiddleware struct {
	tokenService *service.TokenService
	authService  *service.AuthService
}

func NewJWTMiddleware(tokenService *service.TokenService, authService *service.AuthService) *JWTMiddleware {
	return &JWTMiddleware{
		tokenService: tokenService,
		authService:  authService,
	}
}

func (m *JWTMiddleware) Authenticate() fiber.Handler {
	return func(c *fiber.Ctx) error {
		var tokenString string

		// Priority 1: Authorization header
		authHeader := c.Get(AuthorizationHeader)
		if authHeader != "" {
			if !strings.HasPrefix(authHeader, BearerPrefix) {
				return response.Error(c, errors.New("INVALID_TOKEN_FORMAT", "Invalid authorization header format", fiber.StatusUnauthorized))
			}
			tokenString = strings.TrimPrefix(authHeader, BearerPrefix)
		}

		// Priority 2: access_token cookie fallback
		if tokenString == "" {
			tokenString = c.Cookies("access_token")
		}

		if tokenString == "" {
			return response.Error(c, errors.New("MISSING_TOKEN", "Authentication token is required", fiber.StatusUnauthorized))
		}

		// Verify token
		claims, err := m.tokenService.VerifyAccessToken(c.Context(), tokenString)
		if err != nil {
			return response.Error(c, err)
		}

		// Parse user ID
		userID, err := uuid.Parse(claims.Subject)
		if err != nil {
			return response.Error(c, errors.ErrTokenInvalid)
		}

		// Validate user exists and is active
		user, err := m.authService.ValidateUser(c.Context(), userID)
		if err != nil {
			return response.Error(c, err)
		}

		if !user.IsActive() {
			return response.Error(c, errors.ErrUserInactive)
		}

		// Set user in context
		authenticatedUser := &AuthenticatedUser{
			ID:       userID,
			Email:    claims.Email,
			JTI:      claims.ID,
			IssuedAt: claims.IssuedAt.Time,
			ExpireAt: claims.ExpiresAt.Time,
		}

		c.Locals(UserContextKey, authenticatedUser)
		c.Locals(TokenClaimsKey, claims)

		return c.Next()
	}
}

func GetAuthenticatedUser(c *fiber.Ctx) *AuthenticatedUser {
	user, ok := c.Locals(UserContextKey).(*AuthenticatedUser)
	if !ok {
		return nil
	}
	return user
}

func GetTokenClaims(c *fiber.Ctx) *service.AccessTokenClaims {
	claims, ok := c.Locals(TokenClaimsKey).(*service.AccessTokenClaims)
	if !ok {
		return nil
	}
	return claims
}
