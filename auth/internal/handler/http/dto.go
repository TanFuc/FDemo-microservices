package http

type RegisterRequest struct {
	Email    string `json:"email" validate:"required,email"`
	Password string `json:"password" validate:"required,min=8,max=100"`
	FullName string `json:"fullName" validate:"required,max=255"`
}

type LoginRequest struct {
	Email    string `json:"email" validate:"required,email"`
	Password string `json:"password" validate:"required"`
}

type RefreshTokenRequest struct {
	RefreshToken string `json:"refreshToken" validate:"required"`
}

type LogoutRequest struct {
	DeviceID string `json:"deviceId,omitempty"`
}

type AuthTokensResponse struct {
	AccessToken  string `json:"accessToken"`
	RefreshToken string `json:"refreshToken"`
	ExpiresIn    int64  `json:"expiresIn"`
}

type UserResponse struct {
	ID        string   `json:"id"`
	Email     string   `json:"email"`
	FullName  string   `json:"fullName"`
	IsActive  bool     `json:"isActive"`
	Roles     []string `json:"roles"`
	CreatedAt string   `json:"createdAt"`
	UpdatedAt string   `json:"updatedAt"`
}

type LoginResponse struct {
	User   *UserResponse       `json:"user"`
	Tokens *AuthTokensResponse `json:"tokens"`
}

type RegisterResponse struct {
	User *UserResponse `json:"user"`
}

type SessionResponse struct {
	ID         string  `json:"id"`
	DeviceID   *string `json:"deviceId,omitempty"`
	DeviceName *string `json:"deviceName,omitempty"`
	Browser    *string `json:"browser,omitempty"`
	OS         *string `json:"os,omitempty"`
	IPAddress  string  `json:"ipAddress"`
	Location   *string `json:"location,omitempty"`
	CreatedAt  string  `json:"createdAt"`
	ExpiresAt  string  `json:"expiresAt"`
	IsCurrent  bool    `json:"isCurrent"`
}

type CheckPermissionResponse struct {
	HasAccess bool `json:"hasAccess"`
}
