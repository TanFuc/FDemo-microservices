import {
  CanActivate,
  ExecutionContext,
  Inject,
  Injectable,
  OnModuleInit,
  UnauthorizedException,
  Logger,
} from '@nestjs/common';
import { Reflector } from '@nestjs/core';
import { ClientGrpc } from '@nestjs/microservices';
import { Observable, lastValueFrom } from 'rxjs';
import { PERMISSION_METADATA_KEY, PermissionMetadata } from '../decorators/permission.decorator';

interface AuthService {
  authorize(data: { userId: string; resource: string; action: string }): Observable<{ allowed: boolean }>;
}

@Injectable()
export class GrpcAuthGuard implements CanActivate, OnModuleInit {
  private authService: AuthService;
  private readonly logger = new Logger(GrpcAuthGuard.name);

  constructor(
    @Inject('AUTH_PACKAGE') private client: ClientGrpc,
    private reflector: Reflector,
  ) {}

  onModuleInit() {
    this.authService = this.client.getService<AuthService>('AuthService');
  }

  async canActivate(context: ExecutionContext): Promise<boolean> {
    const permission = this.reflector.getAllAndOverride<PermissionMetadata>(PERMISSION_METADATA_KEY, [
      context.getHandler(),
      context.getClass(),
    ]);

    if (!permission) {
      return true; // No permission required
    }

    const request = context.switchToHttp().getRequest();
    // 1. Try to get User ID from JWT (if JwtAuthGuard is used)
    let userId = request.user?.sub || request.user?.id;

    // 2. Try to get User ID from header (if Gateway handled AuthN)
    if (!userId && request.headers['x-user-id']) {
      userId = request.headers['x-user-id'];
    }

    if (!userId) {
      this.logger.warn('No user ID found in request');
      throw new UnauthorizedException('User not authenticated');
    }

    try {
      const result = await lastValueFrom(
        this.authService.authorize({
          userId,
          resource: permission.resource,
          action: permission.action,
        }),
      );

      if (!result.allowed) {
        this.logger.warn(`User ${userId} denied access to ${permission.resource}:${permission.action}`);
        throw new UnauthorizedException('Permission denied');
      }

      return true;
    } catch (error) {
      this.logger.error(`gRPC Authorization Check Failed: ${error}`);
      throw new UnauthorizedException('Authorization check failed');
    }
  }
}
