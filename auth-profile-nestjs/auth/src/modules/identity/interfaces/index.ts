import { Request } from 'express';

// JWT Payloads
export interface AccessTokenPayload {
  sub: string;
  email: string;
  jti: string;
  iat?: number;
  exp?: number;
}

export interface RefreshTokenPayload {
  sub: string;
  jti: string;
  deviceId: string;
  iat?: number;
  exp?: number;
}

// Authenticated User (attached to request)
export interface AuthenticatedUser {
  id: string;
  email: string;
  jti: string;
  exp: number;
}

// Request with authenticated user
export interface AuthenticatedRequest extends Request {
  user: AuthenticatedUser;
}

// API Response wrapper
export interface ApiResponse<T> {
  success: boolean;
  code: string;
  message: string;
  data: T;
  timestamp: string;
  path?: string;
}

// Pagination
export interface PaginationMeta {
  page: number;
  limit: number;
  totalItems: number;
  totalPages: number;
  hasNextPage: boolean;
  hasPreviousPage: boolean;
}

export interface PaginatedResponse<T> {
  items: T[];
  meta: PaginationMeta;
}

// User Response
export interface UserResponse {
  id: string;
  email: string;
  fullName: string;
  isActive: boolean;
  roles: string[];
  createdAt: Date;
  updatedAt: Date;
}

// Auth Tokens
export interface AuthTokens {
  accessToken: string;
  refreshToken: string;
  expiresIn: number;
}

// Login Response
export interface LoginResponse {
  user: UserResponse;
  tokens: AuthTokens;
}

// Register Response
export interface RegisterResponse {
  user: UserResponse;
}

// Session Info
export interface SessionInfo {
  refreshTokenId: string;
  deviceId: string;
  ipAddress: string;
  userAgent: string;
  createdAt: Date;
  lastUsedAt: Date;
}

// Device Info extracted from request
export interface DeviceInfo {
  deviceId: string;
  ipAddress: string;
  userAgent: string;
}

// Permission Check Result
export interface PermissionCheckResult {
  hasPermission: boolean;
  requiredPermissions: string[];
  userPermissions: string[];
}

// Config interfaces
export interface JwtConfig {
  accessSecret: string;
  refreshSecret: string;
  accessExpiresIn: string;
  refreshExpiresIn: string;
}

export interface RedisConfig {
  host: string;
  port: number;
  password?: string;
  db?: number;
}

export interface IdentityConfig {
  jwt: JwtConfig;
  redis: RedisConfig;
  bcryptSaltRounds: number;
}

// Seeding interfaces
export interface SeedRole {
  name: string;
  description: string;
  isSystem: boolean;
  permissions: string[];
}

export interface SeedPermission {
  resource: string;
  action: string;
  description: string;
}
