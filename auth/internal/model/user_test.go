package model

import (
	"testing"
	"time"

	"github.com/google/uuid"
)

func TestUser_IsActive(t *testing.T) {
	activeUser := &User{Status: UserStatusActive}
	if !activeUser.IsActive() {
		t.Error("expected activeUser.IsActive() to be true")
	}

	inactiveUser := &User{Status: UserStatusInactive}
	if inactiveUser.IsActive() {
		t.Error("expected inactiveUser.IsActive() to be false")
	}

	bannedUser := &User{Status: UserStatusBanned}
	if bannedUser.IsActive() {
		t.Error("expected bannedUser.IsActive() to be false")
	}
}

func TestUser_IsLocked(t *testing.T) {
	// Not locked (nil)
	user := &User{LockedUntil: nil}
	if user.IsLocked() {
		t.Error("expected user without LockedUntil to not be locked")
	}

	// Locked in future
	future := time.Now().Add(10 * time.Minute)
	user.LockedUntil = &future
	if !user.IsLocked() {
		t.Error("expected user with future LockedUntil to be locked")
	}

	// Lock expired in past
	past := time.Now().Add(-10 * time.Minute)
	user.LockedUntil = &past
	if user.IsLocked() {
		t.Error("expected user with past LockedUntil to not be locked")
	}
}

func TestUser_GetRoleNames(t *testing.T) {
	user := &User{
		UserRoles: []UserRole{
			{Role: Role{Name: "ADMIN"}},
			{Role: Role{Name: "SELLER"}},
		},
	}

	roles := user.GetRoleNames()
	if len(roles) != 2 {
		t.Fatalf("expected 2 roles, got %d", len(roles))
	}
	if roles[0] != "ADMIN" || roles[1] != "SELLER" {
		t.Errorf("unexpected roles: %v", roles)
	}
}

func TestUser_BeforeCreate(t *testing.T) {
	user := &User{
		Email: "test@nexus.com",
	}

	if err := user.BeforeCreate(nil); err != nil {
		t.Fatalf("unexpected error in BeforeCreate: %v", err)
	}

	if user.ID == uuid.Nil {
		t.Error("expected ID to be generated in BeforeCreate")
	}
	if user.ReferralCode == "" {
		t.Error("expected ReferralCode to be generated in BeforeCreate")
	}
}
