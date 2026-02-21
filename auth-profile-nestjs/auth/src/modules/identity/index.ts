// Module
export { IdentityModule } from './identity.module';

// Services
export {
  AuthService,
  RedisCacheService,
  TokenService,
  UserService,
  SeedingService,
} from './services';

// Guards
export { JwtAuthGuard, PermissionsGuard } from './guards';

// Decorators
export { RequirePermissions, Public, CurrentUser } from './decorators';

// Interfaces
export {
  AccessTokenPayload,
  RefreshTokenPayload,
  AuthenticatedUser,
  AuthenticatedRequest,
  ApiResponse,
  UserResponse,
  AuthTokens,
  LoginResponse,
  RegisterResponse,
  DeviceInfo,
  SessionInfo,
  PermissionCheckResult,
  PaginatedResponse,
  PaginationMeta,
} from './interfaces';

// DTOs
export { RegisterDto, LoginDto, RefreshTokenDto, LogoutDto } from './dto';

// Entities
export { User, Role, Permission, RolePermission, UserRole, RefreshToken } from './entities';

// Constants
export {
  CACHE_KEYS,
  CACHE_TTL,
  TOKEN_CONFIG,
  DEFAULT_ROLES,
  PERMISSIONS,
  ERROR_MESSAGES,
  SUCCESS_MESSAGES,
  RESPONSE_CODES,
  METADATA_KEYS,
} from './constants';

// Config
export { identityConfig, identityConfigValidationSchema } from './config';

// Filters
export { HttpExceptionFilter } from './filters';

// Interceptors
export { ResponseInterceptor } from './interceptors';
