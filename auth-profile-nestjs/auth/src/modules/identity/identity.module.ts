import { Module, Global } from '@nestjs/common';
import { TypeOrmModule } from '@nestjs/typeorm';
import { JwtModule } from '@nestjs/jwt';
import { PassportModule } from '@nestjs/passport';
import { ConfigModule, ConfigService } from '@nestjs/config';
import { CacheModule } from '@nestjs/cache-manager';
import { APP_FILTER, APP_GUARD } from '@nestjs/core';
import { redisStore } from 'cache-manager-redis-yet';

// Entities
import { User, Role, Permission, RolePermission, UserRole, RefreshToken } from './entities';

// Services
import {
  AuthService,
  RedisCacheService,
  TokenService,
  UserService,
  SeedingService,
} from './services';

// Events
import { UserEventsService } from './events';

// Strategies
import { JwtStrategy } from './strategies';

// Guards
import { JwtAuthGuard, PermissionsGuard } from './guards';

// Filters
import { HttpExceptionFilter } from './filters';

// Controller
import { AuthController } from './auth.controller';

// Config
import { identityConfig } from './config';

@Global()
@Module({
  imports: [
    // Config
    ConfigModule.forFeature(identityConfig),

    // TypeORM entities
    TypeOrmModule.forFeature([User, Role, Permission, RolePermission, UserRole, RefreshToken]),

    // Passport configuration
    PassportModule.register({ defaultStrategy: 'jwt' }),

    // JWT configuration
    JwtModule.registerAsync({
      imports: [ConfigModule],
      useFactory: (configService: ConfigService) => ({
        secret: configService.getOrThrow<string>('JWT_ACCESS_SECRET'),
        signOptions: {
          expiresIn: configService.get<string>('JWT_ACCESS_EXPIRY', '15m'),
        },
      }),
      inject: [ConfigService],
    }),

    // Redis cache configuration
    CacheModule.registerAsync({
      imports: [ConfigModule],
      useFactory: async (configService: ConfigService) => ({
        store: await redisStore({
          socket: {
            host: configService.get<string>('REDIS_HOST', 'localhost'),
            port: configService.get<number>('REDIS_PORT', 6379),
          },
          password: configService.get<string>('REDIS_PASSWORD') || undefined,
          ttl: 3600000, // Default TTL: 1 hour in milliseconds
        }),
      }),
      inject: [ConfigService],
    }),
  ],
  controllers: [AuthController],
  providers: [
    // Services
    AuthService,
    RedisCacheService,
    TokenService,
    UserService,
    SeedingService,
    UserEventsService,

    // Strategies
    JwtStrategy,

    // Guards
    JwtAuthGuard,
    PermissionsGuard,

    // Global exception filter for this module
    {
      provide: APP_FILTER,
      useClass: HttpExceptionFilter,
    },
  ],
  exports: [
    // Services
    AuthService,
    RedisCacheService,
    TokenService,
    UserService,

    // Guards
    JwtAuthGuard,
    PermissionsGuard,

    // Modules
    JwtModule,
    PassportModule,
    TypeOrmModule,
  ],
})
export class IdentityModule {}
