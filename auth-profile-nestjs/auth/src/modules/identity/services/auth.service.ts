import {
  Injectable,
  UnauthorizedException,
  InternalServerErrorException,
  Logger,
} from '@nestjs/common';

import { RegisterDto, LoginDto, RefreshTokenDto } from '../dto';
import {
  LoginResponse,
  RegisterResponse,
  AuthTokens,
  DeviceInfo,
  UserResponse,
} from '../interfaces';
import { ERROR_MESSAGES, SUCCESS_MESSAGES } from '../constants';
import { TokenService } from './token.service';
import { UserService } from './user.service';
import { RedisCacheService } from './redis-cache.service';
import { UserEventsService } from '../events';

@Injectable()
export class AuthService {
  private readonly logger = new Logger(AuthService.name);

  constructor(
    private readonly tokenService: TokenService,
    private readonly userService: UserService,
    private readonly redisCacheService: RedisCacheService,
    private readonly userEventsService: UserEventsService,
  ) {}

  /**
   * Register a new user
   */
  async register(dto: RegisterDto): Promise<RegisterResponse> {
    try {
      const user = await this.userService.createUser(dto);
      const userResponse = await this.userService.toUserResponse(user);

      this.logger.log(`User registered: ${user.email}`);

      // Publish user.registered event for Profile Service
      await this.userEventsService.publishUserRegistered({
        userId: user.id,
        email: user.email,
        fullName: dto.fullName || user.email.split('@')[0],
        registeredAt: new Date().toISOString(),
      });

      return { user: userResponse };
    } catch (error) {
      this.logger.error(
        'Registration failed',
        error instanceof Error ? error.stack : String(error),
      );
      throw error;
    }
  }

  /**
   * Login user and generate tokens
   */
  async login(dto: LoginDto, deviceInfo: DeviceInfo): Promise<LoginResponse> {
    try {
      // Validate credentials
      const user = await this.userService.validateCredentials(
        dto.email,
        dto.password,
      );

      if (!user) {
        throw new UnauthorizedException(ERROR_MESSAGES.INVALID_CREDENTIALS);
      }

      // Fetch and cache permissions
      const permissions = await this.userService.getUserPermissions(user.id);
      await this.redisCacheService.cacheUserPermissions(user.id, permissions);

      // Generate token pair
      const tokens = await this.tokenService.generateTokenPair(
        user.id,
        user.email,
        deviceInfo,
      );

      // Get user response
      const userResponse = await this.userService.toUserResponse(user);

      this.logger.log(`User logged in: ${user.email}`);

      return {
        user: userResponse,
        tokens,
      };
    } catch (error) {
      if (error instanceof UnauthorizedException) {
        throw error;
      }
      this.logger.error(
        'Login failed',
        error instanceof Error ? error.stack : String(error),
      );
      throw new InternalServerErrorException(ERROR_MESSAGES.LOGIN_FAILED);
    }
  }

  /**
   * Refresh access and refresh tokens
   */
  async refreshTokens(
    dto: RefreshTokenDto,
    deviceInfo: DeviceInfo,
  ): Promise<AuthTokens> {
    try {
      const result = await this.tokenService.rotateRefreshToken(
        dto.refreshToken,
        deviceInfo,
      );

      this.logger.debug(`Tokens refreshed for user: ${result.userId}`);

      return result.tokens;
    } catch (error) {
      if (error instanceof UnauthorizedException) {
        throw error;
      }
      this.logger.error(
        'Token refresh failed',
        error instanceof Error ? error.stack : String(error),
      );
      throw new UnauthorizedException(ERROR_MESSAGES.REFRESH_TOKEN_INVALID);
    }
  }

  /**
   * Logout user - revoke tokens and clear cache
   */
  async logout(
    userId: string,
    accessTokenJti: string,
    accessTokenExp: number,
    deviceId?: string,
  ): Promise<void> {
    try {
      // Blacklist the access token
      const remainingTtl = accessTokenExp - Math.floor(Date.now() / 1000);
      await this.tokenService.revokeToken(accessTokenJti, remainingTtl);

      // Revoke refresh tokens
      if (deviceId) {
        await this.tokenService.revokeDeviceTokens(userId, deviceId);
        await this.redisCacheService.removeActiveSession(userId, deviceId);
      } else {
        await this.tokenService.revokeAllUserTokens(userId);
      }

      // Invalidate cached permissions
      await this.redisCacheService.invalidateUserPermissions(userId);

      this.logger.log(`User logged out: ${userId}`);
    } catch (error) {
      this.logger.error(
        `Logout failed for user ${userId}`,
        error instanceof Error ? error.stack : String(error),
      );
      throw new InternalServerErrorException(ERROR_MESSAGES.LOGOUT_FAILED);
    }
  }

  /**
   * Get user profile by ID
   */
  async getProfile(userId: string): Promise<UserResponse> {
    const user = await this.userService.findById(userId);

    if (!user) {
      throw new UnauthorizedException(ERROR_MESSAGES.USER_NOT_FOUND);
    }

    return this.userService.toUserResponse(user);
  }

  /**
   * Fetch user permissions (for guards)
   */
  async fetchUserPermissionsFromDB(userId: string): Promise<string[]> {
    return this.userService.getUserPermissions(userId);
  }

  /**
   * Validate user exists (for JWT strategy)
   */
  async validateUser(userId: string) {
    return this.userService.findById(userId);
  }

  /**
   * Logout from all devices
   */
  async logoutAllDevices(userId: string): Promise<void> {
    try {
      await this.tokenService.revokeAllUserTokens(userId);
      await this.redisCacheService.invalidateUserPermissions(userId);

      this.logger.log(`User logged out from all devices: ${userId}`);
    } catch (error) {
      this.logger.error(
        `Logout all devices failed for user ${userId}`,
        error instanceof Error ? error.stack : String(error),
      );
      throw new InternalServerErrorException(ERROR_MESSAGES.LOGOUT_FAILED);
    }
  }

  /**
   * Get user's active sessions
   */
  async getActiveSessions(userId: string) {
    return this.tokenService.getUserSessions(userId);
  }
}
