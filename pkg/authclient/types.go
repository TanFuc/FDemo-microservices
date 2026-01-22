package authclient

// CheckPermissionRequest is the request for CheckPermission RPC
type CheckPermissionRequest struct {
	Token    string            `json:"token"`
	Resource string            `json:"resource"`
	Action   string            `json:"action"`
	Context  map[string]string `json:"context,omitempty"`
}

// CheckPermissionResponse is the response for CheckPermission RPC
type CheckPermissionResponse struct {
	Allowed     bool     `json:"allowed"`
	UserId      string   `json:"userId"`
	Reason      string   `json:"reason"`
	Roles       []string `json:"roles"`
	Permissions []string `json:"permissions"`
}

// GetUserInfoRequest is the request for GetUserInfo RPC
type GetUserInfoRequest struct {
	UserId string `json:"userId"`
}

// GetUserInfoResponse is the response for GetUserInfo RPC
type GetUserInfoResponse struct {
	UserId   string   `json:"userId"`
	Email    string   `json:"email"`
	FullName string   `json:"fullName"`
	Status   string   `json:"status"`
	Roles    []string `json:"roles"`
}

// ValidateTokenRequest is the request for ValidateToken RPC
type ValidateTokenRequest struct {
	Token string `json:"token"`
}

// ValidateTokenResponse is the response for ValidateToken RPC
type ValidateTokenResponse struct {
	Valid     bool   `json:"valid"`
	UserId    string `json:"userId"`
	Email     string `json:"email"`
	ExpiresAt int64  `json:"expiresAt"`
}

// AuthorizeRequest is the request for Authorize RPC
type AuthorizeRequest struct {
	UserId   string `json:"userId"`
	Resource string `json:"resource"`
	Action   string `json:"action"`
}

// AuthorizeResponse is the response for Authorize RPC
type AuthorizeResponse struct {
	Allowed bool   `json:"allowed"`
	Reason  string `json:"reason"`
}

// UserInfo represents authenticated user information stored in context
type UserInfo struct {
	UserID      string   `json:"userId"`
	Email       string   `json:"email"`
	Roles       []string `json:"roles"`
	Permissions []string `json:"permissions"`
}
