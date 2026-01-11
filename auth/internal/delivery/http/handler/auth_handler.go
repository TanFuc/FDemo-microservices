package handler

import (
	"time"

	"github.com/go-playground/validator/v10"
	"github.com/gofiber/fiber/v2"

	"tafu-auth/internal/delivery/http/dto"
	"tafu-auth/internal/delivery/http/middleware"
	"tafu-auth/internal/domain/service"
	"tafu-auth/pkg/errors"
	"tafu-auth/pkg/response"
)

type AuthHandler struct {
	authService *service.AuthService
	validate    *validator.Validate
}

func NewAuthHandler(authService *service.AuthService) *AuthHandler {
	return &AuthHandler{
		authService: authService,
		validate:    validator.New(),
	}
}

// Register godoc
// @Summary Register a new user
// @Description Register a new user with email and password
// @Tags auth
// @Accept json
// @Produce json
// @Param request body dto.RegisterRequest true "Registration details"
// @Success 201 {object} response.Response{data=dto.RegisterResponse}
// @Failure 400 {object} response.ErrorResponse
// @Failure 409 {object} response.ErrorResponse
// @Router /auth/register [post]
func (h *AuthHandler) Register(c *fiber.Ctx) error {
	var req dto.RegisterRequest
	if err := c.BodyParser(&req); err != nil {
		return response.BadRequest(c, "Invalid request body")
	}

	if err := h.validate.Struct(&req); err != nil {
		validationErrors := formatValidationErrors(err)
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

	return response.CreatedWithMessage(c, &dto.RegisterResponse{
		User: toUserResponseDTO(result.User),
	}, "User registered successfully")
}

// Login godoc
// @Summary Login user
// @Description Login with email and password
// @Tags auth
// @Accept json
// @Produce json
// @Param request body dto.LoginRequest true "Login credentials"
// @Param X-Device-ID header string false "Device ID"
// @Success 200 {object} response.Response{data=dto.LoginResponse}
// @Failure 400 {object} response.ErrorResponse
// @Failure 401 {object} response.ErrorResponse
// @Router /auth/login [post]
func (h *AuthHandler) Login(c *fiber.Ctx) error {
	var req dto.LoginRequest
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

	return response.Success(c, &dto.LoginResponse{
		User: toUserResponseDTO(result.User),
		Tokens: &dto.AuthTokensResponse{
			AccessToken:  result.Tokens.AccessToken,
			RefreshToken: result.Tokens.RefreshToken,
			ExpiresIn:    result.Tokens.ExpiresIn,
		},
	})
}

// RefreshToken godoc
// @Summary Refresh access token
// @Description Get new access token using refresh token
// @Tags auth
// @Accept json
// @Produce json
// @Param request body dto.RefreshTokenRequest true "Refresh token"
// @Param X-Device-ID header string false "Device ID"
// @Success 200 {object} response.Response{data=dto.AuthTokensResponse}
// @Failure 400 {object} response.ErrorResponse
// @Failure 401 {object} response.ErrorResponse
// @Router /auth/refresh [post]
func (h *AuthHandler) RefreshToken(c *fiber.Ctx) error {
	var req dto.RefreshTokenRequest
	if err := c.BodyParser(&req); err != nil {
		return response.BadRequest(c, "Invalid request body")
	}

	if err := h.validate.Struct(&req); err != nil {
		validationErrors := formatValidationErrors(err)
		return response.ValidationError(c, validationErrors)
	}

	deviceInfo := extractDeviceInfo(c)

	tokens, err := h.authService.RefreshTokens(c.Context(), req.RefreshToken, deviceInfo)
	if err != nil {
		return response.Error(c, err)
	}

	return response.Success(c, &dto.AuthTokensResponse{
		AccessToken:  tokens.AccessToken,
		RefreshToken: tokens.RefreshToken,
		ExpiresIn:    tokens.ExpiresIn,
	})
}

// Logout godoc
// @Summary Logout user
// @Description Logout and invalidate tokens
// @Tags auth
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param request body dto.LogoutRequest false "Logout options"
// @Success 200 {object} response.Response
// @Failure 401 {object} response.ErrorResponse
// @Router /auth/logout [post]
func (h *AuthHandler) Logout(c *fiber.Ctx) error {
	user := middleware.GetAuthenticatedUser(c)
	if user == nil {
		return response.Error(c, errors.New("UNAUTHORIZED", "Authentication required", fiber.StatusUnauthorized))
	}

	var req dto.LogoutRequest
	c.BodyParser(&req) // Optional body

	deviceID := req.DeviceID
	if deviceID == "" {
		deviceID = middleware.GetDeviceID(c)
	}

	err := h.authService.Logout(c.Context(), user.ID, user.JTI, user.ExpireAt, &deviceID)
	if err != nil {
		return response.Error(c, err)
	}

	return response.SuccessWithMessage(c, nil, "Logged out successfully")
}

// LogoutAll godoc
// @Summary Logout from all devices
// @Description Logout and invalidate all tokens for the user
// @Tags auth
// @Accept json
// @Produce json
// @Security BearerAuth
// @Success 200 {object} response.Response
// @Failure 401 {object} response.ErrorResponse
// @Router /auth/logout-all [post]
func (h *AuthHandler) LogoutAll(c *fiber.Ctx) error {
	user := middleware.GetAuthenticatedUser(c)
	if user == nil {
		return response.Error(c, errors.New("UNAUTHORIZED", "Authentication required", fiber.StatusUnauthorized))
	}

	err := h.authService.LogoutAllDevices(c.Context(), user.ID)
	if err != nil {
		return response.Error(c, err)
	}

	return response.SuccessWithMessage(c, nil, "Logged out from all devices successfully")
}

// GetProfile godoc
// @Summary Get current user profile
// @Description Get the authenticated user's profile
// @Tags auth
// @Produce json
// @Security BearerAuth
// @Success 200 {object} response.Response{data=dto.UserResponse}
// @Failure 401 {object} response.ErrorResponse
// @Router /auth/profile [get]
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

// GetSessions godoc
// @Summary Get active sessions
// @Description Get all active sessions for the authenticated user
// @Tags auth
// @Produce json
// @Security BearerAuth
// @Success 200 {object} response.Response{data=[]dto.SessionResponse}
// @Failure 401 {object} response.ErrorResponse
// @Router /auth/sessions [get]
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

	sessionResponses := make([]dto.SessionResponse, len(sessions))
	for i, s := range sessions {
		sessionResponses[i] = dto.SessionResponse{
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

// CheckPermission godoc
// @Summary Check permission
// @Description Check if user has a specific permission (example endpoint)
// @Tags auth
// @Produce json
// @Security BearerAuth
// @Success 200 {object} response.Response{data=dto.CheckPermissionResponse}
// @Failure 401 {object} response.ErrorResponse
// @Failure 403 {object} response.ErrorResponse
// @Router /auth/check-permission [get]
func (h *AuthHandler) CheckPermission(c *fiber.Ctx) error {
	// This endpoint is protected by PermissionsGuard in the router
	// If we reach here, user has the required permission
	return response.Success(c, &dto.CheckPermissionResponse{
		HasAccess: true,
	})
}

// Helper functions
func extractDeviceInfo(c *fiber.Ctx) *service.DeviceInfo {
	return &service.DeviceInfo{
		DeviceID:  middleware.GetDeviceID(c),
		IPAddress: middleware.GetClientIP(c),
		UserAgent: middleware.GetUserAgent(c),
	}
}

func toUserResponseDTO(user *service.UserResponse) *dto.UserResponse {
	if user == nil {
		return nil
	}
	return &dto.UserResponse{
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
