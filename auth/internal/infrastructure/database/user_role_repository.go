package database

import (
	"context"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"microservices/auth/internal/domain/entity"
	"microservices/auth/internal/domain/repository"
)

type userRoleRepository struct {
	db *gorm.DB
}

func NewUserRoleRepository(db *gorm.DB) repository.UserRoleRepository {
	return &userRoleRepository{db: db}
}

func (r *userRoleRepository) Create(ctx context.Context, userRole *entity.UserRole) error {
	return r.db.WithContext(ctx).Create(userRole).Error
}

func (r *userRoleRepository) FindByUserID(ctx context.Context, userID uuid.UUID) ([]entity.UserRole, error) {
	var userRoles []entity.UserRole
	err := r.db.WithContext(ctx).
		Where("user_id = ?", userID).
		Find(&userRoles).Error
	return userRoles, err
}

func (r *userRoleRepository) FindByUserIDWithRoles(ctx context.Context, userID uuid.UUID) ([]entity.UserRole, error) {
	var userRoles []entity.UserRole
	err := r.db.WithContext(ctx).
		Preload("Role").
		Where("user_id = ?", userID).
		Find(&userRoles).Error
	return userRoles, err
}

func (r *userRoleRepository) Delete(ctx context.Context, userID uuid.UUID, roleID int64) error {
	return r.db.WithContext(ctx).
		Where("user_id = ? AND role_id = ?", userID, roleID).
		Delete(&entity.UserRole{}).Error
}

func (r *userRoleRepository) DeleteByUserID(ctx context.Context, userID uuid.UUID) error {
	return r.db.WithContext(ctx).
		Where("user_id = ?", userID).
		Delete(&entity.UserRole{}).Error
}

func (r *userRoleRepository) Exists(ctx context.Context, userID uuid.UUID, roleID int64) (bool, error) {
	var count int64
	err := r.db.WithContext(ctx).
		Model(&entity.UserRole{}).
		Where("user_id = ? AND role_id = ?", userID, roleID).
		Count(&count).Error
	return count > 0, err
}

func (r *userRoleRepository) AssignRole(ctx context.Context, userID uuid.UUID, roleID int64, grantedBy *uuid.UUID) error {
	userRole := &entity.UserRole{
		UserID:    userID,
		RoleID:    roleID,
		GrantedBy: grantedBy,
	}
	return r.db.WithContext(ctx).Create(userRole).Error
}
