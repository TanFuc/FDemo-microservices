import {
  Injectable,
  ConflictException,
  NotFoundException,
  InternalServerErrorException,
  Logger,
} from '@nestjs/common';
import { InjectRepository } from '@nestjs/typeorm';
import { Repository } from 'typeorm';
import * as bcrypt from 'bcrypt';

import { User, Role, UserRole } from '../entities';
import { UserResponse } from '../interfaces';
import { TOKEN_CONFIG, ERROR_MESSAGES, DEFAULT_ROLES } from '../constants';
import { RegisterDto } from '../dto';
import { RedisCacheService } from './redis-cache.service';

@Injectable()
export class UserService {
  private readonly logger = new Logger(UserService.name);

  constructor(
    @InjectRepository(User)
    private readonly userRepository: Repository<User>,
    @InjectRepository(Role)
    private readonly roleRepository: Repository<Role>,
    @InjectRepository(UserRole)
    private readonly userRoleRepository: Repository<UserRole>,
    private readonly redisCacheService: RedisCacheService,
  ) {}

  /**
   * Create a new user
   */
  async createUser(dto: RegisterDto): Promise<User> {
    try {
      // Check if email already exists
      const existingUser = await this.userRepository.findOne({
        where: { email: dto.email.toLowerCase() },
      });

      if (existingUser) {
        throw new ConflictException(ERROR_MESSAGES.EMAIL_EXISTS);
      }

      // Hash password
      const passwordHash = await bcrypt.hash(dto.password, TOKEN_CONFIG.BCRYPT_SALT_ROUNDS);

      // Create user entity
      const user = this.userRepository.create({
        email: dto.email.toLowerCase(),
        passwordHash,
        fullName: dto.fullName,
        status: 'ACTIVE',
      });

      const savedUser = await this.userRepository.save(user);

      // Assign default role 'CUSTOMER'
      await this.assignRole(savedUser.id, DEFAULT_ROLES.CUSTOMER);

      return savedUser;
    } catch (error) {
      if (error instanceof ConflictException) {
        throw error;
      }
      this.logger.error(
        'Failed to create user',
        error instanceof Error ? error.stack : String(error),
      );
      throw new InternalServerErrorException(ERROR_MESSAGES.REGISTRATION_FAILED);
    }
  }

  /**
   * Find user by ID
   */
  async findById(id: string): Promise<User | null> {
    return this.userRepository.findOne({
      where: { id, status: 'ACTIVE' },
    });
  }

  /**
   * Find user by email with password hash (for authentication)
   */
  async findByEmailWithPassword(email: string): Promise<User | null> {
    return this.userRepository
      .createQueryBuilder('user')
      .addSelect('user.passwordHash')
      .where('user.email = :email', { email: email.toLowerCase() })
      .andWhere('user.status = :status', { status: 'ACTIVE' })
      .getOne();
  }

  /**
   * Validate user credentials
   */
  async validateCredentials(email: string, password: string): Promise<User | null> {
    const user = await this.findByEmailWithPassword(email);

    if (!user) {
      return null;
    }

    const isPasswordValid = await bcrypt.compare(password, user.passwordHash);

    if (!isPasswordValid) {
      return null;
    }

    return user;
  }

  /**
   * Get user with roles
   */
  async getUserWithRoles(userId: string): Promise<User | null> {
    return this.userRepository.findOne({
      where: { id: userId, status: 'ACTIVE' },
      relations: ['userRoles', 'userRoles.role'],
    });
  }

  /**
   * Get user permissions from database
   */
  async getUserPermissions(userId: string): Promise<string[]> {
    try {
      const userRoles = await this.userRoleRepository.find({
        where: { userId },
        relations: ['role', 'role.rolePermissions', 'role.rolePermissions.permission'],
      });

      const permissions = new Set<string>();

      for (const userRole of userRoles) {
        // Check for super admin (all permissions)
        if (userRole.role?.name === DEFAULT_ROLES.SUPER_ADMIN) {
          permissions.add('*');
        }

        if (userRole.role?.rolePermissions) {
          for (const rolePermission of userRole.role.rolePermissions) {
            if (rolePermission.permission?.slug) {
              permissions.add(rolePermission.permission.slug);
            }
          }
        }
      }

      return Array.from(permissions);
    } catch (error) {
      this.logger.error(
        `Failed to fetch permissions for user ${userId}`,
        error instanceof Error ? error.stack : String(error),
      );
      return [];
    }
  }

  /**
   * Get user permissions (with caching)
   */
  async getUserPermissionsCached(userId: string): Promise<string[]> {
    // Try cache first
    let permissions = await this.redisCacheService.getUserPermissions(userId);

    if (!permissions) {
      // Cache miss - fetch from DB and cache
      permissions = await this.getUserPermissions(userId);
      await this.redisCacheService.cacheUserPermissions(userId, permissions);
    }

    return permissions;
  }

  /**
   * Assign a role to user
   */
  async assignRole(userId: string, roleName: string): Promise<void> {
    try {
      const role = await this.roleRepository.findOne({
        where: { name: roleName },
      });

      if (!role) {
        this.logger.warn(`Role ${roleName} not found`);
        return;
      }

      // Check if already assigned
      const existing = await this.userRoleRepository.findOne({
        where: { userId, roleId: role.id },
      });

      if (existing) {
        return;
      }

      const userRole = this.userRoleRepository.create({
        userId,
        roleId: role.id,
      });

      await this.userRoleRepository.save(userRole);

      // Invalidate permissions cache
      await this.redisCacheService.invalidateUserPermissions(userId);
    } catch (error) {
      this.logger.error(
        `Failed to assign role ${roleName} to user ${userId}`,
        error instanceof Error ? error.stack : String(error),
      );
    }
  }

  /**
   * Remove a role from user
   */
  async removeRole(userId: string, roleName: string): Promise<void> {
    try {
      const role = await this.roleRepository.findOne({
        where: { name: roleName },
      });

      if (!role) {
        return;
      }

      await this.userRoleRepository.delete({ userId, roleId: role.id });

      // Invalidate permissions cache
      await this.redisCacheService.invalidateUserPermissions(userId);
    } catch (error) {
      this.logger.error(
        `Failed to remove role ${roleName} from user ${userId}`,
        error instanceof Error ? error.stack : String(error),
      );
    }
  }

  /**
   * Get user roles as string array
   */
  async getUserRoles(userId: string): Promise<string[]> {
    const userRoles = await this.userRoleRepository.find({
      where: { userId },
      relations: ['role'],
    });

    return userRoles.map((ur) => ur.role.name);
  }

  /**
   * Update user profile
   */
  async updateProfile(userId: string, updates: { fullName?: string }): Promise<User> {
    const user = await this.findById(userId);

    if (!user) {
      throw new NotFoundException(ERROR_MESSAGES.USER_NOT_FOUND);
    }

    if (updates.fullName) {
      user.fullName = updates.fullName;
    }

    return this.userRepository.save(user);
  }

  /**
   * Deactivate user
   */
  async deactivateUser(userId: string): Promise<void> {
    await this.userRepository.update({ id: userId }, { status: 'INACTIVE' });
    await this.redisCacheService.invalidateUserPermissions(userId);
  }

  /**
   * Transform User entity to UserResponse
   */
  async toUserResponse(user: User): Promise<UserResponse> {
    const roles = await this.getUserRoles(user.id);

    return {
      id: user.id,
      email: user.email,
      fullName: user.fullName,
      isActive: user.status === 'ACTIVE',
      roles,
      createdAt: user.createdAt,
      updatedAt: user.updatedAt,
    };
  }

  /**
   * Check if user has specific permission
   */
  async hasPermission(userId: string, permission: string): Promise<boolean> {
    const permissions = await this.getUserPermissionsCached(userId);

    // Super admin has all permissions
    if (permissions.includes('*')) {
      return true;
    }

    return permissions.includes(permission);
  }

  /**
   * Check if user has any of the specified permissions
   */
  async hasAnyPermission(userId: string, requiredPermissions: string[]): Promise<boolean> {
    const permissions = await this.getUserPermissionsCached(userId);

    // Super admin has all permissions
    if (permissions.includes('*')) {
      return true;
    }

    return requiredPermissions.some((p) => permissions.includes(p));
  }
}
