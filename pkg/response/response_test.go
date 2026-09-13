package response

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gofiber/fiber/v2"
)

func TestResponseHelpers(t *testing.T) {
	app := fiber.New()

	app.Get("/test/success", func(c *fiber.Ctx) error {
		return Success(c, map[string]string{"key": "value"})
	})

	app.Get("/test/success-message", func(c *fiber.Ctx) error {
		return SuccessWithMessage(c, "operation completed", map[string]int{"count": 42})
	})

	app.Get("/test/created", func(c *fiber.Ctx) error {
		return Created(c, map[string]string{"id": "entity-123"})
	})

	app.Get("/test/bad-request", func(c *fiber.Ctx) error {
		return BadRequest(c, "invalid payload")
	})

	app.Get("/test/unauthorized", func(c *fiber.Ctx) error {
		return Unauthorized(c, "token expired")
	})

	app.Get("/test/forbidden", func(c *fiber.Ctx) error {
		return Forbidden(c, "access denied")
	})

	app.Get("/test/not-found", func(c *fiber.Ctx) error {
		return NotFound(c, "user not found")
	})

	app.Get("/test/internal-error", func(c *fiber.Ctx) error {
		return InternalError(c, "unexpected database error")
	})

	app.Get("/test/validation-error", func(c *fiber.Ctx) error {
		errs := []FieldError{
			{Field: "email", Message: "must be a valid email"},
			{Field: "password", Message: "too short"},
		}
		return ValidationError(c, errs)
	})

	tests := []struct {
		name           string
		path           string
		expectedStatus int
		validate       func(t *testing.T, r Response)
	}{
		{
			name:           "Success",
			path:           "/test/success",
			expectedStatus: http.StatusOK,
			validate: func(t *testing.T, r Response) {
				if !r.Success {
					t.Errorf("expected success to be true, got %v", r.Success)
				}
				if r.Timestamp == "" {
					t.Error("expected non-empty timestamp")
				}
			},
		},
		{
			name:           "SuccessWithMessage",
			path:           "/test/success-message",
			expectedStatus: http.StatusOK,
			validate: func(t *testing.T, r Response) {
				if !r.Success {
					t.Errorf("expected success to be true, got %v", r.Success)
				}
				if r.Message != "operation completed" {
					t.Errorf("expected message 'operation completed', got %s", r.Message)
				}
			},
		},
		{
			name:           "Created",
			path:           "/test/created",
			expectedStatus: http.StatusCreated,
			validate: func(t *testing.T, r Response) {
				if !r.Success {
					t.Errorf("expected success to be true, got %v", r.Success)
				}
			},
		},
		{
			name:           "BadRequest",
			path:           "/test/bad-request",
			expectedStatus: http.StatusBadRequest,
			validate: func(t *testing.T, r Response) {
				if r.Success {
					t.Error("expected success to be false")
				}
				if r.Code != "BAD_REQUEST" {
					t.Errorf("expected code 'BAD_REQUEST', got %s", r.Code)
				}
				if r.Message != "invalid payload" {
					t.Errorf("expected message 'invalid payload', got %s", r.Message)
				}
			},
		},
		{
			name:           "Unauthorized",
			path:           "/test/unauthorized",
			expectedStatus: http.StatusUnauthorized,
			validate: func(t *testing.T, r Response) {
				if r.Success {
					t.Error("expected success to be false")
				}
				if r.Code != "UNAUTHORIZED" {
					t.Errorf("expected code 'UNAUTHORIZED', got %s", r.Code)
				}
			},
		},
		{
			name:           "Forbidden",
			path:           "/test/forbidden",
			expectedStatus: http.StatusForbidden,
			validate: func(t *testing.T, r Response) {
				if r.Success {
					t.Error("expected success to be false")
				}
				if r.Code != "FORBIDDEN" {
					t.Errorf("expected code 'FORBIDDEN', got %s", r.Code)
				}
			},
		},
		{
			name:           "NotFound",
			path:           "/test/not-found",
			expectedStatus: http.StatusNotFound,
			validate: func(t *testing.T, r Response) {
				if r.Success {
					t.Error("expected success to be false")
				}
				if r.Code != "NOT_FOUND" {
					t.Errorf("expected code 'NOT_FOUND', got %s", r.Code)
				}
			},
		},
		{
			name:           "InternalServerError",
			path:           "/test/internal-error",
			expectedStatus: http.StatusInternalServerError,
			validate: func(t *testing.T, r Response) {
				if r.Success {
					t.Error("expected success to be false")
				}
				if r.Code != "INTERNAL_ERROR" {
					t.Errorf("expected code 'INTERNAL_ERROR', got %s", r.Code)
				}
			},
		},
		{
			name:           "ValidationError",
			path:           "/test/validation-error",
			expectedStatus: http.StatusBadRequest,
			validate: func(t *testing.T, r Response) {
				if r.Success {
					t.Error("expected success to be false")
				}
				if r.Code != "VALIDATION_ERROR" {
					t.Errorf("expected code 'VALIDATION_ERROR', got %s", r.Code)
				}
				if r.Errors == nil {
					t.Error("expected errors to be present")
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, tt.path, nil)
			resp, err := app.Test(req)
			if err != nil {
				t.Fatalf("unexpected error making test request: %v", err)
			}
			if resp.StatusCode != tt.expectedStatus {
				t.Fatalf("expected status %d, got %d", tt.expectedStatus, resp.StatusCode)
			}

			bodyBytes, err := io.ReadAll(resp.Body)
			if err != nil {
				t.Fatalf("unexpected error reading response body: %v", err)
			}

			var parsedResp Response
			if err := json.Unmarshal(bodyBytes, &parsedResp); err != nil {
				t.Fatalf("unexpected error unmarshaling json response: %v", err)
			}

			tt.validate(t, parsedResp)
		})
	}
}
