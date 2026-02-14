package service

import (
	"context"
	"time"

	"github.com/google/uuid"

	"microservices/auth/internal/cache"
	"microservices/auth/internal/model"
	"microservices/auth/internal/queue"
	"microservices/auth/pkg/errors"
	"microservices/auth/pkg/logger"
)

type RegisterInput struct {
	Email    string `json:"email" validate:"required,email"`
	Password string `json:"password" validate:"required,min=8,max=100"`
	FullName string `json:"fullName" validate:"required,max=255"`
}

type LoginInput struct {
	Email    string `json:"email" validate:"required,email"`
	Password string `json:"password" validate:"required"`
}

type LoginResponse struct {
	User   *UserResponse `json:"user"`
	Tokens *AuthTokens   `json:"tokens"`
}

type RegisterResponse struct {
	User *UserResponse `json:"user"`
}

type SessionInfo struct {
	ID         string    `json:"id"`
	DeviceID   *string   `json:"deviceId,omitempty"`
	DeviceName *string   `json:"deviceName,omitempty"`
	Browser    *string   `json:"browser,omitempty"`
	OS         *string   `json:"os,omitempty"`
	IPAddress  string    `json:"ipAddress"`
	Location   *string   `json:"location,omitempty"`
	CreatedAt  time.Time `json:"createdAt"`
	ExpiresAt  time.Time `json:"expiresAt"`
	IsCurrent  bool      `json:"isCurrent"`
}

type AuthService struct {
	userService  *UserService
	tokenService *TokenService
	cache        *cache.RedisClient
	natsClient   *queue.NATSClient
}

func NewAuthService(
	userService *UserService,
	tokenService *TokenService,
	cache *cache.RedisClient,
	natsClient *queue.NATSClient,
) *AuthService {
	return &AuthService{
		userService:  userService,
		tokenService: tokenService,
		cache:        cache,
		natsClient:   natsClient,
	}
}

func (s *AuthService) Register(ctx context.Context, input *RegisterInput) (*RegisterResponse, error) {
	// Create user
	user, err := s.userService.CreateUser(ctx, &CreateUserInput{
		Email:    input.Email,
		Password: input.Password,
		FullName: input.FullName,
	})
	if err != nil {
		return nil, err
	}

	// Publish user.registered event
	if s.natsClient != nil {
		event := &queue.UserRegisteredEvent{
			UserID:       user.ID.String(),
			Email:        user.Email,
			FullName:     user.FullName,
			RegisteredAt: time.Now().Format(time.RFC3339),
		}
		if err := s.natsClient.PublishUserRegistered(ctx, event); err != nil {
			logger.Warn().Err(err).Msg("Failed to publish user.registered event")
		}
	}

	// Get user with roles
	userWithRoles, err := s.userService.GetUserWithRoles(ctx, user.ID)
	if err != nil {
		logger.Warn().Err(err).Msg("Failed to get user with roles")
		userWithRoles = user
	}

	return &RegisterResponse{
		User: s.userService.ToUserResponse(userWithRoles),
	}, nil
}

func (s *AuthService) Login(ctx context.Context, input *LoginInput, deviceInfo *DeviceInfo) (*LoginResponse, error) {
	// Validate credentials
	user, err := s.userService.ValidateCredentials(ctx, input.Email, input.Password)
	if err != nil {
		// Record failed login
		if user != nil {
			ua := deviceInfo.UserAgent
			s.userService.RecordLoginHistory(ctx, user.ID, model.LoginStatusFailed, model.AuthMethodPassword, deviceInfo.IPAddress, &ua)
		}
		return nil, err
	}

	// Get user permissions and cache them
	permissions, err := s.userService.GetUserPermissions(ctx, user.ID)
	if err != nil {
		logger.Warn().Err(err).Msg("Failed to get user permissions")
	} else if len(permissions) > 0 {
		if err := s.cache.CacheUserPermissions(ctx, user.ID.String(), permissions, PermissionsTTL); err != nil {
			logger.Warn().Err(err).Msg("Failed to cache user permissions")
		}
	}

	// Generate token pair
	tokens, err := s.tokenService.GenerateTokenPair(ctx, user.ID, user.Email, deviceInfo)
	if err != nil {
		return nil, err
	}

	// Update login info
	if err := s.userService.UpdateLoginInfo(ctx, user.ID, deviceInfo.IPAddress); err != nil {
		logger.Warn().Err(err).Msg("Failed to update login info")
	}

	// Record successful login
	ua := deviceInfo.UserAgent
	s.userService.RecordLoginHistory(ctx, user.ID, model.LoginStatusSuccess, model.AuthMethodPassword, deviceInfo.IPAddress, &ua)

	// Get user with roles
	userWithRoles, err := s.userService.GetUserWithRoles(ctx, user.ID)
	if err != nil {
		logger.Warn().Err(err).Msg("Failed to get user with roles")
		userWithRoles = user
	}

	return &LoginResponse{
		User:   s.userService.ToUserResponse(userWithRoles),
		Tokens: tokens,
	}, nil
}

func (s *AuthService) RefreshTokens(ctx context.Context, refreshToken string, deviceInfo *DeviceInfo) (*AuthTokens, error) {
	return s.tokenService.RotateRefreshToken(ctx, refreshToken, deviceInfo)
}

func (s *AuthService) Logout(ctx context.Context, userID uuid.UUID, accessTokenJti string, accessTokenExp time.Time, deviceID *string) error {
	// Blacklist the access token
	remainingTTL := time.Until(accessTokenExp)
	if remainingTTL > 0 {
		if err := s.tokenService.RevokeToken(ctx, accessTokenJti, remainingTTL); err != nil {
			logger.Warn().Err(err).Msg("Failed to blacklist access token")
		}
	}

	// Revoke refresh tokens for the device
	if deviceID != nil && *deviceID != "" {
		if err := s.tokenService.RevokeDeviceTokens(ctx, userID, *deviceID); err != nil {
			logger.Warn().Err(err).Msg("Failed to revoke device tokens")
		}
	}

	// Remove active session from cache
	if deviceID != nil && *deviceID != "" {
		if err := s.cache.RemoveActiveSession(ctx, userID.String(), *deviceID); err != nil {
			logger.Warn().Err(err).Msg("Failed to remove active session")
		}
	}

	// Invalidate permissions cache
	if err := s.userService.InvalidateUserPermissionsCache(ctx, userID); err != nil {
		logger.Warn().Err(err).Msg("Failed to invalidate permissions cache")
	}

	return nil
}

func (s *AuthService) LogoutAllDevices(ctx context.Context, userID uuid.UUID) error {
	// Revoke all refresh tokens
	if err := s.tokenService.RevokeAllUserTokens(ctx, userID); err != nil {
		return errors.Wrap(err, "LOGOUT_FAILED", "Failed to revoke all tokens", 500)
	}

	// Invalidate permissions cache
	if err := s.userService.InvalidateUserPermissionsCache(ctx, userID); err != nil {
		logger.Warn().Err(err).Msg("Failed to invalidate permissions cache")
	}

	return nil
}

func (s *AuthService) GetProfile(ctx context.Context, userID uuid.UUID) (*UserResponse, error) {
	user, err := s.userService.GetUserWithRoles(ctx, userID)
	if err != nil {
		return nil, err
	}
	return s.userService.ToUserResponse(user), nil
}

func (s *AuthService) GetActiveSessions(ctx context.Context, userID uuid.UUID, currentDeviceID string) ([]SessionInfo, error) {
	tokens, err := s.tokenService.GetUserSessions(ctx, userID)
	if err != nil {
		return nil, errors.Wrap(err, "SESSION_FETCH_FAILED", "Failed to get active sessions", 500)
	}

	sessions := make([]SessionInfo, 0, len(tokens))
	for _, token := range tokens {
		isCurrent := token.DeviceID != nil && *token.DeviceID == currentDeviceID
		sessions = append(sessions, SessionInfo{
			ID:         token.ID.String(),
			DeviceID:   token.DeviceID,
			DeviceName: token.DeviceName,
			Browser:    token.Browser,
			OS:         token.OS,
			IPAddress:  token.IPAddress,
			Location:   token.Location,
			CreatedAt:  token.CreatedAt,
			ExpiresAt:  token.ExpiresAt,
			IsCurrent:  isCurrent,
		})
	}

	return sessions, nil
}

func (s *AuthService) ValidateUser(ctx context.Context, userID uuid.UUID) (*model.User, error) {
	return s.userService.FindByID(ctx, userID)
}

func (s *AuthService) CheckPermission(ctx context.Context, userID uuid.UUID, permission string) (bool, error) {
	return s.userService.HasPermission(ctx, userID, permission)
}

func (s *AuthService) CheckAnyPermission(ctx context.Context, userID uuid.UUID, permissions []string) (bool, error) {
	return s.userService.HasAnyPermission(ctx, userID, permissions)
}
