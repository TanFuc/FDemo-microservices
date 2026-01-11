package response

import (
	"time"

	"github.com/gofiber/fiber/v2"
	"microservices/auth/pkg/errors"
)

type Response struct {
	Success   bool        `json:"success"`
	Code      string      `json:"code"`
	Message   string      `json:"message"`
	Data      interface{} `json:"data,omitempty"`
	Timestamp string      `json:"timestamp"`
	Path      string      `json:"path,omitempty"`
}

type ErrorResponse struct {
	Success   bool     `json:"success"`
	Code      string   `json:"code"`
	Message   string   `json:"message"`
	Errors    []string `json:"errors,omitempty"`
	Timestamp string   `json:"timestamp"`
	Path      string   `json:"path,omitempty"`
}

func Success(c *fiber.Ctx, data interface{}) error {
	return c.Status(fiber.StatusOK).JSON(Response{
		Success:   true,
		Code:      errors.CodeSuccess,
		Message:   "Success",
		Data:      data,
		Timestamp: time.Now().UTC().Format(time.RFC3339),
		Path:      c.Path(),
	})
}

func SuccessWithMessage(c *fiber.Ctx, data interface{}, message string) error {
	return c.Status(fiber.StatusOK).JSON(Response{
		Success:   true,
		Code:      errors.CodeSuccess,
		Message:   message,
		Data:      data,
		Timestamp: time.Now().UTC().Format(time.RFC3339),
		Path:      c.Path(),
	})
}

func Created(c *fiber.Ctx, data interface{}) error {
	return c.Status(fiber.StatusCreated).JSON(Response{
		Success:   true,
		Code:      errors.CodeCreated,
		Message:   "Created successfully",
		Data:      data,
		Timestamp: time.Now().UTC().Format(time.RFC3339),
		Path:      c.Path(),
	})
}

func CreatedWithMessage(c *fiber.Ctx, data interface{}, message string) error {
	return c.Status(fiber.StatusCreated).JSON(Response{
		Success:   true,
		Code:      errors.CodeCreated,
		Message:   message,
		Data:      data,
		Timestamp: time.Now().UTC().Format(time.RFC3339),
		Path:      c.Path(),
	})
}

func Error(c *fiber.Ctx, err error) error {
	appErr := errors.GetAppError(err)
	return c.Status(appErr.StatusCode).JSON(ErrorResponse{
		Success:   false,
		Code:      appErr.Code,
		Message:   appErr.Message,
		Timestamp: time.Now().UTC().Format(time.RFC3339),
		Path:      c.Path(),
	})
}

func ErrorWithDetails(c *fiber.Ctx, err error, details []string) error {
	appErr := errors.GetAppError(err)
	return c.Status(appErr.StatusCode).JSON(ErrorResponse{
		Success:   false,
		Code:      appErr.Code,
		Message:   appErr.Message,
		Errors:    details,
		Timestamp: time.Now().UTC().Format(time.RFC3339),
		Path:      c.Path(),
	})
}

func BadRequest(c *fiber.Ctx, message string) error {
	return c.Status(fiber.StatusBadRequest).JSON(ErrorResponse{
		Success:   false,
		Code:      errors.CodeBadRequest,
		Message:   message,
		Timestamp: time.Now().UTC().Format(time.RFC3339),
		Path:      c.Path(),
	})
}

func Unauthorized(c *fiber.Ctx, message string) error {
	return c.Status(fiber.StatusUnauthorized).JSON(ErrorResponse{
		Success:   false,
		Code:      errors.CodeUnauthorized,
		Message:   message,
		Timestamp: time.Now().UTC().Format(time.RFC3339),
		Path:      c.Path(),
	})
}

func Forbidden(c *fiber.Ctx, message string) error {
	return c.Status(fiber.StatusForbidden).JSON(ErrorResponse{
		Success:   false,
		Code:      errors.CodeForbidden,
		Message:   message,
		Timestamp: time.Now().UTC().Format(time.RFC3339),
		Path:      c.Path(),
	})
}

func NotFound(c *fiber.Ctx, message string) error {
	return c.Status(fiber.StatusNotFound).JSON(ErrorResponse{
		Success:   false,
		Code:      errors.CodeNotFound,
		Message:   message,
		Timestamp: time.Now().UTC().Format(time.RFC3339),
		Path:      c.Path(),
	})
}

func Conflict(c *fiber.Ctx, message string) error {
	return c.Status(fiber.StatusConflict).JSON(ErrorResponse{
		Success:   false,
		Code:      errors.CodeConflict,
		Message:   message,
		Timestamp: time.Now().UTC().Format(time.RFC3339),
		Path:      c.Path(),
	})
}

func InternalError(c *fiber.Ctx, message string) error {
	return c.Status(fiber.StatusInternalServerError).JSON(ErrorResponse{
		Success:   false,
		Code:      errors.CodeInternal,
		Message:   message,
		Timestamp: time.Now().UTC().Format(time.RFC3339),
		Path:      c.Path(),
	})
}

func ValidationError(c *fiber.Ctx, validationErrors []string) error {
	return c.Status(fiber.StatusBadRequest).JSON(ErrorResponse{
		Success:   false,
		Code:      errors.CodeValidation,
		Message:   "Validation failed",
		Errors:    validationErrors,
		Timestamp: time.Now().UTC().Format(time.RFC3339),
		Path:      c.Path(),
	})
}
