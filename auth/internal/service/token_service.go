package service

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"

	"microservices/auth/internal/cache"
	"microservices/auth/internal/config"
	"microservices/auth/internal/model"
	"microservices/auth/internal/repository"
	"microservices/auth/pkg/errors"
	"microservices/auth/pkg/logger"
)

const (
	BcryptCost = 12
)

type AccessTokenClaims struct {
	jwt.RegisteredClaims
	Email string `json:"email"`
}

type RefreshTokenClaims struct {
	jwt.RegisteredClaims
	DeviceID string `json:"deviceId"`
}

type AuthTokens struct {
	AccessToken  string `json:"accessToken"`
	RefreshToken string `json:"refreshToken"`
	ExpiresIn    int64  `json:"expiresIn"`
}

type DeviceInfo struct {
	DeviceID  string
	IPAddress string
	UserAgent string
}

type TokenService struct {
	cfg              *config.Config
	refreshTokenRepo repository.RefreshTokenRepository
	cache            *cache.RedisClient
}

func NewTokenService(
	cfg *config.Config,
	refreshTokenRepo repository.RefreshTokenRepository,
	cache *cache.RedisClient,
) *TokenService {
	return &TokenService{
		cfg:              cfg,
		refreshTokenRepo: refreshTokenRepo,
		cache:            cache,
	}
}

func (s *TokenService) GenerateTokenPair(ctx context.Context, userID uuid.UUID, email string, deviceInfo *DeviceInfo) (*AuthTokens, error) {
	jti := uuid.New().String()
	now := time.Now()

	// Generate access token
	accessClaims := AccessTokenClaims{
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   userID.String(),
			ID:        jti,
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(s.cfg.JWT.AccessExpiry)),
		},
		Email: email,
	}

	accessToken := jwt.NewWithClaims(jwt.SigningMethodHS256, accessClaims)
	accessTokenString, err := accessToken.SignedString([]byte(s.cfg.JWT.AccessSecret))
	if err != nil {
		return nil, errors.Wrap(err, "TOKEN_GENERATION_FAILED", "Failed to generate access token", 500)
	}

	// Generate refresh token
	refreshJti := uuid.New().String()
	refreshClaims := RefreshTokenClaims{
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   userID.String(),
			ID:        refreshJti,
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(s.cfg.JWT.RefreshExpiry)),
		},
		DeviceID: deviceInfo.DeviceID,
	}

	refreshToken := jwt.NewWithClaims(jwt.SigningMethodHS256, refreshClaims)
	refreshTokenString, err := refreshToken.SignedString([]byte(s.cfg.JWT.RefreshSecret))
	if err != nil {
		return nil, errors.Wrap(err, "TOKEN_GENERATION_FAILED", "Failed to generate refresh token", 500)
	}

	// Hash refresh token for storage
	tokenHash := hashToken(refreshTokenString)

	// Save refresh token to database
	refreshTokenEntity := &model.RefreshToken{
		ID:        uuid.MustParse(refreshJti),
		UserID:    userID,
		TokenHash: tokenHash,
		DeviceID:  &deviceInfo.DeviceID,
		IPAddress: deviceInfo.IPAddress,
		ExpiresAt: now.Add(s.cfg.JWT.RefreshExpiry),
	}

	if deviceInfo.UserAgent != "" {
		refreshTokenEntity.Browser = &deviceInfo.UserAgent
	}

	if err := s.refreshTokenRepo.Create(ctx, refreshTokenEntity); err != nil {
		return nil, errors.Wrap(err, "TOKEN_STORAGE_FAILED", "Failed to store refresh token", 500)
	}

	// Set active session in Redis
	sessionData := &cache.SessionData{
		RefreshTokenID: refreshJti,
		IPAddress:      deviceInfo.IPAddress,
		DeviceInfo:     deviceInfo.UserAgent,
		CreatedAt:      now.Format(time.RFC3339),
	}
	if err := s.cache.SetActiveSession(ctx, userID.String(), deviceInfo.DeviceID, sessionData, s.cfg.JWT.RefreshExpiry); err != nil {
		logger.Warn().Err(err).Msg("Failed to set active session in cache")
	}

	return &AuthTokens{
		AccessToken:  accessTokenString,
		RefreshToken: refreshTokenString,
		ExpiresIn:    int64(s.cfg.JWT.AccessExpiry.Seconds()),
	}, nil
}

func (s *TokenService) VerifyAccessToken(ctx context.Context, tokenString string) (*AccessTokenClaims, error) {
	token, err := jwt.ParseWithClaims(tokenString, &AccessTokenClaims{}, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return []byte(s.cfg.JWT.AccessSecret), nil
	})

	if err != nil {
		if err == jwt.ErrTokenExpired {
			return nil, errors.ErrTokenExpired
		}
		return nil, errors.ErrTokenInvalid
	}

	claims, ok := token.Claims.(*AccessTokenClaims)
	if !ok || !token.Valid {
		return nil, errors.ErrTokenInvalid
	}

	// Check if token is blacklisted
	blacklisted, err := s.cache.IsTokenBlacklisted(ctx, claims.ID)
	if err != nil {
		logger.Warn().Err(err).Msg("Failed to check token blacklist")
	}
	if blacklisted {
		return nil, errors.ErrTokenBlacklisted
	}

	return claims, nil
}

func (s *TokenService) VerifyRefreshToken(ctx context.Context, tokenString string) (*RefreshTokenClaims, error) {
	token, err := jwt.ParseWithClaims(tokenString, &RefreshTokenClaims{}, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return []byte(s.cfg.JWT.RefreshSecret), nil
	})

	if err != nil {
		if err == jwt.ErrTokenExpired {
			return nil, errors.ErrRefreshTokenExpired
		}
		return nil, errors.ErrRefreshTokenInvalid
	}

	claims, ok := token.Claims.(*RefreshTokenClaims)
	if !ok || !token.Valid {
		return nil, errors.ErrRefreshTokenInvalid
	}

	return claims, nil
}

func (s *TokenService) RotateRefreshToken(ctx context.Context, oldTokenString string, deviceInfo *DeviceInfo) (*AuthTokens, error) {
	// Verify old refresh token
	claims, err := s.VerifyRefreshToken(ctx, oldTokenString)
	if err != nil {
		return nil, err
	}

	// Find token in database
	tokenID, err := uuid.Parse(claims.ID)
	if err != nil {
		return nil, errors.ErrRefreshTokenInvalid
	}

	storedToken, err := s.refreshTokenRepo.FindByID(ctx, tokenID)
	if err != nil {
		return nil, errors.ErrRefreshTokenInvalid
	}

	// Verify token is not revoked
	if storedToken.IsRevoked {
		return nil, errors.ErrTokenRevoked
	}

	// Verify token hash matches
	if !verifyTokenHash(oldTokenString, storedToken.TokenHash) {
		return nil, errors.ErrRefreshTokenInvalid
	}

	// Revoke old token
	if err := s.refreshTokenRepo.Revoke(ctx, tokenID, "token_rotation"); err != nil {
		logger.Error().Err(err).Msg("Failed to revoke old refresh token")
	}

	// Generate new token pair
	userID, err := uuid.Parse(claims.Subject)
	if err != nil {
		return nil, errors.ErrRefreshTokenInvalid
	}

	// Use existing device ID from token if not provided
	if deviceInfo.DeviceID == "" {
		deviceInfo.DeviceID = claims.DeviceID
	}

	return s.GenerateTokenPair(ctx, userID, "", deviceInfo)
}

func (s *TokenService) RevokeToken(ctx context.Context, jti string, remainingTTL time.Duration) error {
	return s.cache.BlacklistToken(ctx, jti, remainingTTL)
}

func (s *TokenService) RevokeAllUserTokens(ctx context.Context, userID uuid.UUID) error {
	return s.refreshTokenRepo.RevokeAllByUserID(ctx, userID, "logout_all_devices")
}

func (s *TokenService) RevokeDeviceTokens(ctx context.Context, userID uuid.UUID, deviceID string) error {
	err := s.refreshTokenRepo.RevokeByDeviceID(ctx, userID, deviceID, "device_logout")
	if err != nil {
		return err
	}
	return s.cache.RemoveActiveSession(ctx, userID.String(), deviceID)
}

func (s *TokenService) GetUserSessions(ctx context.Context, userID uuid.UUID) ([]model.RefreshToken, error) {
	return s.refreshTokenRepo.FindActiveByUserID(ctx, userID)
}

func (s *TokenService) CleanupExpiredTokens(ctx context.Context) (int64, error) {
	return s.refreshTokenRepo.DeleteExpired(ctx)
}

// Helper functions
func hashToken(token string) string {
	hash := sha256.Sum256([]byte(token))
	return hex.EncodeToString(hash[:])
}

func verifyTokenHash(token, hash string) bool {
	return hashToken(token) == hash
}

func HashPassword(password string) (string, error) {
	bytes, err := bcrypt.GenerateFromPassword([]byte(password), BcryptCost)
	if err != nil {
		return "", err
	}
	return string(bytes), nil
}

func CheckPassword(password, hash string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))
	return err == nil
}
