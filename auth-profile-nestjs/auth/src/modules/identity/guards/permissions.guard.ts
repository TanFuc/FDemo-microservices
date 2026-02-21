import {
  Injectable,
  CanActivate,
  ExecutionContext,
  ForbiddenException,
  Logger,
} from '@nestjs/common';
import { Reflector } from '@nestjs/core';

import { UserService } from '../services/user.service';
import { AuthenticatedUser, AuthenticatedRequest } from '../interfaces';
import { METADATA_KEYS, ERROR_MESSAGES } from '../constants';

@Injectable()
export class PermissionsGuard implements CanActivate {
  private readonly logger = new Logger(PermissionsGuard.name);

  constructor(
    private readonly reflector: Reflector,
    private readonly userService: UserService,
  ) {}

  async canActivate(context: ExecutionContext): Promise<boolean> {
    // Get required permissions from decorator
    const requiredPermissions = this.reflector.get<string[]>(
      METADATA_KEYS.PERMISSIONS,
      context.getHandler(),
    );

    // If no permissions required, allow access
    if (!requiredPermissions || requiredPermissions.length === 0) {
      return true;
    }

    const request = context.switchToHttp().getRequest<AuthenticatedRequest>();
    const user: AuthenticatedUser = request.user;

    if (!user) {
      this.logger.warn('No user found in request');
      throw new ForbiddenException(ERROR_MESSAGES.PERMISSION_DENIED);
    }

    // Check permissions (with caching)
    const hasPermission = await this.userService.hasAnyPermission(user.id, requiredPermissions);

    if (!hasPermission) {
      this.logger.warn(
        `User ${user.id} denied access. Required permissions: ${requiredPermissions.join(', ')}`,
      );
      throw new ForbiddenException(ERROR_MESSAGES.INSUFFICIENT_PERMISSIONS);
    }

    return true;
  }
}
