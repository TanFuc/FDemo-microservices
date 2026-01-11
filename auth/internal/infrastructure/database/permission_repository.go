package database

import (
	"context"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"tafu-auth/internal/domain/entity"
	"tafu-auth/internal/domain/repository"
)

type permissionRepository struct {
	db *gorm.DB
}

func NewPermissionRepository(db *gorm.DB) repository.PermissionRepository {
	return &permissionRepository{db: db}
}

func (r *permissionRepository) Create(ctx context.Context, permission *entity.Permission) error {
	return r.db.WithContext(ctx).Create(permission).Error
}

func (r *permissionRepository) FindByID(ctx context.Context, id int64) (*entity.Permission, error) {
	var permission entity.Permission
	err := r.db.WithContext(ctx).First(&permission, id).Error
	if err != nil {
		return nil, err
	}
	return &permission, nil
}

func (r *permissionRepository) FindBySlug(ctx context.Context, slug string) (*entity.Permission, error) {
	var permission entity.Permission
	err := r.db.WithContext(ctx).
		Where("slug = ?", slug).
		First(&permission).Error
	if err != nil {
		return nil, err
	}
	return &permission, nil
}

func (r *permissionRepository) FindAll(ctx context.Context) ([]entity.Permission, error) {
	var permissions []entity.Permission
	err := r.db.WithContext(ctx).
		Order("category, resource, action").
		Find(&permissions).Error
	return permissions, err
}

func (r *permissionRepository) FindByCategory(ctx context.Context, category string) ([]entity.Permission, error) {
	var permissions []entity.Permission
	err := r.db.WithContext(ctx).
		Where("category = ?", category).
		Order("resource, action").
		Find(&permissions).Error
	return permissions, err
}

func (r *permissionRepository) Update(ctx context.Context, permission *entity.Permission) error {
	return r.db.WithContext(ctx).Save(permission).Error
}

func (r *permissionRepository) Delete(ctx context.Context, id int64) error {
	return r.db.WithContext(ctx).Delete(&entity.Permission{}, id).Error
}

func (r *permissionRepository) ExistsBySlug(ctx context.Context, slug string) (bool, error) {
	var count int64
	err := r.db.WithContext(ctx).
		Model(&entity.Permission{}).
		Where("slug = ?", slug).
		Count(&count).Error
	return count > 0, err
}

func (r *permissionRepository) Upsert(ctx context.Context, permission *entity.Permission) error {
	return r.db.WithContext(ctx).
		Clauses(clause.OnConflict{
			Columns:   []clause.Column{{Name: "slug"}},
			DoUpdates: clause.AssignmentColumns([]string{"display_name", "description", "category", "is_dangerous"}),
		}).
		Create(permission).Error
}
