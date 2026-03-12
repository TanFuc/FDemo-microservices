package middleware

import (
	"strings"

	"api-gateway/internal/config"

	"github.com/gofiber/fiber/v2"
	"github.com/golang-jwt/jwt/v5"
)

type JWTClaims struct {
	UserID string `json:"user_id"`
	Email  string `json:"email"`
	Role   string `json:"role"`
	ShopID string `json:"shop_id,omitempty"`
	jwt.RegisteredClaims
}

func JWTAuth(cfg *config.JWTConfig) fiber.Handler {
	return func(c *fiber.Ctx) error {
		authHeader := c.Get("Authorization")
		if authHeader == "" {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"error":   "unauthorized",
				"message": "missing authorization header",
			})
		}

		parts := strings.Split(authHeader, " ")
		if len(parts) != 2 || strings.ToLower(parts[0]) != "bearer" {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"error":   "unauthorized",
				"message": "invalid authorization header format",
			})
		}

		tokenString := parts[1]

		token, err := jwt.ParseWithClaims(tokenString, &JWTClaims{}, func(token *jwt.Token) (interface{}, error) {
			if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, fiber.NewError(fiber.StatusUnauthorized, "unexpected signing method")
			}
			return []byte(cfg.Secret), nil
		})

		if err != nil {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"error":   "unauthorized",
				"message": "invalid or expired token",
			})
		}

		claims, ok := token.Claims.(*JWTClaims)
		if !ok || !token.Valid {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"error":   "unauthorized",
				"message": "invalid token claims",
			})
		}

		// Also try to get user_id from Subject claim (standard JWT)
		userID := claims.UserID
		if userID == "" && claims.Subject != "" {
			userID = claims.Subject
		}

		c.Locals("user_id", userID)
		c.Locals("email", claims.Email)
		c.Locals("role", claims.Role)
		if claims.ShopID != "" {
			c.Locals("shop_id", claims.ShopID)
		}

		c.Request().Header.Set("X-User-ID", userID)
		c.Request().Header.Set("X-User-Email", claims.Email)
		c.Request().Header.Set("X-User-Role", claims.Role)
		if claims.ShopID != "" {
			c.Request().Header.Set("X-Shop-ID", claims.ShopID)
		}

		return c.Next()
	}
}

func OptionalJWTAuth(cfg *config.JWTConfig) fiber.Handler {
	return func(c *fiber.Ctx) error {
		authHeader := c.Get("Authorization")
		if authHeader == "" {
			return c.Next()
		}

		parts := strings.Split(authHeader, " ")
		if len(parts) != 2 || strings.ToLower(parts[0]) != "bearer" {
			return c.Next()
		}

		tokenString := parts[1]

		token, err := jwt.ParseWithClaims(tokenString, &JWTClaims{}, func(token *jwt.Token) (interface{}, error) {
			if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, fiber.NewError(fiber.StatusUnauthorized, "unexpected signing method")
			}
			return []byte(cfg.Secret), nil
		})

		if err != nil || !token.Valid {
			return c.Next()
		}

		claims, ok := token.Claims.(*JWTClaims)
		if ok && token.Valid {
			// Also try to get user_id from Subject claim (standard JWT)
			userID := claims.UserID
			if userID == "" && claims.Subject != "" {
				userID = claims.Subject
			}

			c.Locals("user_id", userID)
			c.Locals("email", claims.Email)
			c.Locals("role", claims.Role)
			if claims.ShopID != "" {
				c.Locals("shop_id", claims.ShopID)
			}

			c.Request().Header.Set("X-User-ID", userID)
			c.Request().Header.Set("X-User-Email", claims.Email)
			c.Request().Header.Set("X-User-Role", claims.Role)
			if claims.ShopID != "" {
				c.Request().Header.Set("X-Shop-ID", claims.ShopID)
			}
		}

		return c.Next()
	}
}
