package authorization

import (
	"errors"
	"testing"
)

func TestAuthError_Formatting(t *testing.T) {
	underlying := errors.New("connection reset")
	authErr := NewAuthError("enforce", "user:123", "order:456", "read", underlying)

	if authErr.Op != "enforce" || authErr.Subject != "user:123" || authErr.Object != "order:456" || authErr.Action != "read" {
		t.Errorf("fields not populated correctly: %+v", authErr)
	}

	if !errors.Is(authErr, underlying) {
		t.Error("expected authErr to unwrap to underlying error")
	}

	errStr := authErr.Error()
	if errStr == "" {
		t.Error("expected non-empty error string")
	}

	// Test error without underlying error
	authErr2 := NewAuthError("delete", "admin", "catalog", "purge", nil)
	if authErr2.Error() == "" {
		t.Error("expected non-empty error string when underlying error is nil")
	}
}

func TestErrorHelpers(t *testing.T) {
	if !IsNotFound(ErrNotFound) {
		t.Error("expected IsNotFound(ErrNotFound) to be true")
	}
	if !IsNotFound(ErrRoleNotFound) {
		t.Error("expected IsNotFound(ErrRoleNotFound) to be true")
	}
	if !IsNotFound(ErrPermissionNotFound) {
		t.Error("expected IsNotFound(ErrPermissionNotFound) to be true")
	}
	if !IsNotFound(ErrUserNotFound) {
		t.Error("expected IsNotFound(ErrUserNotFound) to be true")
	}
	if IsNotFound(ErrUnauthorized) {
		t.Error("expected IsNotFound(ErrUnauthorized) to be false")
	}

	if !IsUnauthorized(ErrUnauthorized) {
		t.Error("expected IsUnauthorized(ErrUnauthorized) to be true")
	}
	if IsUnauthorized(ErrForbidden) {
		t.Error("expected IsUnauthorized(ErrForbidden) to be false")
	}
}
