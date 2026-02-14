package repository

import (
	"context"

	"github.com/google/uuid"

	"microservices/auth/internal/model"
)

type UserRepository interface {
	Create(ctx context.Context, user *model.User) error
	FindByID(ctx context.Context, id uuid.UUID) (*model.User, error)
	FindByEmail(ctx context.Context, email string) (*model.User, error)
	FindByEmailWithPassword(ctx context.Context, email string) (*model.User, error)
	FindByPhone(ctx context.Context, phone string) (*model.User, error)
	Update(ctx context.Context, user *model.User) error
	Delete(ctx context.Context, id uuid.UUID) error
	ExistsByEmail(ctx context.Context, email string) (bool, error)
	ExistsByPhone(ctx context.Context, phone string) (bool, error)
	GetUserWithRoles(ctx context.Context, userID uuid.UUID) (*model.User, error)
	IncrementLoginCount(ctx context.Context, userID uuid.UUID, ip string) error
	IncrementFailedAttempts(ctx context.Context, userID uuid.UUID) error
	ResetFailedAttempts(ctx context.Context, userID uuid.UUID) error
	LockAccount(ctx context.Context, userID uuid.UUID, until interface{}) error
}

type RoleRepository interface {
	Create(ctx context.Context, role *model.Role) error
	FindByID(ctx context.Context, id int64) (*model.Role, error)
	FindByName(ctx context.Context, name string) (*model.Role, error)
	FindAll(ctx context.Context) ([]model.Role, error)
	FindDefault(ctx context.Context) (*model.Role, error)
	Update(ctx context.Context, role *model.Role) error
	Delete(ctx context.Context, id int64) error
	ExistsByName(ctx context.Context, name string) (bool, error)
	Upsert(ctx context.Context, role *model.Role) error
}

type PermissionRepository interface {
	Create(ctx context.Context, permission *model.Permission) error
	FindByID(ctx context.Context, id int64) (*model.Permission, error)
	FindBySlug(ctx context.Context, slug string) (*model.Permission, error)
	FindAll(ctx context.Context) ([]model.Permission, error)
	FindByCategory(ctx context.Context, category string) ([]model.Permission, error)
	Update(ctx context.Context, permission *model.Permission) error
	Delete(ctx context.Context, id int64) error
	ExistsBySlug(ctx context.Context, slug string) (bool, error)
	Upsert(ctx context.Context, permission *model.Permission) error
}

type UserRoleRepository interface {
	Create(ctx context.Context, userRole *model.UserRole) error
	FindByUserID(ctx context.Context, userID uuid.UUID) ([]model.UserRole, error)
	FindByUserIDWithRoles(ctx context.Context, userID uuid.UUID) ([]model.UserRole, error)
	Delete(ctx context.Context, userID uuid.UUID, roleID int64) error
	DeleteByUserID(ctx context.Context, userID uuid.UUID) error
	Exists(ctx context.Context, userID uuid.UUID, roleID int64) (bool, error)
	AssignRole(ctx context.Context, userID uuid.UUID, roleID int64, grantedBy *uuid.UUID) error
}

type RolePermissionRepository interface {
	Create(ctx context.Context, rp *model.RolePermission) error
	FindByRoleID(ctx context.Context, roleID int64) ([]model.RolePermission, error)
	FindByRoleIDWithPermissions(ctx context.Context, roleID int64) ([]model.RolePermission, error)
	Delete(ctx context.Context, roleID, permissionID int64) error
	DeleteByRoleID(ctx context.Context, roleID int64) error
	Exists(ctx context.Context, roleID, permissionID int64) (bool, error)
	GetUserPermissions(ctx context.Context, userID uuid.UUID) ([]string, error)
}

type RefreshTokenRepository interface {
	Create(ctx context.Context, token *model.RefreshToken) error
	FindByID(ctx context.Context, id uuid.UUID) (*model.RefreshToken, error)
	FindByTokenHash(ctx context.Context, tokenHash string) (*model.RefreshToken, error)
	FindByUserID(ctx context.Context, userID uuid.UUID) ([]model.RefreshToken, error)
	FindActiveByUserID(ctx context.Context, userID uuid.UUID) ([]model.RefreshToken, error)
	Update(ctx context.Context, token *model.RefreshToken) error
	Revoke(ctx context.Context, id uuid.UUID, reason string) error
	RevokeAllByUserID(ctx context.Context, userID uuid.UUID, reason string) error
	RevokeByDeviceID(ctx context.Context, userID uuid.UUID, deviceID string, reason string) error
	DeleteExpired(ctx context.Context) (int64, error)
}

type LoginHistoryRepository interface {
	Create(ctx context.Context, history *model.LoginHistory) error
	FindByUserID(ctx context.Context, userID uuid.UUID, limit int) ([]model.LoginHistory, error)
	FindRecentByUserID(ctx context.Context, userID uuid.UUID, since interface{}) ([]model.LoginHistory, error)
	CountFailedAttempts(ctx context.Context, userID uuid.UUID, since interface{}) (int64, error)
}

type PasswordResetRepository interface {
	Create(ctx context.Context, reset *model.PasswordReset) error
	FindByID(ctx context.Context, id uuid.UUID) (*model.PasswordReset, error)
	FindByTokenHash(ctx context.Context, tokenHash string) (*model.PasswordReset, error)
	FindActiveByUserID(ctx context.Context, userID uuid.UUID) (*model.PasswordReset, error)
	MarkUsed(ctx context.Context, id uuid.UUID) error
	DeleteExpired(ctx context.Context) (int64, error)
}
