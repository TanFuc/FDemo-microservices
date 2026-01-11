package service

import (
	"context"

	"tafu-auth/internal/domain/entity"
	"tafu-auth/internal/domain/repository"
	"tafu-auth/pkg/logger"
)

type SeedingService struct {
	roleRepo           repository.RoleRepository
	permissionRepo     repository.PermissionRepository
	rolePermissionRepo repository.RolePermissionRepository
}

func NewSeedingService(
	roleRepo repository.RoleRepository,
	permissionRepo repository.PermissionRepository,
	rolePermissionRepo repository.RolePermissionRepository,
) *SeedingService {
	return &SeedingService{
		roleRepo:           roleRepo,
		permissionRepo:     permissionRepo,
		rolePermissionRepo: rolePermissionRepo,
	}
}

func (s *SeedingService) SeedAll(ctx context.Context) error {
	logger.Info().Msg("Starting database seeding...")

	if err := s.SeedRoles(ctx); err != nil {
		return err
	}

	if err := s.SeedPermissions(ctx); err != nil {
		return err
	}

	if err := s.SeedRolePermissions(ctx); err != nil {
		return err
	}

	logger.Info().Msg("Database seeding completed")
	return nil
}

func (s *SeedingService) SeedRoles(ctx context.Context) error {
	logger.Info().Msg("Seeding roles...")

	for _, role := range entity.DefaultRoles {
		if err := s.roleRepo.Upsert(ctx, &role); err != nil {
			logger.Error().Err(err).Str("role", role.Name).Msg("Failed to seed role")
			return err
		}
	}

	logger.Info().Int("count", len(entity.DefaultRoles)).Msg("Roles seeded successfully")
	return nil
}

func (s *SeedingService) SeedPermissions(ctx context.Context) error {
	logger.Info().Msg("Seeding permissions...")

	for _, permission := range entity.DefaultPermissions {
		if err := s.permissionRepo.Upsert(ctx, &permission); err != nil {
			logger.Error().Err(err).Str("permission", permission.Slug).Msg("Failed to seed permission")
			return err
		}
	}

	logger.Info().Int("count", len(entity.DefaultPermissions)).Msg("Permissions seeded successfully")
	return nil
}

func (s *SeedingService) SeedRolePermissions(ctx context.Context) error {
	logger.Info().Msg("Seeding role-permission mappings...")

	for roleName, permissionSlugs := range entity.RolePermissionMapping {
		// Skip SUPER_ADMIN as it has wildcard permissions
		if roleName == entity.RoleSuperAdmin {
			continue
		}

		role, err := s.roleRepo.FindByName(ctx, roleName)
		if err != nil {
			logger.Warn().Err(err).Str("role", roleName).Msg("Role not found, skipping")
			continue
		}

		for _, slug := range permissionSlugs {
			permission, err := s.permissionRepo.FindBySlug(ctx, slug)
			if err != nil {
				logger.Warn().Err(err).Str("permission", slug).Msg("Permission not found, skipping")
				continue
			}

			// Check if mapping exists
			exists, err := s.rolePermissionRepo.Exists(ctx, role.ID, permission.ID)
			if err != nil {
				logger.Warn().Err(err).Msg("Failed to check role-permission existence")
				continue
			}

			if !exists {
				rp := &entity.RolePermission{
					RoleID:       role.ID,
					PermissionID: permission.ID,
				}
				if err := s.rolePermissionRepo.Create(ctx, rp); err != nil {
					logger.Warn().Err(err).
						Str("role", roleName).
						Str("permission", slug).
						Msg("Failed to create role-permission mapping")
				}
			}
		}
	}

	logger.Info().Msg("Role-permission mappings seeded successfully")
	return nil
}

func (s *SeedingService) ResyncRolePermissions(ctx context.Context, roleName string) error {
	role, err := s.roleRepo.FindByName(ctx, roleName)
	if err != nil {
		return err
	}

	// Delete existing permissions
	if err := s.rolePermissionRepo.DeleteByRoleID(ctx, role.ID); err != nil {
		return err
	}

	// Re-add permissions
	permissionSlugs, ok := entity.RolePermissionMapping[roleName]
	if !ok {
		return nil
	}

	for _, slug := range permissionSlugs {
		permission, err := s.permissionRepo.FindBySlug(ctx, slug)
		if err != nil {
			continue
		}

		rp := &entity.RolePermission{
			RoleID:       role.ID,
			PermissionID: permission.ID,
		}
		if err := s.rolePermissionRepo.Create(ctx, rp); err != nil {
			logger.Warn().Err(err).
				Str("role", roleName).
				Str("permission", slug).
				Msg("Failed to create role-permission mapping")
		}
	}

	return nil
}

func (s *SeedingService) ResetAllSeededData(ctx context.Context) error {
	logger.Warn().Msg("Resetting all seeded data...")

	// Delete role permissions first (foreign key constraint)
	for _, role := range entity.DefaultRoles {
		r, err := s.roleRepo.FindByName(ctx, role.Name)
		if err != nil {
			continue
		}
		if err := s.rolePermissionRepo.DeleteByRoleID(ctx, r.ID); err != nil {
			logger.Warn().Err(err).Str("role", role.Name).Msg("Failed to delete role permissions")
		}
	}

	// Delete permissions
	for _, permission := range entity.DefaultPermissions {
		p, err := s.permissionRepo.FindBySlug(ctx, permission.Slug)
		if err != nil {
			continue
		}
		if err := s.permissionRepo.Delete(ctx, p.ID); err != nil {
			logger.Warn().Err(err).Str("permission", permission.Slug).Msg("Failed to delete permission")
		}
	}

	// Delete roles
	for _, role := range entity.DefaultRoles {
		r, err := s.roleRepo.FindByName(ctx, role.Name)
		if err != nil {
			continue
		}
		if err := s.roleRepo.Delete(ctx, r.ID); err != nil {
			logger.Warn().Err(err).Str("role", role.Name).Msg("Failed to delete role")
		}
	}

	logger.Info().Msg("Seeded data reset completed")
	return nil
}
