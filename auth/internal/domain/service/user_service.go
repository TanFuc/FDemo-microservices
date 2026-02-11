package service

import (
	"context"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"microservices/auth/internal/domain/entity"
	"microservices/auth/internal/domain/repository"
	"microservices/auth/internal/infrastructure/cache"
	"microservices/auth/pkg/errors"
	"microservices/auth/pkg/logger"
)

const (
	PermissionsTTL        = time.Hour
	MaxFailedAttempts     = 5
	LockDurationMinutes   = 30
)

type UserResponse struct {
	ID        uuid.UUID `json:"id"`
	Email     string    `json:"email"`
	FullName  string    `json:"fullName"`
	IsActive  bool      `json:"isActive"`
	Roles     []string  `json:"roles"`
	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
}

type CreateUserInput struct {
	Email    string
	Password string
	FullName string
}

type UserService struct {
	userRepo           repository.UserRepository
	roleRepo           repository.RoleRepository
	userRoleRepo       repository.UserRoleRepository
	rolePermissionRepo repository.RolePermissionRepository
	loginHistoryRepo   repository.LoginHistoryRepository
	cache              *cache.RedisClient
}

func NewUserService(
	userRepo repository.UserRepository,
	roleRepo repository.RoleRepository,
	userRoleRepo repository.UserRoleRepository,
	rolePermissionRepo repository.RolePermissionRepository,
	loginHistoryRepo repository.LoginHistoryRepository,
	cache *cache.RedisClient,
) *UserService {
	return &UserService{
		userRepo:           userRepo,
		roleRepo:           roleRepo,
		userRoleRepo:       userRoleRepo,
		rolePermissionRepo: rolePermissionRepo,
		loginHistoryRepo:   loginHistoryRepo,
		cache:              cache,
	}
}

func (s *UserService) CreateUser(ctx context.Context, input *CreateUserInput) (*entity.User, error) {
	// Check if email exists
	exists, err := s.userRepo.ExistsByEmail(ctx, input.Email)
	if err != nil {
		return nil, errors.Wrap(err, "DATABASE_ERROR", "Failed to check email existence", 500)
	}
	if exists {
		return nil, errors.ErrEmailExists
	}

	// Hash password
	passwordHash, err := HashPassword(input.Password)
	if err != nil {
		return nil, errors.Wrap(err, "HASH_ERROR", "Failed to hash password", 500)
	}

	// Create user
	user := &entity.User{
		Email:        input.Email,
		PasswordHash: passwordHash,
		FullName:     input.FullName,
		Status:       entity.UserStatusActive,
	}

	if err := s.userRepo.Create(ctx, user); err != nil {
		return nil, errors.Wrap(err, "DATABASE_ERROR", "Failed to create user", 500)
	}

	// Assign default role (USER)
	defaultRole, err := s.roleRepo.FindDefault(ctx)
	if err != nil {
		if err != gorm.ErrRecordNotFound {
			logger.Error().Err(err).Msg("Failed to find default role")
		}
		// Try to find USER role
		defaultRole, err = s.roleRepo.FindByName(ctx, entity.RoleUser)
		if err != nil {
			logger.Error().Err(err).Msg("Failed to find USER role")
		}
	}

	if defaultRole != nil {
		if err := s.userRoleRepo.AssignRole(ctx, user.ID, defaultRole.ID, nil); err != nil {
			logger.Error().Err(err).Msg("Failed to assign default role")
		}
	}

	return user, nil
}

func (s *UserService) FindByID(ctx context.Context, id uuid.UUID) (*entity.User, error) {
	user, err := s.userRepo.FindByID(ctx, id)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, errors.ErrUserNotFound
		}
		return nil, errors.Wrap(err, "DATABASE_ERROR", "Failed to find user", 500)
	}
	return user, nil
}

func (s *UserService) FindByEmail(ctx context.Context, email string) (*entity.User, error) {
	user, err := s.userRepo.FindByEmail(ctx, email)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, errors.ErrUserNotFound
		}
		return nil, errors.Wrap(err, "DATABASE_ERROR", "Failed to find user", 500)
	}
	return user, nil
}

func (s *UserService) ValidateCredentials(ctx context.Context, email, password string) (*entity.User, error) {
	user, err := s.userRepo.FindByEmailWithPassword(ctx, email)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, errors.ErrInvalidCredentials
		}
		return nil, errors.Wrap(err, "DATABASE_ERROR", "Failed to find user", 500)
	}

	// Check if account is locked
	if user.IsLocked() {
		return nil, errors.ErrAccountLocked
	}

	// Check user status
	switch user.Status {
	case entity.UserStatusInactive:
		return nil, errors.ErrUserInactive
	case entity.UserStatusSuspended:
		return nil, errors.ErrUserSuspended
	case entity.UserStatusBanned:
		return nil, errors.ErrUserBanned
	}

	// Verify password
	if !CheckPassword(password, user.PasswordHash) {
		// Increment failed attempts
		if err := s.userRepo.IncrementFailedAttempts(ctx, user.ID); err != nil {
			logger.Error().Err(err).Msg("Failed to increment failed attempts")
		}

		// Check if account should be locked
		if user.FailedLoginAttempts+1 >= MaxFailedAttempts {
			lockUntil := time.Now().Add(LockDurationMinutes * time.Minute)
			if err := s.userRepo.LockAccount(ctx, user.ID, lockUntil); err != nil {
				logger.Error().Err(err).Msg("Failed to lock account")
			}
		}

		return nil, errors.ErrInvalidCredentials
	}

	return user, nil
}

func (s *UserService) GetUserWithRoles(ctx context.Context, userID uuid.UUID) (*entity.User, error) {
	user, err := s.userRepo.GetUserWithRoles(ctx, userID)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, errors.ErrUserNotFound
		}
		return nil, errors.Wrap(err, "DATABASE_ERROR", "Failed to get user with roles", 500)
	}
	return user, nil
}

func (s *UserService) GetUserPermissions(ctx context.Context, userID uuid.UUID) ([]string, error) {
	return s.rolePermissionRepo.GetUserPermissions(ctx, userID)
}

func (s *UserService) GetUserPermissionsCached(ctx context.Context, userID uuid.UUID) ([]string, error) {
	// Try cache first
	permissions, err := s.cache.GetUserPermissions(ctx, userID.String())
	if err == nil && len(permissions) > 0 {
		return permissions, nil
	}

	// Fetch from database
	permissions, err = s.GetUserPermissions(ctx, userID)
	if err != nil {
		return nil, err
	}

	// Cache permissions
	if len(permissions) > 0 {
		if err := s.cache.CacheUserPermissions(ctx, userID.String(), permissions, PermissionsTTL); err != nil {
			logger.Warn().Err(err).Msg("Failed to cache user permissions")
		}
	}

	return permissions, nil
}

func (s *UserService) InvalidateUserPermissionsCache(ctx context.Context, userID uuid.UUID) error {
	return s.cache.InvalidateUserPermissions(ctx, userID.String())
}

func (s *UserService) HasPermission(ctx context.Context, userID uuid.UUID, permission string) (bool, error) {
	permissions, err := s.GetUserPermissionsCached(ctx, userID)
	if err != nil {
		return false, err
	}

	// Check for wildcard (super admin)
	for _, p := range permissions {
		if p == "*" {
			return true, nil
		}
		if p == permission {
			return true, nil
		}
	}

	return false, nil
}

func (s *UserService) HasAnyPermission(ctx context.Context, userID uuid.UUID, requiredPermissions []string) (bool, error) {
	permissions, err := s.GetUserPermissionsCached(ctx, userID)
	if err != nil {
		return false, err
	}

	permissionSet := make(map[string]bool)
	for _, p := range permissions {
		if p == "*" {
			return true, nil
		}
		permissionSet[p] = true
	}

	for _, required := range requiredPermissions {
		if permissionSet[required] {
			return true, nil
		}
	}

	return false, nil
}

func (s *UserService) AssignRole(ctx context.Context, userID uuid.UUID, roleName string, grantedBy *uuid.UUID) error {
	role, err := s.roleRepo.FindByName(ctx, roleName)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return errors.ErrRoleNotFound
		}
		return errors.Wrap(err, "DATABASE_ERROR", "Failed to find role", 500)
	}

	// Check if role already assigned
	exists, err := s.userRoleRepo.Exists(ctx, userID, role.ID)
	if err != nil {
		return errors.Wrap(err, "DATABASE_ERROR", "Failed to check role assignment", 500)
	}
	if exists {
		return nil // Already assigned
	}

	if err := s.userRoleRepo.AssignRole(ctx, userID, role.ID, grantedBy); err != nil {
		return errors.Wrap(err, "DATABASE_ERROR", "Failed to assign role", 500)
	}

	// Invalidate permissions cache
	if err := s.InvalidateUserPermissionsCache(ctx, userID); err != nil {
		logger.Warn().Err(err).Msg("Failed to invalidate permissions cache")
	}

	return nil
}

func (s *UserService) RemoveRole(ctx context.Context, userID uuid.UUID, roleName string) error {
	role, err := s.roleRepo.FindByName(ctx, roleName)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return errors.ErrRoleNotFound
		}
		return errors.Wrap(err, "DATABASE_ERROR", "Failed to find role", 500)
	}

	if err := s.userRoleRepo.Delete(ctx, userID, role.ID); err != nil {
		return errors.Wrap(err, "DATABASE_ERROR", "Failed to remove role", 500)
	}

	// Invalidate permissions cache
	if err := s.InvalidateUserPermissionsCache(ctx, userID); err != nil {
		logger.Warn().Err(err).Msg("Failed to invalidate permissions cache")
	}

	return nil
}

func (s *UserService) GetUserRoles(ctx context.Context, userID uuid.UUID) ([]string, error) {
	userRoles, err := s.userRoleRepo.FindByUserIDWithRoles(ctx, userID)
	if err != nil {
		return nil, errors.Wrap(err, "DATABASE_ERROR", "Failed to get user roles", 500)
	}

	roles := make([]string, 0, len(userRoles))
	for _, ur := range userRoles {
		if !ur.IsExpired() {
			roles = append(roles, ur.Role.Name)
		}
	}

	return roles, nil
}

func (s *UserService) UpdateLoginInfo(ctx context.Context, userID uuid.UUID, ip string) error {
	return s.userRepo.IncrementLoginCount(ctx, userID, ip)
}

func (s *UserService) RecordLoginHistory(ctx context.Context, userID uuid.UUID, status entity.LoginStatus, authMethod entity.AuthMethod, ipAddress string, userAgent *string) error {
	history := entity.NewLoginHistory(userID, status, authMethod, ipAddress, userAgent)
	return s.loginHistoryRepo.Create(ctx, history)
}

func (s *UserService) ToUserResponse(user *entity.User) *UserResponse {
	return &UserResponse{
		ID:        user.ID,
		Email:     user.Email,
		FullName:  user.FullName,
		IsActive:  user.IsActive(),
		Roles:     user.GetRoleNames(),
		CreatedAt: user.CreatedAt,
		UpdatedAt: user.UpdatedAt,
	}
}
