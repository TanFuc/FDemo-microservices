import {
  Injectable,
  UnauthorizedException,
  Logger,
  InternalServerErrorException,
} from '@nestjs/common';
import { JwtService } from '@nestjs/jwt';
import { ConfigService } from '@nestjs/config';
import { InjectRepository } from '@nestjs/typeorm';
import { Repository, LessThan } from 'typeorm';
import * as bcrypt from 'bcrypt';
import { v4 as uuidv4 } from 'uuid';

import { RefreshToken } from '../entities';
import {
  AccessTokenPayload,
  RefreshTokenPayload,
  AuthTokens,
  DeviceInfo,
} from '../interfaces';
import { TOKEN_CONFIG, ERROR_MESSAGES } from '../constants';
import { RedisCacheService } from './redis-cache.service';

@Injectable()
export class TokenService {
  private readonly logger = new Logger(TokenService.name);

  constructor(
    @InjectRepository(RefreshToken)
    private readonly refreshTokenRepository: Repository<RefreshToken>,
    private readonly jwtService: JwtService,
    private readonly configService: ConfigService,
    private readonly redisCacheService: RedisCacheService,
  ) {}

  /**
   * Generate access and refresh token pair
   */
  async generateTokenPair(
    userId: string,
    email: string,
    deviceInfo: DeviceInfo,
  ): Promise<AuthTokens> {
    try {
      const accessTokenJti = uuidv4();
      const refreshTokenJti = uuidv4();

      const accessTokenPayload: AccessTokenPayload = {
        sub: userId,
        email,
        jti: accessTokenJti,
      };

      const refreshTokenPayload: RefreshTokenPayload = {
        sub: userId,
        jti: refreshTokenJti,
        deviceId: deviceInfo.deviceId,
      };

      const [accessToken, refreshToken] = await Promise.all([
        this.jwtService.signAsync(accessTokenPayload, {
          secret: this.configService.getOrThrow<string>('JWT_ACCESS_SECRET'),
          expiresIn: TOKEN_CONFIG.ACCESS_TOKEN_EXPIRY,
        }),
        this.jwtService.signAsync(refreshTokenPayload, {
          secret: this.configService.getOrThrow<string>('JWT_REFRESH_SECRET'),
          expiresIn: TOKEN_CONFIG.REFRESH_TOKEN_EXPIRY,
        }),
      ]);

      // Save refresh token to database
      await this.saveRefreshToken(
        refreshTokenJti,
        userId,
        refreshToken,
        deviceInfo,
      );

      // Calculate expiresIn in seconds
      const decoded = this.jwtService.decode(accessToken) as AccessTokenPayload;
      const expiresIn = decoded.exp ? decoded.exp - Math.floor(Date.now() / 1000) : 900;

      return {
        accessToken,
        refreshToken,
        expiresIn,
      };
    } catch (error) {
      this.logger.error(
        'Failed to generate token pair',
        error instanceof Error ? error.stack : String(error),
      );
      throw new InternalServerErrorException('Failed to generate tokens');
    }
  }

  /**
   * Verify and decode access token
   */
  async verifyAccessToken(token: string): Promise<AccessTokenPayload> {
    try {
      const payload = await this.jwtService.verifyAsync<AccessTokenPayload>(token, {
        secret: this.configService.getOrThrow<string>('JWT_ACCESS_SECRET'),
      });

      // Check if token is blacklisted
      const isBlacklisted = await this.redisCacheService.isTokenBlacklisted(payload.jti);
      if (isBlacklisted) {
        throw new UnauthorizedException(ERROR_MESSAGES.TOKEN_REVOKED);
      }

      return payload;
    } catch (error) {
      if (error instanceof UnauthorizedException) {
        throw error;
      }
      this.logger.debug('Access token verification failed');
      throw new UnauthorizedException(ERROR_MESSAGES.TOKEN_INVALID);
    }
  }

  /**
   * Verify and decode refresh token
   */
  async verifyRefreshToken(token: string): Promise<RefreshTokenPayload> {
    try {
      return await this.jwtService.verifyAsync<RefreshTokenPayload>(token, {
        secret: this.configService.getOrThrow<string>('JWT_REFRESH_SECRET'),
      });
    } catch {
      throw new UnauthorizedException(ERROR_MESSAGES.REFRESH_TOKEN_INVALID);
    }
  }

  /**
   * Rotate refresh token (revoke old, issue new)
   */
  async rotateRefreshToken(
    oldToken: string,
    deviceInfo: DeviceInfo,
  ): Promise<{ userId: string; email: string; tokens: AuthTokens }> {
    try {
      // Verify the old refresh token
      const payload = await this.verifyRefreshToken(oldToken);

      // Find and validate stored token
      const storedToken = await this.refreshTokenRepository.findOne({
        where: {
          id: payload.jti,
          userId: payload.sub,
          isRevoked: false,
        },
        relations: ['user'],
      });

      if (!storedToken) {
        throw new UnauthorizedException(ERROR_MESSAGES.TOKEN_REVOKED);
      }

      if (new Date() > storedToken.expiresAt) {
        throw new UnauthorizedException(ERROR_MESSAGES.REFRESH_TOKEN_EXPIRED);
      }

      // Revoke the old token
      storedToken.isRevoked = true;
      await this.refreshTokenRepository.save(storedToken);

      // Generate new token pair
      const tokens = await this.generateTokenPair(
        storedToken.userId,
        storedToken.user.email,
        deviceInfo,
      );

      return {
        userId: storedToken.userId,
        email: storedToken.user.email,
        tokens,
      };
    } catch (error) {
      if (error instanceof UnauthorizedException) {
        throw error;
      }
      this.logger.error(
        'Token rotation failed',
        error instanceof Error ? error.stack : String(error),
      );
      throw new UnauthorizedException(ERROR_MESSAGES.REFRESH_TOKEN_INVALID);
    }
  }

  /**
   * Revoke a specific token by JTI
   */
  async revokeToken(jti: string, remainingTtlSeconds: number): Promise<void> {
    try {
      if (remainingTtlSeconds > 0) {
        await this.redisCacheService.blacklistToken(jti, remainingTtlSeconds);
      }
    } catch (error) {
      this.logger.error(
        `Failed to revoke token ${jti}`,
        error instanceof Error ? error.stack : String(error),
      );
    }
  }

  /**
   * Revoke all refresh tokens for a user
   */
  async revokeAllUserTokens(userId: string): Promise<void> {
    try {
      await this.refreshTokenRepository.update(
        { userId, isRevoked: false },
        { isRevoked: true },
      );
      this.logger.debug(`Revoked all tokens for user ${userId}`);
    } catch (error) {
      this.logger.error(
        `Failed to revoke all tokens for user ${userId}`,
        error instanceof Error ? error.stack : String(error),
      );
    }
  }

  /**
   * Revoke refresh tokens for a specific device
   */
  async revokeDeviceTokens(userId: string, deviceId: string): Promise<void> {
    try {
      await this.refreshTokenRepository.update(
        { userId, deviceInfo: deviceId, isRevoked: false },
        { isRevoked: true },
      );
      this.logger.debug(`Revoked tokens for user ${userId}, device ${deviceId}`);
    } catch (error) {
      this.logger.error(
        `Failed to revoke device tokens for user ${userId}`,
        error instanceof Error ? error.stack : String(error),
      );
    }
  }

  /**
   * Clean up expired tokens (scheduled task)
   */
  async cleanupExpiredTokens(): Promise<number> {
    try {
      const result = await this.refreshTokenRepository.delete({
        expiresAt: LessThan(new Date()),
      });
      const deletedCount = result.affected ?? 0;
      this.logger.log(`Cleaned up ${deletedCount} expired refresh tokens`);
      return deletedCount;
    } catch (error) {
      this.logger.error(
        'Failed to cleanup expired tokens',
        error instanceof Error ? error.stack : String(error),
      );
      return 0;
    }
  }

  /**
   * Get active sessions for a user
   */
  async getUserSessions(userId: string): Promise<RefreshToken[]> {
    return this.refreshTokenRepository.find({
      where: { userId, isRevoked: false },
      order: { createdAt: 'DESC' },
    });
  }

  /**
   * Save refresh token hash to database
   */
  private async saveRefreshToken(
    jti: string,
    userId: string,
    token: string,
    deviceInfo: DeviceInfo,
  ): Promise<RefreshToken> {
    const tokenHash = await bcrypt.hash(token, TOKEN_CONFIG.BCRYPT_SALT_ROUNDS);

    const expiresAt = new Date();
    expiresAt.setDate(expiresAt.getDate() + TOKEN_CONFIG.REFRESH_TOKEN_EXPIRY_DAYS);

    const refreshTokenEntity = this.refreshTokenRepository.create({
      id: jti,
      userId,
      tokenHash,
      deviceInfo: deviceInfo.deviceId,
      ipAddress: deviceInfo.ipAddress,
      expiresAt,
      isRevoked: false,
    });

    return this.refreshTokenRepository.save(refreshTokenEntity);
  }

  /**
   * Decode token without verification (for reading claims)
   */
  decodeToken<T extends AccessTokenPayload | RefreshTokenPayload>(token: string): T {
    return this.jwtService.decode(token) as T;
  }
}
