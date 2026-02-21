package response

import (
	"time"

	"github.com/gofiber/fiber/v2"
	"microservices/profile/pkg/errors"
)

type Response struct {
	Success   bool        `json:"success"`
	Data      interface{} `json:"data,omitempty"`
	Message   string      `json:"message,omitempty"`
	Timestamp string      `json:"timestamp"`
}

type ErrorResponse struct {
	Success   bool     `json:"success"`
	Code      string   `json:"code"`
	Message   string   `json:"message"`
	Errors    []string `json:"errors,omitempty"`
	Timestamp string   `json:"timestamp"`
}

func Success(c *fiber.Ctx, data interface{}) error {
	return c.Status(fiber.StatusOK).JSON(Response{
		Success:   true,
		Data:      data,
		Timestamp: time.Now().UTC().Format(time.RFC3339),
	})
}

func SuccessWithMessage(c *fiber.Ctx, data interface{}, message string) error {
	return c.Status(fiber.StatusOK).JSON(Response{
		Success:   true,
		Data:      data,
		Message:   message,
		Timestamp: time.Now().UTC().Format(time.RFC3339),
	})
}

func Created(c *fiber.Ctx, data interface{}) error {
	return c.Status(fiber.StatusCreated).JSON(Response{
		Success:   true,
		Data:      data,
		Timestamp: time.Now().UTC().Format(time.RFC3339),
	})
}

func Error(c *fiber.Ctx, err error) error {
	appErr := errors.GetAppError(err)
	return c.Status(appErr.StatusCode).JSON(ErrorResponse{
		Success:   false,
		Code:      appErr.Code,
		Message:   appErr.Message,
		Timestamp: time.Now().UTC().Format(time.RFC3339),
	})
}

func ValidationError(c *fiber.Ctx, validationErrors []string) error {
	return c.Status(fiber.StatusBadRequest).JSON(ErrorResponse{
		Success:   false,
		Code:      errors.CodeValidation,
		Message:   "Validation failed",
		Errors:    validationErrors,
		Timestamp: time.Now().UTC().Format(time.RFC3339),
	})
}

func BadRequest(c *fiber.Ctx, message string) error {
	return c.Status(fiber.StatusBadRequest).JSON(ErrorResponse{
		Success:   false,
		Code:      errors.CodeBadRequest,
		Message:   message,
		Timestamp: time.Now().UTC().Format(time.RFC3339),
	})
}

func Unauthorized(c *fiber.Ctx, message string) error {
	return c.Status(fiber.StatusUnauthorized).JSON(ErrorResponse{
		Success:   false,
		Code:      errors.CodeUnauthorized,
		Message:   message,
		Timestamp: time.Now().UTC().Format(time.RFC3339),
	})
}

func NotFound(c *fiber.Ctx, message string) error {
	return c.Status(fiber.StatusNotFound).JSON(ErrorResponse{
		Success:   false,
		Code:      errors.CodeNotFound,
		Message:   message,
		Timestamp: time.Now().UTC().Format(time.RFC3339),
	})
}
