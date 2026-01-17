import {
  Controller,
  Post,
  Get,
  Body,
  UseGuards,
  Req,
  HttpCode,
  HttpStatus,
  UseInterceptors,
} from '@nestjs/common';
import {
  ApiTags,
  ApiOperation,
  ApiResponse as SwaggerResponse,
  ApiBearerAuth,
  ApiHeader,
} from '@nestjs/swagger';
import { Request } from 'express';
import { v4 as uuidv4 } from 'uuid';

import { AuthService } from './services/auth.service';
import { RegisterDto, LoginDto, RefreshTokenDto, LogoutDto } from './dto';
import {
  ApiResponse,
  LoginResponse,
  RegisterResponse,
  AuthTokens,
  UserResponse,
  DeviceInfo,
  AuthenticatedUser,
} from './interfaces';
import { JwtAuthGuard, PermissionsGuard } from './guards';
import { RequirePermissions, Public, CurrentUser } from './decorators';
import { ResponseInterceptor } from './interceptors';
import { SUCCESS_MESSAGES, RESPONSE_CODES, PERMISSIONS } from './constants';

@ApiTags('Authentication')
@Controller('auth')
@UseInterceptors(ResponseInterceptor)
export class AuthController {
  constructor(private readonly authService: AuthService) {}

  /**
   * Register a new user
   */
  @Public()
  @Post('register')
  @HttpCode(HttpStatus.CREATED)
  @ApiOperation({ summary: 'Register a new user account' })
  @SwaggerResponse({ status: 201, description: 'User registered successfully' })
  @SwaggerResponse({ status: 400, description: 'Validation error' })
  @SwaggerResponse({ status: 409, description: 'Email already exists' })
  async register(
    @Body() dto: RegisterDto,
  ): Promise<ApiResponse<RegisterResponse>> {
    const result = await this.authService.register(dto);

    return {
      success: true,
      code: RESPONSE_CODES.CREATED,
      message: SUCCESS_MESSAGES.REGISTERED,
      data: result,
      timestamp: new Date().toISOString(),
    };
  }

  /**
   * Login with email and password
   */
  @Public()
  @Post('login')
  @HttpCode(HttpStatus.OK)
  @ApiOperation({ summary: 'Login with email and password' })
  @ApiHeader({ name: 'x-device-id', required: false, description: 'Device identifier' })
  @SwaggerResponse({ status: 200, description: 'Login successful' })
  @SwaggerResponse({ status: 401, description: 'Invalid credentials' })
  async login(
    @Body() dto: LoginDto,
    @Req() req: Request,
  ): Promise<ApiResponse<LoginResponse>> {
    const deviceInfo = this.extractDeviceInfo(req);
    const result = await this.authService.login(dto, deviceInfo);

    return {
      success: true,
      code: RESPONSE_CODES.SUCCESS,
      message: SUCCESS_MESSAGES.LOGGED_IN,
      data: result,
      timestamp: new Date().toISOString(),
    };
  }

  /**
   * Refresh access and refresh tokens
   */
  @Public()
  @Post('refresh')
  @HttpCode(HttpStatus.OK)
  @ApiOperation({ summary: 'Refresh access token using refresh token' })
  @ApiHeader({ name: 'x-device-id', required: false, description: 'Device identifier' })
  @SwaggerResponse({ status: 200, description: 'Tokens refreshed successfully' })
  @SwaggerResponse({ status: 401, description: 'Invalid or expired refresh token' })
  async refresh(
    @Body() dto: RefreshTokenDto,
    @Req() req: Request,
  ): Promise<ApiResponse<AuthTokens>> {
    const deviceInfo = this.extractDeviceInfo(req);
    const tokens = await this.authService.refreshTokens(dto, deviceInfo);

    return {
      success: true,
      code: RESPONSE_CODES.SUCCESS,
      message: SUCCESS_MESSAGES.TOKEN_REFRESHED,
      data: tokens,
      timestamp: new Date().toISOString(),
    };
  }

  /**
   * Logout and revoke tokens
   */
  @UseGuards(JwtAuthGuard)
  @Post('logout')
  @HttpCode(HttpStatus.OK)
  @ApiBearerAuth()
  @ApiOperation({ summary: 'Logout and revoke tokens' })
  @SwaggerResponse({ status: 200, description: 'Logged out successfully' })
  @SwaggerResponse({ status: 401, description: 'Unauthorized' })
  async logout(
    @Body() dto: LogoutDto,
    @CurrentUser() user: AuthenticatedUser,
  ): Promise<ApiResponse<null>> {
    await this.authService.logout(user.id, user.jti, user.exp, dto.deviceId);

    return {
      success: true,
      code: RESPONSE_CODES.SUCCESS,
      message: SUCCESS_MESSAGES.LOGGED_OUT,
      data: null,
      timestamp: new Date().toISOString(),
    };
  }

  /**
   * Logout from all devices
   */
  @UseGuards(JwtAuthGuard)
  @Post('logout-all')
  @HttpCode(HttpStatus.OK)
  @ApiBearerAuth()
  @ApiOperation({ summary: 'Logout from all devices' })
  @SwaggerResponse({ status: 200, description: 'Logged out from all devices' })
  @SwaggerResponse({ status: 401, description: 'Unauthorized' })
  async logoutAll(
    @CurrentUser() user: AuthenticatedUser,
  ): Promise<ApiResponse<null>> {
    await this.authService.logoutAllDevices(user.id);

    return {
      success: true,
      code: RESPONSE_CODES.SUCCESS,
      message: 'Logged out from all devices',
      data: null,
      timestamp: new Date().toISOString(),
    };
  }

  /**
   * Get current user profile
   */
  @UseGuards(JwtAuthGuard)
  @Get('profile')
  @ApiBearerAuth()
  @ApiOperation({ summary: 'Get current user profile' })
  @SwaggerResponse({ status: 200, description: 'Profile retrieved' })
  @SwaggerResponse({ status: 401, description: 'Unauthorized' })
  async getProfile(
    @CurrentUser() user: AuthenticatedUser,
  ): Promise<ApiResponse<UserResponse>> {
    const profile = await this.authService.getProfile(user.id);

    return {
      success: true,
      code: RESPONSE_CODES.SUCCESS,
      message: SUCCESS_MESSAGES.PROFILE_RETRIEVED,
      data: profile,
      timestamp: new Date().toISOString(),
    };
  }

  /**
   * Get active sessions
   */
  @UseGuards(JwtAuthGuard)
  @Get('sessions')
  @ApiBearerAuth()
  @ApiOperation({ summary: 'Get all active sessions' })
  @SwaggerResponse({ status: 200, description: 'Sessions retrieved' })
  @SwaggerResponse({ status: 401, description: 'Unauthorized' })
  async getSessions(@CurrentUser() user: AuthenticatedUser) {
    const sessions = await this.authService.getActiveSessions(user.id);

    return {
      success: true,
      code: RESPONSE_CODES.SUCCESS,
      message: 'Sessions retrieved successfully',
      data: sessions.map((s) => ({
        id: s.id,
        deviceInfo: s.deviceInfo,
        ipAddress: s.ipAddress,
        createdAt: s.createdAt,
        expiresAt: s.expiresAt,
      })),
      timestamp: new Date().toISOString(),
    };
  }

  /**
   * Check if user has specific permission (example protected endpoint)
   */
  @UseGuards(JwtAuthGuard, PermissionsGuard)
  @RequirePermissions(PERMISSIONS.PRODUCT_CREATE)
  @Get('check-permission')
  @ApiBearerAuth()
  @ApiOperation({ summary: 'Check user permission (example endpoint)' })
  @SwaggerResponse({ status: 200, description: 'Permission granted' })
  @SwaggerResponse({ status: 401, description: 'Unauthorized' })
  @SwaggerResponse({ status: 403, description: 'Permission denied' })
  checkPermission(): ApiResponse<{ hasAccess: boolean }> {
    return {
      success: true,
      code: RESPONSE_CODES.SUCCESS,
      message: SUCCESS_MESSAGES.PERMISSION_GRANTED,
      data: { hasAccess: true },
      timestamp: new Date().toISOString(),
    };
  }

  /**
   * Extract device information from request
   */
  private extractDeviceInfo(req: Request): DeviceInfo {
    const deviceId = this.getDeviceId(req);
    const ipAddress = this.getClientIp(req);
    const userAgent = req.headers['user-agent'] || 'unknown';

    return {
      deviceId,
      ipAddress,
      userAgent,
    };
  }

  /**
   * Extract client IP address from request
   */
  private getClientIp(req: Request): string {
    const forwarded = req.headers['x-forwarded-for'];
    if (typeof forwarded === 'string') {
      return forwarded.split(',')[0].trim();
    }
    if (Array.isArray(forwarded)) {
      return forwarded[0];
    }
    return req.ip || req.socket?.remoteAddress || 'unknown';
  }

  /**
   * Extract or generate device ID from request
   */
  private getDeviceId(req: Request): string {
    const deviceId = req.headers['x-device-id'];
    if (typeof deviceId === 'string' && deviceId.length > 0) {
      return deviceId;
    }
    // Generate a device ID based on user-agent if not provided
    const userAgent = req.headers['user-agent'] || 'unknown';
    return `${userAgent.substring(0, 50)}-${uuidv4().substring(0, 8)}`;
  }
}
