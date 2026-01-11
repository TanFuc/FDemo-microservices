package middleware

import (
	"crypto/sha256"
	"encoding/hex"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"

	"microservices/auth/pkg/logger"
)

const (
	RequestIDHeader = "X-Request-ID"
	DeviceIDHeader  = "X-Device-ID"
)

func RequestID() fiber.Handler {
	return func(c *fiber.Ctx) error {
		requestID := c.Get(RequestIDHeader)
		if requestID == "" {
			requestID = uuid.New().String()
		}
		c.Set(RequestIDHeader, requestID)
		c.Locals("requestId", requestID)
		return c.Next()
	}
}

func RequestLogger() fiber.Handler {
	return func(c *fiber.Ctx) error {
		start := time.Now()

		err := c.Next()

		duration := time.Since(start)
		requestID := c.Locals("requestId")

		logEvent := logger.Info()
		if c.Response().StatusCode() >= 400 {
			logEvent = logger.Warn()
		}
		if c.Response().StatusCode() >= 500 {
			logEvent = logger.Error()
		}

		logEvent.
			Str("requestId", requestID.(string)).
			Str("method", c.Method()).
			Str("path", c.Path()).
			Int("status", c.Response().StatusCode()).
			Dur("duration", duration).
			Str("ip", c.IP()).
			Msg("HTTP Request")

		return err
	}
}

func Recovery() fiber.Handler {
	return func(c *fiber.Ctx) error {
		defer func() {
			if r := recover(); r != nil {
				requestID := c.Locals("requestId")
				logger.Error().
					Interface("panic", r).
					Interface("requestId", requestID).
					Str("path", c.Path()).
					Msg("Panic recovered")

				c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
					"success":   false,
					"code":      "INTERNAL_ERROR",
					"message":   "An internal error occurred",
					"timestamp": time.Now().UTC().Format(time.RFC3339),
					"path":      c.Path(),
				})
			}
		}()
		return c.Next()
	}
}

func GetClientIP(c *fiber.Ctx) string {
	// Check X-Forwarded-For header first
	xff := c.Get("X-Forwarded-For")
	if xff != "" {
		return xff
	}

	// Check X-Real-IP header
	xri := c.Get("X-Real-IP")
	if xri != "" {
		return xri
	}

	return c.IP()
}

func GetDeviceID(c *fiber.Ctx) string {
	// First check header
	deviceID := c.Get(DeviceIDHeader)
	if deviceID != "" {
		return deviceID
	}

	// Generate from user agent and IP
	userAgent := c.Get("User-Agent")
	ip := GetClientIP(c)

	hash := sha256.Sum256([]byte(userAgent + ip))
	return hex.EncodeToString(hash[:])[:16]
}

func GetUserAgent(c *fiber.Ctx) string {
	return c.Get("User-Agent")
}
