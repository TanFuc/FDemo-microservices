package authclient

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"
)

// HTTPMiddleware provides authentication and authorization middleware for standard net/http
type HTTPMiddleware struct {
	client *Client
}

// NewHTTPMiddleware creates a new HTTP middleware instance
func NewHTTPMiddleware(client *Client) *HTTPMiddleware {
	return &HTTPMiddleware{client: client}
}

// RequireAuth validates JWT token and stores user info in context
func (m *HTTPMiddleware) RequireAuth(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
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

		ctx := r.Context()
		ctx = context.WithValue(ctx, UserIDKey, resp.UserId)
		ctx = context.WithValue(ctx, EmailKey, resp.Email)
		ctx = context.WithValue(ctx, TokenKey, token)

		next.ServeHTTP(w, r.WithContext(ctx))
	}
}

// RequireAuthHandler wraps http.Handler instead of http.HandlerFunc
func (m *HTTPMiddleware) RequireAuthHandler(next http.Handler) http.Handler {
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

		ctx := r.Context()
		ctx = context.WithValue(ctx, UserIDKey, resp.UserId)
		ctx = context.WithValue(ctx, EmailKey, resp.Email)
		ctx = context.WithValue(ctx, TokenKey, token)

		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// RequirePermission checks if user has specific permission
func (m *HTTPMiddleware) RequirePermission(resource, action string, next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
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
	}
}

// RequireRole checks if user has specific role
func (m *HTTPMiddleware) RequireRole(role string, next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
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
	}
}

// OptionalAuth validates JWT token if present but doesn't require it
func (m *HTTPMiddleware) OptionalAuth(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
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
	}
}

// HTTPGetUserID retrieves the user ID from request context
func HTTPGetUserID(r *http.Request) string {
	if userID, ok := r.Context().Value(UserIDKey).(string); ok {
		return userID
	}
	return ""
}

// HTTPGetEmail retrieves the email from request context
func HTTPGetEmail(r *http.Request) string {
	if email, ok := r.Context().Value(EmailKey).(string); ok {
		return email
	}
	return ""
}

// HTTPGetRoles retrieves the roles from request context
func HTTPGetRoles(r *http.Request) []string {
	if roles, ok := r.Context().Value(RolesKey).([]string); ok {
		return roles
	}
	return nil
}

// writeJSONError writes a JSON error response
func writeJSONError(w http.ResponseWriter, statusCode int, code, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": false,
		"code":    code,
		"message": message,
	})
}
