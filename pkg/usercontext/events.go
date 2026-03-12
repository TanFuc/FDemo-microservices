package usercontext

import (
	"time"
)

// NATS event subjects for identity and profile events
const (
	// Identity events (from Auth service)
	SubjectUserCreated       = "identity.user.created"
	SubjectUserRoleChanged   = "identity.user.role_changed"
	SubjectUserStatusChanged = "identity.user.status_changed"
	SubjectUserDeleted       = "identity.user.deleted"

	// Profile events (from Profile service)
	SubjectShopRegistered = "profile.shop.registered"
	SubjectShopApproved   = "profile.shop.approved"
	SubjectShopUpdated    = "profile.shop.updated"
	SubjectUserUpdated    = "profile.user.updated"
)

// UserCreatedEvent is published when a new user registers
type UserCreatedEvent struct {
	UserID      string    `json:"userId"`
	Email       string    `json:"email"`
	DisplayName string    `json:"displayName"`
	Role        UserRole  `json:"role"`
	Timestamp   time.Time `json:"timestamp"`
}

// UserRoleChangedEvent is published when user role changes
type UserRoleChangedEvent struct {
	UserID    string    `json:"userId"`
	OldRole   UserRole  `json:"oldRole"`
	NewRole   UserRole  `json:"newRole"`
	Timestamp time.Time `json:"timestamp"`
}

// UserStatusChangedEvent is published when user status changes (ban/unban)
type UserStatusChangedEvent struct {
	UserID    string     `json:"userId"`
	OldStatus UserStatus `json:"oldStatus"`
	NewStatus UserStatus `json:"newStatus"`
	Reason    string     `json:"reason,omitempty"`
	Timestamp time.Time  `json:"timestamp"`
}

// UserDeletedEvent is published when user is soft-deleted
type UserDeletedEvent struct {
	UserID    string    `json:"userId"`
	Reason    string    `json:"reason,omitempty"`
	Timestamp time.Time `json:"timestamp"`
}

// ShopRegisteredEvent is published when a user registers a shop
type ShopRegisteredEvent struct {
	UserID       string    `json:"userId"`
	ShopID       string    `json:"shopId"`
	ShopName     string    `json:"shopName"`
	BusinessType string    `json:"businessType"`
	Status       string    `json:"status"`
	Timestamp    time.Time `json:"timestamp"`
}

// ShopApprovedEvent is published when admin approves a shop
type ShopApprovedEvent struct {
	UserID    string    `json:"userId"`
	ShopID    string    `json:"shopId"`
	Status    string    `json:"status"`
	Timestamp time.Time `json:"timestamp"`
}

// ShopUpdatedEvent is published when shop details are updated
type ShopUpdatedEvent struct {
	UserID    string    `json:"userId"`
	ShopID    string    `json:"shopId"`
	ShopName  string    `json:"shopName,omitempty"`
	Status    string    `json:"status,omitempty"`
	Timestamp time.Time `json:"timestamp"`
}

// UserUpdatedEvent is published when user profile is updated
type UserUpdatedEvent struct {
	UserID      string    `json:"userId"`
	DisplayName string    `json:"displayName,omitempty"`
	AvatarURL   string    `json:"avatarUrl,omitempty"`
	Email       string    `json:"email,omitempty"`
	Version     int64     `json:"version"`
	Timestamp   time.Time `json:"timestamp"`
}

// EventMeta contains common metadata for all events
type EventMeta struct {
	EventID   string    `json:"eventId"`
	Source    string    `json:"source"`
	Timestamp time.Time `json:"timestamp"`
	Version   int64     `json:"version"`
}

// BaseEvent is an envelope for all events
type BaseEvent struct {
	Meta    EventMeta   `json:"meta"`
	Payload interface{} `json:"payload"`
}
