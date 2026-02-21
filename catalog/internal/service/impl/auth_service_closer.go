package impl

import (
	"microservices/catalog/internal/service"
)

// AuthServiceCloser wraps AuthService with Close capability
type AuthServiceCloser struct {
	AuthService service.AuthService
	closer      interface{ Close() error }
}

// AuthService interface for type assertion
type AuthService = service.AuthService
