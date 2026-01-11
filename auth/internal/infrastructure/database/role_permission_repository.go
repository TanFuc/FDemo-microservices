package database

import (
	"context"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"microservices/auth/internal/domain/entity"
	"microservices/auth/internal/domain/repository"
)

type rolePermissionRepository struct {
	db *gorm.DB
}

func NewRolePermissionRepository(db *gorm.DB) repository.RolePermissionRepository {
	return &rolePermissionRepository{db: db}
}

func (r *rolePermissionRepository) Create(ctx context.Context, rp *entity.RolePermission) error {
	return r.db.WithContext(ctx).Create(rp).Error
}

func (r *rolePermissionRepository) FindByRoleID(ctx context.Context, roleID int64) ([]entity.RolePermission, error) {
	var rps []entity.RolePermission
	err := r.db.WithContext(ctx).
		Where("role_id = ?", roleID).
		Find(&rps).Error
	return rps, err
}

func (r *rolePermissionRepository) FindByRoleIDWithPermissions(ctx context.Context, roleID int64) ([]entity.RolePermission, error) {
	var rps []entity.RolePermission
	err := r.db.WithContext(ctx).
		Preload("Permission").
		Where("role_id = ?", roleID).
		Find(&rps).Error
	return rps, err
}

func (r *rolePermissionRepository) Delete(ctx context.Context, roleID, permissionID int64) error {
	return r.db.WithContext(ctx).
		Where("role_id = ? AND permission_id = ?", roleID, permissionID).
		Delete(&entity.RolePermission{}).Error
}

func (r *rolePermissionRepository) DeleteByRoleID(ctx context.Context, roleID int64) error {
	return r.db.WithContext(ctx).
		Where("role_id = ?", roleID).
		Delete(&entity.RolePermission{}).Error
}

func (r *rolePermissionRepository) Exists(ctx context.Context, roleID, permissionID int64) (bool, error) {
	var count int64
	err := r.db.WithContext(ctx).
		Model(&entity.RolePermission{}).
		Where("role_id = ? AND permission_id = ?", roleID, permissionID).
		Count(&count).Error
	return count > 0, err
}

func (r *rolePermissionRepository) GetUserPermissions(ctx context.Context, userID uuid.UUID) ([]string, error) {
	var permissions []string

	// First, get all roles for the user
	var userRoles []entity.UserRole
	err := r.db.WithContext(ctx).
		Preload("Role").
		Where("user_id = ?", userID).
		Where("expires_at IS NULL OR expires_at > NOW()").
		Find(&userRoles).Error
	if err != nil {
		return nil, err
	}

	// Check if user has SUPER_ADMIN role
	for _, ur := range userRoles {
		if ur.Role.Name == entity.RoleSuperAdmin {
			return []string{"*"}, nil
		}
	}

	// Get role IDs
	roleIDs := make([]int64, len(userRoles))
	for i, ur := range userRoles {
		roleIDs[i] = ur.RoleID
	}

	if len(roleIDs) == 0 {
		return []string{}, nil
	}

	// Get all permissions for these roles
	err = r.db.WithContext(ctx).
		Table("role_permissions").
		Select("DISTINCT permissions.slug").
		Joins("JOIN permissions ON permissions.id = role_permissions.permission_id").
		Where("role_permissions.role_id IN ?", roleIDs).
		Pluck("slug", &permissions).Error

	if err != nil {
		return nil, err
	}

	return permissions, nil
}
