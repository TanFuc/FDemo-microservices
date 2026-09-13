package authclient

import (
	"testing"
	"time"
)

func TestDefaultConfig(t *testing.T) {
	cfg := DefaultConfig()
	if cfg.GRPCAddr != "localhost:50051" {
		t.Errorf("expected GRPCAddr 'localhost:50051', got '%s'", cfg.GRPCAddr)
	}
	if cfg.Timeout != 3*time.Second {
		t.Errorf("expected Timeout 3s, got %v", cfg.Timeout)
	}
}

func TestClient_IsConnectedAndClose(t *testing.T) {
	client := &Client{
		conn:    nil,
		addr:    "localhost:50051",
		timeout: 2 * time.Second,
	}

	if client.IsConnected() {
		t.Error("expected IsConnected to be false for nil conn")
	}

	if client.Address() != "localhost:50051" {
		t.Errorf("expected address 'localhost:50051', got '%s'", client.Address())
	}

	if err := client.Close(); err != nil {
		t.Errorf("expected nil error on closing nil conn, got %v", err)
	}
}

func TestUserInfo_Struct(t *testing.T) {
	user := UserInfo{
		UserID:      "u-123",
		Email:       "test@nexus.com",
		Roles:       []string{"admin", "seller"},
		Permissions: []string{"product:create", "product:delete"},
	}

	if user.UserID != "u-123" || user.Email != "test@nexus.com" {
		t.Errorf("unexpected UserInfo fields: %+v", user)
	}
	if len(user.Roles) != 2 || len(user.Permissions) != 2 {
		t.Errorf("unexpected roles/permissions count: %+v", user)
	}
}
