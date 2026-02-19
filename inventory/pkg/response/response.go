package response

import (
	"github.com/gofiber/fiber/v2"
)

type Response struct {
	Success bool        `json:"success"`
	Message string      `json:"message,omitempty"`
	Data    interface{} `json:"data,omitempty"`
	Error   string      `json:"error,omitempty"`
}

type PaginatedResponse struct {
	Success bool        `json:"success"`
	Data    interface{} `json:"data"`
	Meta    Meta        `json:"meta"`
}

type Meta struct {
	Total int64 `json:"total"`
	Page  int   `json:"page"`
	Limit int   `json:"limit"`
}

func Success(c *fiber.Ctx, data interface{}) error {
	return c.Status(fiber.StatusOK).JSON(Response{
		Success: true,
		Data:    data,
	})
}

func SuccessWithMessage(c *fiber.Ctx, message string, data interface{}) error {
	return c.Status(fiber.StatusOK).JSON(Response{
		Success: true,
		Message: message,
		Data:    data,
	})
}

func Created(c *fiber.Ctx, data interface{}) error {
	return c.Status(fiber.StatusCreated).JSON(Response{
		Success: true,
		Data:    data,
	})
}

func NoContent(c *fiber.Ctx) error {
	return c.SendStatus(fiber.StatusNoContent)
}

func Paginated(c *fiber.Ctx, data interface{}, total int64, page, limit int) error {
	return c.Status(fiber.StatusOK).JSON(PaginatedResponse{
		Success: true,
		Data:    data,
		Meta: Meta{
			Total: total,
			Page:  page,
			Limit: limit,
		},
	})
}

func BadRequest(c *fiber.Ctx, message string) error {
	return c.Status(fiber.StatusBadRequest).JSON(Response{
		Success: false,
		Error:   message,
	})
}

func Unauthorized(c *fiber.Ctx, message string) error {
	if message == "" {
		message = "unauthorized"
	}
	return c.Status(fiber.StatusUnauthorized).JSON(Response{
		Success: false,
		Error:   message,
	})
}

func Forbidden(c *fiber.Ctx, message string) error {
	if message == "" {
		message = "forbidden"
	}
	return c.Status(fiber.StatusForbidden).JSON(Response{
		Success: false,
		Error:   message,
	})
}

func NotFound(c *fiber.Ctx, message string) error {
	if message == "" {
		message = "resource not found"
	}
	return c.Status(fiber.StatusNotFound).JSON(Response{
		Success: false,
		Error:   message,
	})
}

func Conflict(c *fiber.Ctx, message string) error {
	return c.Status(fiber.StatusConflict).JSON(Response{
		Success: false,
		Error:   message,
	})
}

func InternalError(c *fiber.Ctx, err error) error {
	return c.Status(fiber.StatusInternalServerError).JSON(Response{
		Success: false,
		Error:   "internal server error",
	})
}
