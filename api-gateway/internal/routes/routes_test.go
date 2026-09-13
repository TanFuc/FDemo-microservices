package routes

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"microservices/api-gateway/internal/config"

	"github.com/gofiber/fiber/v2"
)

func TestRouter_HealthCheck(t *testing.T) {
	app := fiber.New()
	cfg := &config.Config{}
	router := NewRouter(app, cfg, nil)
	router.setupHealthCheck()

	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("unexpected error making test request: %v", err)
	}
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected status 200, got %d", resp.StatusCode)
	}
}

func TestRouter_RealtimeUnavailableWithoutURL(t *testing.T) {
	app := fiber.New()
	cfg := &config.Config{
		Services: config.ServicesConfig{
			RealtimeURL: "",
		},
	}
	router := NewRouter(app, cfg, nil)
	router.setupRealtimeRoutes()

	req := httptest.NewRequest(http.MethodGet, "/api/v1/ws", nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("unexpected error making test request: %v", err)
	}
	if resp.StatusCode != http.StatusServiceUnavailable {
		t.Fatalf("expected status 503 for unconfigured realtime URL, got %d", resp.StatusCode)
	}
}
