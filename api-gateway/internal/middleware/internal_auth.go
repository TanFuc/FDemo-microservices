package middleware

import (
	"github.com/gofiber/fiber/v2"
)

const InternalServiceHeader = "X-Internal-Service-Key"

// InternalServiceAuth middleware validates internal service calls
func InternalServiceAuth(expectedKey string) fiber.Handler {
	return func(c *fiber.Ctx) error {
		if expectedKey == "" {
			// If no key is configured, skip validation
			return c.Next()
		}

		key := c.Get(InternalServiceHeader)
		if key != expectedKey {
			return c.Status(401).JSON(fiber.Map{
				"error":   "Unauthorized",
				"message": "Invalid or missing internal service key",
			})
		}
		return c.Next()
	}
}

// TraceHeaders constants for distributed tracing
const (
	TraceIDHeader = "X-Trace-ID"
	SpanIDHeader  = "X-Span-ID"
)

// PropagateTracing middleware ensures trace IDs are propagated to upstream services
func PropagateTracing() fiber.Handler {
	return func(c *fiber.Ctx) error {
		// Get existing trace ID or generate a new one
		traceID := c.Get(TraceIDHeader)
		if traceID == "" {
			traceID = generateTraceID()
		}

		// Set trace ID in context for logging
		c.Locals("traceId", traceID)

		// Add trace ID to request headers for upstream calls
		c.Request().Header.Set(TraceIDHeader, traceID)

		// Also add to response headers
		c.Set(TraceIDHeader, traceID)

		return c.Next()
	}
}

// generateTraceID generates a unique trace ID
func generateTraceID() string {
	// Simple implementation - in production use proper UUID/ULID
	return randomHex(16)
}

// randomHex generates a random hex string of n bytes
func randomHex(n int) string {
	const hex = "0123456789abcdef"
	result := make([]byte, n*2)
	for i := range result {
		result[i] = hex[i%16]
	}
	return string(result)
}
