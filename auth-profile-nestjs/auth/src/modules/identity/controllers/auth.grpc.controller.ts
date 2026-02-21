import { Controller, Logger } from '@nestjs/common';
import { GrpcMethod } from '@nestjs/microservices';
import { AuthService } from '../services/auth.service';
import { TokenService } from '../services/token.service';
import { UserService } from '../services/user.service';

interface CheckPermissionRequest {
  token: string;
  resource: string;
  action: string;
  context?: Record<string, string>;
}

interface CheckPermissionResponse {
  allowed: boolean;
  userId: string;
  reason: string;
  roles: string[];
  permissions: string[];
}

interface GetUserInfoRequest {
  userId: string;
}

interface GetUserInfoResponse {
  userId: string;
  email: string;
  fullName: string;
  status: string;
  roles: string[];
}

interface ValidateTokenRequest {
  token: string;
}

interface ValidateTokenResponse {
  valid: boolean;
  userId: string;
  email: string;
  expiresAt: number;
}

@Controller()
export class AuthGrpcController {
  private readonly logger = new Logger(AuthGrpcController.name);

  constructor(
    private readonly authService: AuthService,
    private readonly tokenService: TokenService,
    private readonly userService: UserService,
  ) {}

  @GrpcMethod('AuthService', 'Authorize')
  async authorize(data: { userId: string; resource: string; action: string }) {
    return this.authService.authorize(data.userId, data.resource, data.action);
  }

  @GrpcMethod('AuthService', 'CheckPermission')
  async checkPermission(data: CheckPermissionRequest): Promise<CheckPermissionResponse> {
    try {
      // 1. Validate token
      let decoded;
      try {
        decoded = await this.tokenService.verifyAccessToken(data.token);
      } catch {
        return {
          allowed: false,
          userId: '',
          reason: 'Invalid or expired token',
          roles: [],
          permissions: [],
        };
      }

      const userId = decoded.sub;

      // 2. Get user with roles
      const user = await this.userService.findById(userId);
      if (!user || user.status !== 'ACTIVE') {
        return {
          allowed: false,
          userId,
          reason: 'User not found or inactive',
          roles: [],
          permissions: [],
        };
      }

      // 3. Check permission
      const hasPermission = await this.userService.hasPermission(
        userId,
        `${data.resource}:${data.action}`,
      );

      // 4. Get roles and permissions
      const roles = await this.userService.getUserRoles(userId);
      const permissions = await this.userService.getUserPermissionsCached(userId);

      return {
        allowed: hasPermission,
        userId,
        reason: hasPermission ? 'Authorized' : 'Insufficient permissions',
        roles,
        permissions,
      };
    } catch (error) {
      this.logger.error(
        'CheckPermission failed',
        error instanceof Error ? error.stack : String(error),
      );
      return {
        allowed: false,
        userId: '',
        reason: error instanceof Error ? error.message : 'Internal error',
        roles: [],
        permissions: [],
      };
    }
  }

  @GrpcMethod('AuthService', 'GetUserInfo')
  async getUserInfo(data: GetUserInfoRequest): Promise<GetUserInfoResponse> {
    try {
      const user = await this.userService.findById(data.userId);
      if (!user) {
        return {
          userId: '',
          email: '',
          fullName: '',
          status: 'NOT_FOUND',
          roles: [],
        };
      }

      const roles = await this.userService.getUserRoles(data.userId);

      return {
        userId: user.id,
        email: user.email,
        fullName: user.fullName,
        status: user.status,
        roles,
      };
    } catch (error) {
      this.logger.error(
        'GetUserInfo failed',
        error instanceof Error ? error.stack : String(error),
      );
      return {
        userId: '',
        email: '',
        fullName: '',
        status: 'ERROR',
        roles: [],
      };
    }
  }

  @GrpcMethod('AuthService', 'ValidateToken')
  async validateToken(data: ValidateTokenRequest): Promise<ValidateTokenResponse> {
    try {
      const decoded = await this.tokenService.verifyAccessToken(data.token);

      return {
        valid: true,
        userId: decoded.sub,
        email: decoded.email,
        expiresAt: decoded.exp || 0,
      };
    } catch {
      return {
        valid: false,
        userId: '',
        email: '',
        expiresAt: 0,
      };
    }
  }
}
