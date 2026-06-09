import { Injectable, UnauthorizedException } from '@nestjs/common';
import { PassportStrategy } from '@nestjs/passport';
import { ExtractJwt, Strategy } from 'passport-jwt';
import { ConfigService } from '@nestjs/config';

import { UserService } from '../services/user.service';
import { RedisCacheService } from '../services/redis-cache.service';
import { AccessTokenPayload, AuthenticatedUser } from '../interfaces';
import { ERROR_MESSAGES } from '../constants';

@Injectable()
export class JwtStrategy extends PassportStrategy(Strategy, 'jwt') {
  constructor(
    private readonly configService: ConfigService,
    private readonly userService: UserService,
    private readonly redisCacheService: RedisCacheService,
  ) {
    const jwtSecret = configService.get<string>('JWT_ACCESS_SECRET');
    if (!jwtSecret) {
      throw new Error('JWT_ACCESS_SECRET is not configured');
    }

    super({
      jwtFromRequest: ExtractJwt.fromAuthHeaderAsBearerToken(),
      ignoreExpiration: false,
      secretOrKey: jwtSecret,
    });
  }

  async validate(payload: AccessTokenPayload): Promise<AuthenticatedUser> {
    // Check if token is blacklisted
    const isBlacklisted = await this.redisCacheService.isTokenBlacklisted(payload.jti);

    if (isBlacklisted) {
      throw new UnauthorizedException(ERROR_MESSAGES.TOKEN_REVOKED);
    }

    // Validate user exists and is active
    const user = await this.userService.findById(payload.sub);

    if (!user) {
      throw new UnauthorizedException(ERROR_MESSAGES.USER_NOT_FOUND);
    }

    if (user.status !== 'ACTIVE') {
      throw new UnauthorizedException(ERROR_MESSAGES.USER_INACTIVE);
    }

    return {
      id: payload.sub,
      email: payload.email,
      jti: payload.jti,
      exp: payload.exp ?? 0,
    };
  }
}
