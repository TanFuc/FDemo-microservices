package postgres

import (
	"context"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"microservices/auth/internal/model"
	"microservices/auth/internal/repository"
)

type userRoleRepository struct {
	db *gorm.DB
}

func NewUserRoleRepository(db *gorm.DB) repository.UserRoleRepository {
	return &userRoleRepository{db: db}
}

func (r *userRoleRepository) Create(ctx context.Context, userRole *model.UserRole) error {
	return r.db.WithContext(ctx).Create(userRole).Error
}

func (r *userRoleRepository) FindByUserID(ctx context.Context, userID uuid.UUID) ([]model.UserRole, error) {
	var userRoles []model.UserRole
	err := r.db.WithContext(ctx).
		Where("user_id = ?", userID).
		Find(&userRoles).Error
	return userRoles, err
}

func (r *userRoleRepository) FindByUserIDWithRoles(ctx context.Context, userID uuid.UUID) ([]model.UserRole, error) {
	var userRoles []model.UserRole
	err := r.db.WithContext(ctx).
		Preload("Role").
		Where("user_id = ?", userID).
		Find(&userRoles).Error
	return userRoles, err
}

func (r *userRoleRepository) Delete(ctx context.Context, userID uuid.UUID, roleID int64) error {
	return r.db.WithContext(ctx).
		Where("user_id = ? AND role_id = ?", userID, roleID).
		Delete(&model.UserRole{}).Error
}

func (r *userRoleRepository) DeleteByUserID(ctx context.Context, userID uuid.UUID) error {
	return r.db.WithContext(ctx).
		Where("user_id = ?", userID).
		Delete(&model.UserRole{}).Error
}

func (r *userRoleRepository) Exists(ctx context.Context, userID uuid.UUID, roleID int64) (bool, error) {
	var count int64
	err := r.db.WithContext(ctx).
		Model(&model.UserRole{}).
		Where("user_id = ? AND role_id = ?", userID, roleID).
		Count(&count).Error
	return count > 0, err
}

func (r *userRoleRepository) AssignRole(ctx context.Context, userID uuid.UUID, roleID int64, grantedBy *uuid.UUID) error {
	userRole := &model.UserRole{
		UserID:    userID,
		RoleID:    roleID,
		GrantedBy: grantedBy,
	}
	return r.db.WithContext(ctx).Create(userRole).Error
}
