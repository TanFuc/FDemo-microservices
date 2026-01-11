package repository

import (
	"context"

	"github.com/google/uuid"

	"tafu-auth/internal/domain/entity"
)

type UserRepository interface {
	Create(ctx context.Context, user *entity.User) error
	FindByID(ctx context.Context, id uuid.UUID) (*entity.User, error)
	FindByEmail(ctx context.Context, email string) (*entity.User, error)
	FindByEmailWithPassword(ctx context.Context, email string) (*entity.User, error)
	FindByPhone(ctx context.Context, phone string) (*entity.User, error)
	Update(ctx context.Context, user *entity.User) error
	Delete(ctx context.Context, id uuid.UUID) error
	ExistsByEmail(ctx context.Context, email string) (bool, error)
	ExistsByPhone(ctx context.Context, phone string) (bool, error)
	GetUserWithRoles(ctx context.Context, userID uuid.UUID) (*entity.User, error)
	IncrementLoginCount(ctx context.Context, userID uuid.UUID, ip string) error
	IncrementFailedAttempts(ctx context.Context, userID uuid.UUID) error
	ResetFailedAttempts(ctx context.Context, userID uuid.UUID) error
	LockAccount(ctx context.Context, userID uuid.UUID, until interface{}) error
}

type RoleRepository interface {
	Create(ctx context.Context, role *entity.Role) error
	FindByID(ctx context.Context, id int64) (*entity.Role, error)
	FindByName(ctx context.Context, name string) (*entity.Role, error)
	FindAll(ctx context.Context) ([]entity.Role, error)
	FindDefault(ctx context.Context) (*entity.Role, error)
	Update(ctx context.Context, role *entity.Role) error
	Delete(ctx context.Context, id int64) error
	ExistsByName(ctx context.Context, name string) (bool, error)
	Upsert(ctx context.Context, role *entity.Role) error
}

type PermissionRepository interface {
	Create(ctx context.Context, permission *entity.Permission) error
	FindByID(ctx context.Context, id int64) (*entity.Permission, error)
	FindBySlug(ctx context.Context, slug string) (*entity.Permission, error)
	FindAll(ctx context.Context) ([]entity.Permission, error)
	FindByCategory(ctx context.Context, category string) ([]entity.Permission, error)
	Update(ctx context.Context, permission *entity.Permission) error
	Delete(ctx context.Context, id int64) error
	ExistsBySlug(ctx context.Context, slug string) (bool, error)
	Upsert(ctx context.Context, permission *entity.Permission) error
}

type UserRoleRepository interface {
	Create(ctx context.Context, userRole *entity.UserRole) error
	FindByUserID(ctx context.Context, userID uuid.UUID) ([]entity.UserRole, error)
	FindByUserIDWithRoles(ctx context.Context, userID uuid.UUID) ([]entity.UserRole, error)
	Delete(ctx context.Context, userID uuid.UUID, roleID int64) error
	DeleteByUserID(ctx context.Context, userID uuid.UUID) error
	Exists(ctx context.Context, userID uuid.UUID, roleID int64) (bool, error)
	AssignRole(ctx context.Context, userID uuid.UUID, roleID int64, grantedBy *uuid.UUID) error
}

type RolePermissionRepository interface {
	Create(ctx context.Context, rp *entity.RolePermission) error
	FindByRoleID(ctx context.Context, roleID int64) ([]entity.RolePermission, error)
	FindByRoleIDWithPermissions(ctx context.Context, roleID int64) ([]entity.RolePermission, error)
	Delete(ctx context.Context, roleID, permissionID int64) error
	DeleteByRoleID(ctx context.Context, roleID int64) error
	Exists(ctx context.Context, roleID, permissionID int64) (bool, error)
	GetUserPermissions(ctx context.Context, userID uuid.UUID) ([]string, error)
}

type RefreshTokenRepository interface {
	Create(ctx context.Context, token *entity.RefreshToken) error
	FindByID(ctx context.Context, id uuid.UUID) (*entity.RefreshToken, error)
	FindByTokenHash(ctx context.Context, tokenHash string) (*entity.RefreshToken, error)
	FindByUserID(ctx context.Context, userID uuid.UUID) ([]entity.RefreshToken, error)
	FindActiveByUserID(ctx context.Context, userID uuid.UUID) ([]entity.RefreshToken, error)
	Update(ctx context.Context, token *entity.RefreshToken) error
	Revoke(ctx context.Context, id uuid.UUID, reason string) error
	RevokeAllByUserID(ctx context.Context, userID uuid.UUID, reason string) error
	RevokeByDeviceID(ctx context.Context, userID uuid.UUID, deviceID string, reason string) error
	DeleteExpired(ctx context.Context) (int64, error)
}

type LoginHistoryRepository interface {
	Create(ctx context.Context, history *entity.LoginHistory) error
	FindByUserID(ctx context.Context, userID uuid.UUID, limit int) ([]entity.LoginHistory, error)
	FindRecentByUserID(ctx context.Context, userID uuid.UUID, since interface{}) ([]entity.LoginHistory, error)
	CountFailedAttempts(ctx context.Context, userID uuid.UUID, since interface{}) (int64, error)
}

type PasswordResetRepository interface {
	Create(ctx context.Context, reset *entity.PasswordReset) error
	FindByID(ctx context.Context, id uuid.UUID) (*entity.PasswordReset, error)
	FindByTokenHash(ctx context.Context, tokenHash string) (*entity.PasswordReset, error)
	FindActiveByUserID(ctx context.Context, userID uuid.UUID) (*entity.PasswordReset, error)
	MarkUsed(ctx context.Context, id uuid.UUID) error
	DeleteExpired(ctx context.Context) (int64, error)
}
