import { Injectable, Logger, OnModuleInit } from '@nestjs/common';
import { InjectRepository } from '@nestjs/typeorm';
import { Repository } from 'typeorm';

import { Role, Permission, RolePermission } from '../entities';
import {
  ROLE_DEFINITIONS,
  PERMISSION_DEFINITIONS,
  ROLE_PERMISSION_MAPPING,
  DEFAULT_ROLES,
} from '../constants';

@Injectable()
export class SeedingService implements OnModuleInit {
  private readonly logger = new Logger(SeedingService.name);

  constructor(
    @InjectRepository(Role)
    private readonly roleRepository: Repository<Role>,
    @InjectRepository(Permission)
    private readonly permissionRepository: Repository<Permission>,
    @InjectRepository(RolePermission)
    private readonly rolePermissionRepository: Repository<RolePermission>,
  ) {}

  async onModuleInit(): Promise<void> {
    await this.seed();
  }

  async seed(): Promise<void> {
    this.logger.log('Starting database seeding...');

    try {
      await this.seedPermissions();
      await this.seedRoles();
      await this.seedRolePermissions();

      this.logger.log('Database seeding completed successfully');
    } catch (error) {
      this.logger.error(
        'Database seeding failed',
        error instanceof Error ? error.stack : String(error),
      );
    }
  }

  private async seedPermissions(): Promise<void> {
    let created = 0;
    let skipped = 0;

    for (const perm of PERMISSION_DEFINITIONS) {
      const slug = `${perm.resource}:${perm.action}`;
      const exists = await this.permissionRepository.findOne({
        where: { slug },
      });

      if (!exists) {
        const permission = this.permissionRepository.create({
          resource: perm.resource,
          action: perm.action,
          slug,
          displayName: perm.displayName,
          category: perm.category,
          isDangerous: 'isDangerous' in perm ? perm.isDangerous : false,
          description: `Permission to ${perm.action} ${perm.resource}`,
        });
        await this.permissionRepository.save(permission);
        created++;
      } else {
        skipped++;
      }
    }

    this.logger.log(`Permissions: ${created} created, ${skipped} already exist`);
  }

  private async seedRoles(): Promise<void> {
    let created = 0;
    let skipped = 0;

    for (const roleData of ROLE_DEFINITIONS) {
      const exists = await this.roleRepository.findOne({
        where: { name: roleData.name },
      });

      if (!exists) {
        const role = this.roleRepository.create({
          name: roleData.name,
          displayName: roleData.displayName,
          description: `${roleData.displayName} role`,
          isSystem: roleData.isSystem,
          isDefault: 'isDefault' in roleData ? roleData.isDefault : false,
          priority: roleData.priority,
        });
        await this.roleRepository.save(role);
        created++;
      } else {
        skipped++;
      }
    }

    this.logger.log(`Roles: ${created} created, ${skipped} already exist`);
  }

  private async seedRolePermissions(): Promise<void> {
    let created = 0;
    let skipped = 0;

    for (const [roleName, permissionSlugs] of Object.entries(ROLE_PERMISSION_MAPPING)) {
      const role = await this.roleRepository.findOne({
        where: { name: roleName },
      });

      if (!role) {
        this.logger.warn(`Role not found: ${roleName}`);
        continue;
      }

      // Handle super admin with all permissions
      if (roleName === DEFAULT_ROLES.SUPER_ADMIN && permissionSlugs.includes('*')) {
        const allPermissions = await this.permissionRepository.find();
        for (const permission of allPermissions) {
          const exists = await this.rolePermissionRepository.findOne({
            where: { roleId: role.id, permissionId: permission.id },
          });

          if (!exists) {
            const rolePermission = this.rolePermissionRepository.create({
              roleId: role.id,
              permissionId: permission.id,
            });
            await this.rolePermissionRepository.save(rolePermission);
            created++;
          } else {
            skipped++;
          }
        }
        continue;
      }

      for (const slug of permissionSlugs) {
        if (slug === '*') continue;

        const permission = await this.permissionRepository.findOne({
          where: { slug },
        });

        if (!permission) {
          this.logger.warn(`Permission not found: ${slug}`);
          continue;
        }

        const exists = await this.rolePermissionRepository.findOne({
          where: { roleId: role.id, permissionId: permission.id },
        });

        if (!exists) {
          const rolePermission = this.rolePermissionRepository.create({
            roleId: role.id,
            permissionId: permission.id,
          });
          await this.rolePermissionRepository.save(rolePermission);
          created++;
        } else {
          skipped++;
        }
      }
    }

    this.logger.log(`Role-Permissions: ${created} created, ${skipped} already exist`);
  }

  /**
   * Reset all seeded data (for testing)
   */
  async reset(): Promise<void> {
    this.logger.warn('Resetting seeded data...');

    await this.rolePermissionRepository.delete({});
    await this.permissionRepository.delete({});
    await this.roleRepository.delete({ isSystem: true });

    this.logger.log('Seeded data reset complete');
  }

  /**
   * Resync permissions for a specific role
   */
  async resyncRolePermissions(roleName: string): Promise<void> {
    const role = await this.roleRepository.findOne({
      where: { name: roleName },
    });

    if (!role) {
      throw new Error(`Role not found: ${roleName}`);
    }

    const permissionSlugs = ROLE_PERMISSION_MAPPING[roleName];
    if (!permissionSlugs) {
      throw new Error(`No permission mapping found for role: ${roleName}`);
    }

    // Remove existing permissions
    await this.rolePermissionRepository.delete({ roleId: role.id });

    // Re-add permissions
    for (const slug of permissionSlugs) {
      if (slug === '*') {
        const allPermissions = await this.permissionRepository.find();
        for (const permission of allPermissions) {
          const rolePermission = this.rolePermissionRepository.create({
            roleId: role.id,
            permissionId: permission.id,
          });
          await this.rolePermissionRepository.save(rolePermission);
        }
      } else {
        const permission = await this.permissionRepository.findOne({
          where: { slug },
        });

        if (permission) {
          const rolePermission = this.rolePermissionRepository.create({
            roleId: role.id,
            permissionId: permission.id,
          });
          await this.rolePermissionRepository.save(rolePermission);
        }
      }
    }

    this.logger.log(`Resynced permissions for role: ${roleName}`);
  }
}
