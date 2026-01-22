package authclient

import (
	"context"
	"net/http"
	"strings"
)

// ChiMiddleware provides authentication and authorization middleware for Chi router
type ChiMiddleware struct {
	client *Client
}

// NewChiMiddleware creates a new Chi middleware instance
func NewChiMiddleware(client *Client) *ChiMiddleware {
	return &ChiMiddleware{client: client}
}

// Context keys for storing user info
type contextKey string

const (
	UserIDKey      contextKey = "userId"
	EmailKey       contextKey = "email"
	TokenKey       contextKey = "token"
	RolesKey       contextKey = "userRoles"
	PermissionsKey contextKey = "userPermissions"
)

// RequireAuth validates JWT token and stores user info in context
func (m *ChiMiddleware) RequireAuth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		authHeader := r.Header.Get("Authorization")
		if authHeader == "" {
			writeJSONError(w, http.StatusUnauthorized, "UNAUTHORIZED", "Missing authorization header")
			return
		}

		token := strings.TrimPrefix(authHeader, "Bearer ")
		if token == authHeader {
			writeJSONError(w, http.StatusUnauthorized, "UNAUTHORIZED", "Invalid authorization header format")
			return
		}

		resp, err := m.client.ValidateToken(r.Context(), token)
		if err != nil {
			writeJSONError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to validate token")
			return
		}

		if !resp.Valid {
			writeJSONError(w, http.StatusUnauthorized, "UNAUTHORIZED", "Invalid or expired token")
			return
		}

		// Store user info in context
		ctx := r.Context()
		ctx = context.WithValue(ctx, UserIDKey, resp.UserId)
		ctx = context.WithValue(ctx, EmailKey, resp.Email)
		ctx = context.WithValue(ctx, TokenKey, token)

		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// RequirePermission checks if user has specific permission
func (m *ChiMiddleware) RequirePermission(resource, action string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			token, ok := r.Context().Value(TokenKey).(string)
			if !ok || token == "" {
				writeJSONError(w, http.StatusUnauthorized, "UNAUTHORIZED", "Authentication required")
				return
			}

			resp, err := m.client.CheckPermission(r.Context(), token, resource, action, nil)
			if err != nil {
				writeJSONError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to check permission")
				return
			}

			if !resp.Allowed {
				writeJSONError(w, http.StatusForbidden, "FORBIDDEN", "Access denied: "+resp.Reason)
				return
			}

			ctx := r.Context()
			ctx = context.WithValue(ctx, RolesKey, resp.Roles)
			ctx = context.WithValue(ctx, PermissionsKey, resp.Permissions)

			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// RequireRole checks if user has specific role
func (m *ChiMiddleware) RequireRole(role string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			token, ok := r.Context().Value(TokenKey).(string)
			if !ok || token == "" {
				writeJSONError(w, http.StatusUnauthorized, "UNAUTHORIZED", "Authentication required")
				return
			}

			resp, err := m.client.CheckPermission(r.Context(), token, "", "", nil)
			if err != nil {
				writeJSONError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to check role")
				return
			}

			hasRole := false
			for _, r := range resp.Roles {
				if r == role {
					hasRole = true
					break
				}
			}

			if !hasRole {
				writeJSONError(w, http.StatusForbidden, "FORBIDDEN", "Required role not found: "+role)
				return
			}

			ctx := r.Context()
			ctx = context.WithValue(ctx, RolesKey, resp.Roles)
			ctx = context.WithValue(ctx, PermissionsKey, resp.Permissions)

			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// OptionalAuth validates JWT token if present but doesn't require it
func (m *ChiMiddleware) OptionalAuth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		authHeader := r.Header.Get("Authorization")
		if authHeader == "" {
			next.ServeHTTP(w, r)
			return
		}

		token := strings.TrimPrefix(authHeader, "Bearer ")
		if token == authHeader {
			next.ServeHTTP(w, r)
			return
		}

		resp, err := m.client.ValidateToken(r.Context(), token)
		if err == nil && resp.Valid {
			ctx := r.Context()
			ctx = context.WithValue(ctx, UserIDKey, resp.UserId)
			ctx = context.WithValue(ctx, EmailKey, resp.Email)
			ctx = context.WithValue(ctx, TokenKey, token)
			r = r.WithContext(ctx)
		}

		next.ServeHTTP(w, r)
	})
}

// ChiGetUserID retrieves the user ID from context
func ChiGetUserID(r *http.Request) string {
	if userID, ok := r.Context().Value(UserIDKey).(string); ok {
		return userID
	}
	return ""
}

// ChiGetEmail retrieves the email from context
func ChiGetEmail(r *http.Request) string {
	if email, ok := r.Context().Value(EmailKey).(string); ok {
		return email
	}
	return ""
}

// ChiGetRoles retrieves the roles from context
func ChiGetRoles(r *http.Request) []string {
	if roles, ok := r.Context().Value(RolesKey).([]string); ok {
		return roles
	}
	return nil
}
