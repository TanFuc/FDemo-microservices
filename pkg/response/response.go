package response

import (
	"time"

	"github.com/gofiber/fiber/v2"
)

type Response struct {
	Success   bool        `json:"success"`
	Code      string      `json:"code,omitempty"`
	Message   string      `json:"message,omitempty"`
	Data      interface{} `json:"data,omitempty"`
	Errors    interface{} `json:"errors,omitempty"`
	Timestamp string      `json:"timestamp"`
	Path      string      `json:"path,omitempty"`
}

type FieldError struct {
	Field   string `json:"field"`
	Message string `json:"message"`
}

func Success(c *fiber.Ctx, data interface{}) error {
	return c.Status(fiber.StatusOK).JSON(Response{
		Success:   true,
		Data:      data,
		Timestamp: time.Now().UTC().Format(time.RFC3339),
		Path:      c.Path(),
	})
}

func SuccessWithMessage(c *fiber.Ctx, message string, data interface{}) error {
	return c.Status(fiber.StatusOK).JSON(Response{
		Success:   true,
		Message:   message,
		Data:      data,
		Timestamp: time.Now().UTC().Format(time.RFC3339),
		Path:      c.Path(),
	})
}

func Created(c *fiber.Ctx, data interface{}) error {
	return c.Status(fiber.StatusCreated).JSON(Response{
		Success:   true,
		Data:      data,
		Timestamp: time.Now().UTC().Format(time.RFC3339),
		Path:      c.Path(),
	})
}

func BadRequest(c *fiber.Ctx, message string) error {
	return c.Status(fiber.StatusBadRequest).JSON(Response{
		Success:   false,
		Code:      "BAD_REQUEST",
		Message:   message,
		Timestamp: time.Now().UTC().Format(time.RFC3339),
		Path:      c.Path(),
	})
}

func ValidationError(c *fiber.Ctx, errors []FieldError) error {
	return c.Status(fiber.StatusBadRequest).JSON(Response{
		Success:   false,
		Code:      "VALIDATION_ERROR",
		Message:   "Validation failed",
		Errors:    errors,
		Timestamp: time.Now().UTC().Format(time.RFC3339),
		Path:      c.Path(),
	})
}

func NotFound(c *fiber.Ctx, message string) error {
	return c.Status(fiber.StatusNotFound).JSON(Response{
		Success:   false,
		Code:      "NOT_FOUND",
		Message:   message,
		Timestamp: time.Now().UTC().Format(time.RFC3339),
		Path:      c.Path(),
	})
}

func InternalError(c *fiber.Ctx, message string) error {
	return c.Status(fiber.StatusInternalServerError).JSON(Response{
		Success:   false,
		Code:      "INTERNAL_ERROR",
		Message:   message,
		Timestamp: time.Now().UTC().Format(time.RFC3339),
		Path:      c.Path(),
	})
}

// InternalServerError is an alias for InternalError
func InternalServerError(c *fiber.Ctx, message string) error {
	return InternalError(c, message)
}

func Unauthorized(c *fiber.Ctx, message string) error {
	return c.Status(fiber.StatusUnauthorized).JSON(Response{
		Success:   false,
		Code:      "UNAUTHORIZED",
		Message:   message,
		Timestamp: time.Now().UTC().Format(time.RFC3339),
		Path:      c.Path(),
	})
}

func Forbidden(c *fiber.Ctx, message string) error {
	return c.Status(fiber.StatusForbidden).JSON(Response{
		Success:   false,
		Code:      "FORBIDDEN",
		Message:   message,
		Timestamp: time.Now().UTC().Format(time.RFC3339),
		Path:      c.Path(),
	})
}
