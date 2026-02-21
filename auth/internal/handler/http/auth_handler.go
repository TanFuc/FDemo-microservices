package http

import (
	"strings"
	"time"

	"github.com/go-playground/validator/v10"
	"github.com/gofiber/fiber/v2"

	"microservices/auth/internal/config"
	"microservices/auth/internal/middleware"
	"microservices/auth/internal/service"
	"microservices/auth/pkg/errors"
	"microservices/auth/pkg/logger"
	"microservices/auth/pkg/response"
)

type AuthHandler struct {
	authService *service.AuthService
	cfg         *config.Config
	validate    *validator.Validate
}

func NewAuthHandler(authService *service.AuthService, cfg *config.Config) *AuthHandler {
	return &AuthHandler{
		authService: authService,
		cfg:         cfg,
		validate:    validator.New(),
	}
}

func (h *AuthHandler) Register(c *fiber.Ctx) error {
	var req RegisterRequest
	if err := c.BodyParser(&req); err != nil {
		logger.Error().Err(err).Msg("Register: Body parsing failed")
		return response.BadRequest(c, "Invalid request body")
	}

	if err := h.validate.Struct(&req); err != nil {
		validationErrors := formatValidationErrors(err)
		logger.Error().Interface("errors", validationErrors).Msg("Register: Validation failed")
		return response.ValidationError(c, validationErrors)
	}

	input := &service.RegisterInput{
		Email:    req.Email,
		Password: req.Password,
		FullName: req.FullName,
	}

	result, err := h.authService.Register(c.Context(), input)
	if err != nil {
		return response.Error(c, err)
	}

	return response.CreatedWithMessage(c, &RegisterResponse{
		User: toUserResponseDTO(result.User),
	}, "User registered successfully")
}

func (h *AuthHandler) Login(c *fiber.Ctx) error {
	var req LoginRequest
	if err := c.BodyParser(&req); err != nil {
		return response.BadRequest(c, "Invalid request body")
	}

	if err := h.validate.Struct(&req); err != nil {
		validationErrors := formatValidationErrors(err)
		return response.ValidationError(c, validationErrors)
	}

	deviceInfo := extractDeviceInfo(c)

	input := &service.LoginInput{
		Email:    req.Email,
		Password: req.Password,
	}

	result, err := h.authService.Login(c.Context(), input, deviceInfo)
	if err != nil {
		return response.Error(c, err)
	}

	h.setTokenCookies(c, result.Tokens.AccessToken, result.Tokens.RefreshToken)

	return response.Success(c, &LoginResponse{
		User: toUserResponseDTO(result.User),
		Tokens: &AuthTokensResponse{
			AccessToken:  result.Tokens.AccessToken,
			RefreshToken: result.Tokens.RefreshToken,
			ExpiresIn:    result.Tokens.ExpiresIn,
		},
	})
}

func (h *AuthHandler) RefreshToken(c *fiber.Ctx) error {
	var req RefreshTokenRequest
	c.BodyParser(&req)

	refreshToken := req.RefreshToken
	if refreshToken == "" {
		refreshToken = c.Cookies("refresh_token")
	}

	if refreshToken == "" {
		return response.BadRequest(c, "Refresh token is required")
	}

	deviceInfo := extractDeviceInfo(c)

	tokens, err := h.authService.RefreshTokens(c.Context(), refreshToken, deviceInfo)
	if err != nil {
		return response.Error(c, err)
	}

	h.setTokenCookies(c, tokens.AccessToken, tokens.RefreshToken)

	return response.Success(c, &AuthTokensResponse{
		AccessToken:  tokens.AccessToken,
		RefreshToken: tokens.RefreshToken,
		ExpiresIn:    tokens.ExpiresIn,
	})
}

func (h *AuthHandler) Logout(c *fiber.Ctx) error {
	user := middleware.GetAuthenticatedUser(c)
	if user == nil {
		return response.Error(c, errors.New("UNAUTHORIZED", "Authentication required", fiber.StatusUnauthorized))
	}

	var req LogoutRequest
	c.BodyParser(&req)

	deviceID := req.DeviceID
	if deviceID == "" {
		deviceID = middleware.GetDeviceID(c)
	}

	err := h.authService.Logout(c.Context(), user.ID, user.JTI, user.ExpireAt, &deviceID)
	if err != nil {
		return response.Error(c, err)
	}

	h.clearTokenCookies(c)

	return response.SuccessWithMessage(c, nil, "Logged out successfully")
}

func (h *AuthHandler) LogoutAll(c *fiber.Ctx) error {
	user := middleware.GetAuthenticatedUser(c)
	if user == nil {
		return response.Error(c, errors.New("UNAUTHORIZED", "Authentication required", fiber.StatusUnauthorized))
	}

	err := h.authService.LogoutAllDevices(c.Context(), user.ID)
	if err != nil {
		return response.Error(c, err)
	}

	h.clearTokenCookies(c)

	return response.SuccessWithMessage(c, nil, "Logged out from all devices successfully")
}

func (h *AuthHandler) GetProfile(c *fiber.Ctx) error {
	user := middleware.GetAuthenticatedUser(c)
	if user == nil {
		return response.Error(c, errors.New("UNAUTHORIZED", "Authentication required", fiber.StatusUnauthorized))
	}

	profile, err := h.authService.GetProfile(c.Context(), user.ID)
	if err != nil {
		return response.Error(c, err)
	}

	return response.Success(c, toUserResponseDTO(profile))
}

func (h *AuthHandler) GetSessions(c *fiber.Ctx) error {
	user := middleware.GetAuthenticatedUser(c)
	if user == nil {
		return response.Error(c, errors.New("UNAUTHORIZED", "Authentication required", fiber.StatusUnauthorized))
	}

	currentDeviceID := middleware.GetDeviceID(c)
	sessions, err := h.authService.GetActiveSessions(c.Context(), user.ID, currentDeviceID)
	if err != nil {
		return response.Error(c, err)
	}

	sessionResponses := make([]SessionResponse, len(sessions))
	for i, s := range sessions {
		sessionResponses[i] = SessionResponse{
			ID:         s.ID,
			DeviceID:   s.DeviceID,
			DeviceName: s.DeviceName,
			Browser:    s.Browser,
			OS:         s.OS,
			IPAddress:  s.IPAddress,
			Location:   s.Location,
			CreatedAt:  s.CreatedAt.Format(time.RFC3339),
			ExpiresAt:  s.ExpiresAt.Format(time.RFC3339),
			IsCurrent:  s.IsCurrent,
		}
	}

	return response.Success(c, sessionResponses)
}

func (h *AuthHandler) CheckPermission(c *fiber.Ctx) error {
	return response.Success(c, &CheckPermissionResponse{
		HasAccess: true,
	})
}

func parseSameSite(s string) string {
	switch strings.ToLower(s) {
	case "strict":
		return "Strict"
	case "none":
		return "None"
	default:
		return "Lax"
	}
}

func (h *AuthHandler) setTokenCookies(c *fiber.Ctx, accessToken, refreshToken string) {
	sameSite := parseSameSite(h.cfg.Cookie.SameSite)

	c.Cookie(&fiber.Cookie{
		Name:     "access_token",
		Value:    accessToken,
		Path:     "/api",
		Domain:   h.cfg.Cookie.Domain,
		MaxAge:   int(h.cfg.JWT.AccessExpiry.Seconds()),
		Secure:   h.cfg.Cookie.Secure,
		HTTPOnly: true,
		SameSite: sameSite,
	})

	c.Cookie(&fiber.Cookie{
		Name:     "refresh_token",
		Value:    refreshToken,
		Path:     "/api/v1/auth/refresh",
		Domain:   h.cfg.Cookie.Domain,
		MaxAge:   int(h.cfg.JWT.RefreshExpiry.Seconds()),
		Secure:   h.cfg.Cookie.Secure,
		HTTPOnly: true,
		SameSite: sameSite,
	})
}

func (h *AuthHandler) clearTokenCookies(c *fiber.Ctx) {
	sameSite := parseSameSite(h.cfg.Cookie.SameSite)

	c.Cookie(&fiber.Cookie{
		Name:     "access_token",
		Value:    "",
		Path:     "/api",
		Domain:   h.cfg.Cookie.Domain,
		MaxAge:   -1,
		Secure:   h.cfg.Cookie.Secure,
		HTTPOnly: true,
		SameSite: sameSite,
	})

	c.Cookie(&fiber.Cookie{
		Name:     "refresh_token",
		Value:    "",
		Path:     "/api/v1/auth/refresh",
		Domain:   h.cfg.Cookie.Domain,
		MaxAge:   -1,
		Secure:   h.cfg.Cookie.Secure,
		HTTPOnly: true,
		SameSite: sameSite,
	})
}

func extractDeviceInfo(c *fiber.Ctx) *service.DeviceInfo {
	return &service.DeviceInfo{
		DeviceID:  middleware.GetDeviceID(c),
		IPAddress: middleware.GetClientIP(c),
		UserAgent: middleware.GetUserAgent(c),
	}
}

func toUserResponseDTO(user *service.UserResponse) *UserResponse {
	if user == nil {
		return nil
	}
	return &UserResponse{
		ID:        user.ID.String(),
		Email:     user.Email,
		FullName:  user.FullName,
		IsActive:  user.IsActive,
		Roles:     user.Roles,
		CreatedAt: user.CreatedAt.Format(time.RFC3339),
		UpdatedAt: user.UpdatedAt.Format(time.RFC3339),
	}
}

func formatValidationErrors(err error) []string {
	var errors []string
	if validationErrors, ok := err.(validator.ValidationErrors); ok {
		for _, e := range validationErrors {
			errors = append(errors, formatFieldError(e))
		}
	}
	return errors
}

func formatFieldError(e validator.FieldError) string {
	switch e.Tag() {
	case "required":
		return e.Field() + " is required"
	case "email":
		return e.Field() + " must be a valid email address"
	case "min":
		return e.Field() + " must be at least " + e.Param() + " characters"
	case "max":
		return e.Field() + " must be at most " + e.Param() + " characters"
	default:
		return e.Field() + " is invalid"
	}
}
