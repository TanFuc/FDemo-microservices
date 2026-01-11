package domain

import (
	"time"

	"github.com/google/uuid"
)

// UserEvent represents a user behavior event for analytics tracking.
type UserEvent struct {
	EventID   uuid.UUID `json:"event_id"`
	UserID    string    `json:"user_id"`
	EventType string    `json:"event_type"`
	Metadata  string    `json:"metadata"`
	URL       string    `json:"url"`
	IPAddress string    `json:"ip_address"`
	UserAgent string    `json:"user_agent"`
	CreatedAt time.Time `json:"created_at"`
}

// EventInput represents the incoming event data from clients.
type EventInput struct {
	UserID    string `json:"user_id"`
	EventType string `json:"event_type"`
	Metadata  string `json:"metadata"`
	URL       string `json:"url"`
}

// NewUserEvent creates a new UserEvent with server-side fields populated.
func NewUserEvent(input EventInput, ipAddress, userAgent string) UserEvent {
	return UserEvent{
		EventID:   uuid.New(),
		UserID:    input.UserID,
		EventType: input.EventType,
		Metadata:  input.Metadata,
		URL:       input.URL,
		IPAddress: ipAddress,
		UserAgent: userAgent,
		CreatedAt: time.Now().UTC(),
	}
}
