import { Injectable, Inject, Logger } from '@nestjs/common';
import { CACHE_MANAGER } from '@nestjs/cache-manager';
import { Cache } from 'cache-manager';

@Injectable()
export class RedisCacheService {
  private readonly logger = new Logger(RedisCacheService.name);

  private readonly PERMISSIONS_TTL = 3600; // 1 hour in seconds
  private readonly PERMISSIONS_KEY_PREFIX = 'identity:user:';
  private readonly PERMISSIONS_KEY_SUFFIX = ':permissions';
  private readonly BLACKLIST_KEY_PREFIX = 'identity:blacklist:';
  private readonly SESSION_KEY_PREFIX = 'identity:session:';

  constructor(@Inject(CACHE_MANAGER) private readonly cacheManager: Cache) {}

  /**
   * Cache user permissions in Redis
   * Key: identity:user:{userId}:permissions
   * TTL: 3600 seconds (1 hour)
   */
  async cacheUserPermissions(userId: string, permissions: string[]): Promise<void> {
    try {
      const key = `${this.PERMISSIONS_KEY_PREFIX}${userId}${this.PERMISSIONS_KEY_SUFFIX}`;
      await this.cacheManager.set(key, JSON.stringify(permissions), this.PERMISSIONS_TTL * 1000);
      this.logger.debug(`Cached permissions for user ${userId}`);
    } catch (error) {
      this.logger.error(
        `Failed to cache permissions for user ${userId}`,
        error instanceof Error ? error.stack : String(error),
      );
      throw error;
    }
  }

  /**
   * Get user permissions from Redis cache
   * Returns null if cache miss
   */
  async getUserPermissions(userId: string): Promise<string[] | null> {
    try {
      const key = `${this.PERMISSIONS_KEY_PREFIX}${userId}${this.PERMISSIONS_KEY_SUFFIX}`;
      const cached = await this.cacheManager.get<string>(key);

      if (!cached) {
        this.logger.debug(`Cache miss for user permissions: ${userId}`);
        return null;
      }

      const permissions: string[] = JSON.parse(cached);
      this.logger.debug(`Cache hit for user permissions: ${userId}`);
      return permissions;
    } catch (error) {
      this.logger.error(
        `Failed to get cached permissions for user ${userId}`,
        error instanceof Error ? error.stack : String(error),
      );
      return null;
    }
  }

  /**
   * Invalidate user permissions cache
   * Called when user roles are changed
   */
  async invalidateUserPermissions(userId: string): Promise<void> {
    try {
      const key = `${this.PERMISSIONS_KEY_PREFIX}${userId}${this.PERMISSIONS_KEY_SUFFIX}`;
      await this.cacheManager.del(key);
      this.logger.debug(`Invalidated permissions cache for user ${userId}`);
    } catch (error) {
      this.logger.error(
        `Failed to invalidate permissions for user ${userId}`,
        error instanceof Error ? error.stack : String(error),
      );
      throw error;
    }
  }

  /**
   * Add a JWT token to the blacklist
   * Key: identity:blacklist:{jti}
   * TTL: Remaining time until token expiration
   */
  async blacklistToken(jti: string, ttlSeconds: number): Promise<void> {
    try {
      const key = `${this.BLACKLIST_KEY_PREFIX}${jti}`;
      await this.cacheManager.set(key, '1', ttlSeconds * 1000);
      this.logger.debug(`Blacklisted token ${jti} with TTL ${ttlSeconds}s`);
    } catch (error) {
      this.logger.error(
        `Failed to blacklist token ${jti}`,
        error instanceof Error ? error.stack : String(error),
      );
      throw error;
    }
  }

  /**
   * Check if a token is blacklisted
   */
  async isTokenBlacklisted(jti: string): Promise<boolean> {
    try {
      const key = `${this.BLACKLIST_KEY_PREFIX}${jti}`;
      const value = await this.cacheManager.get<string>(key);
      const isBlacklisted = value === '1';

      if (isBlacklisted) {
        this.logger.debug(`Token ${jti} is blacklisted`);
      }

      return isBlacklisted;
    } catch (error) {
      this.logger.error(
        `Failed to check blacklist for token ${jti}`,
        error instanceof Error ? error.stack : String(error),
      );
      return false;
    }
  }

  /**
   * Store active session information
   * Key: identity:session:{userId}:{deviceId}
   */
  async setActiveSession(
    userId: string,
    deviceId: string,
    sessionData: { refreshTokenId: string; ipAddress: string },
    ttlSeconds: number,
  ): Promise<void> {
    try {
      const key = `${this.SESSION_KEY_PREFIX}${userId}:${deviceId}`;
      await this.cacheManager.set(key, JSON.stringify(sessionData), ttlSeconds * 1000);
      this.logger.debug(`Set active session for user ${userId}, device ${deviceId}`);
    } catch (error) {
      this.logger.error(
        `Failed to set active session for user ${userId}`,
        error instanceof Error ? error.stack : String(error),
      );
      throw error;
    }
  }

  /**
   * Remove active session
   */
  async removeActiveSession(userId: string, deviceId: string): Promise<void> {
    try {
      const key = `${this.SESSION_KEY_PREFIX}${userId}:${deviceId}`;
      await this.cacheManager.del(key);
      this.logger.debug(`Removed session for user ${userId}, device ${deviceId}`);
    } catch (error) {
      this.logger.error(
        `Failed to remove session for user ${userId}`,
        error instanceof Error ? error.stack : String(error),
      );
      throw error;
    }
  }
}
