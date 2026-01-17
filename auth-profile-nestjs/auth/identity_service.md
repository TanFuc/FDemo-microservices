# Role
You are a Principal Software Architect and Senior Backend Developer expert in NestJS, TypeORM (PostgreSQL), and Redis.

# Context & Current Status
We are building the **Identity Service** for a large E-commerce Microservices system.
- **Existing Code**: The TypeORM Entities (`User`, `Role`, `Permission`, `RolePermission`, `RefreshToken`) are ALREADY created in `src/modules/identity/entities`. **DO NOT recreate them.**
- **Goal**: Implement the remaining layers: DTOs, Redis Service, Auth Service (Business Logic), Security Strategies, Guards, and Controllers.

# Strict Technical Guidelines
1.  **Type Safety**: STRICTLY NO `any`, `unknown`, or `undefined` (unless optional). Use explicit Interfaces and DTOs.
2.  **Error Handling**: Wrap logic in `try/catch`. Throw specific HTTP Exceptions (`ConflictException`, `UnauthorizedException`, `NotFoundException`) with clear messages.
3.  **Config**: Use `@nestjs/config` to access environment variables (e.g., `process.env.JWT_SECRET`).
4.  **Validation**: Use `class-validator` and `class-transformer` for all DTOs.

---

# PART 1: Data Transfer Objects (DTOs)
**Action:** Create folder `src/modules/identity/dto` and implement these files:

1.  `register.dto.ts`:
    - `email`: @IsEmail, @IsNotEmpty.
    - `password`: @MinLength(8), @IsString.
    - `fullName`: @IsString, @IsNotEmpty.
2.  `login.dto.ts`:
    - `email`: @IsEmail.
    - `password`: @IsString.
3.  `refresh-token.dto.ts`:
    - `refreshToken`: @IsString, @IsNotEmpty.
4.  `logout.dto.ts`:
    - `deviceId`: @IsString (Optional - to revoke specific session).

---

# PART 2: Redis Caching Layer
**Action:** Create `src/modules/identity/services/redis-cache.service.ts`.
**Requirements:**
- Inject `CACHE_MANAGER` (from `@nestjs/cache-manager`) or use `ioredis`.
- Implement these strictly typed methods:
    1.  `cacheUserPermissions(userId: string, permissions: string[])`:
        - Key: `identity:user:${userId}:permissions`
        - TTL: 3600s (1 hour).
    2.  `getUserPermissions(userId: string)`: Returns `Promise<string[] | null>`.
    3.  `blacklistToken(jti: string, ttl: number)`:
        - Key: `identity:blacklist:${jti}`
        - Value: "1".
    4.  `isTokenBlacklisted(jti: string)`: Returns `boolean`.

---

# PART 3: Auth Service (Business Logic)
**Action:** Create `src/modules/identity/services/auth.service.ts`.
**Logic Implementation:**

## 1. Method: `register(dto: RegisterDto)`
- **Step 1**: Check if `userRepository.findOne({ email })`. If yes, throw `ConflictException('Email already exists')`.
- **Step 2**: Hash password using `bcrypt.hash(dto.password, 10)`.
- **Step 3**: Create `User` entity.
- **Step 4**: Assign default Role `CUSTOMER` (Query Role table by name 'CUSTOMER').
- **Step 5**: Save and return User (exclude password).

## 2. Method: `login(dto: LoginDto)`
- **Step 1**: Find User by email (select `password` explicitly). If not found, throw `UnauthorizedException('Invalid credentials')`.
- **Step 2**: Check password `bcrypt.compare()`. If false, throw `UnauthorizedException`.
- **Step 3**: **Fetch Permissions**:
    - Query `User -> UserRoles -> Roles -> RolePermissions -> Permissions`.
    - Flatten to an array of slugs: `['product:create', 'order:read_own']`.
- **Step 4**: **Cache Permissions**: Call `redisCacheService.cacheUserPermissions(user.id, permissions)`.
- **Step 5**: Generate Tokens:
    - `accessToken`: payload `{ sub: user.id, email: user.email }`, expires `15m`.
    - `refreshToken`: payload `{ sub: user.id }`, expires `7d`.
- **Step 6**: Save Refresh Token hash to DB (`RefreshToken` entity) with `deviceId` and `ipAddress` (pass these as args).
- **Return**: `{ user, accessToken, refreshToken }`.

## 3. Method: `refreshTokens(dto: RefreshTokenDto)`
- **Step 1**: Verify `dto.refreshToken` using `jwtService.verify()`.
- **Step 2**: Check DB `RefreshToken` table. If not found or `isRevoked` is true, throw `UnauthorizedException('Token revoked')`.
- **Step 3**: **Token Rotation**:
    - Revoke the old token in DB (set `is_revoked = true`, `replaced_by = new_id`).
    - Generate NEW access and NEW refresh tokens.
    - Save NEW refresh token to DB.
- **Return**: New tokens.

---

# PART 4: Security (Guards & Strategies)
**Action:** Create folder `src/modules/identity/guards` and `strategies`.

1.  **JWT Strategy** (`jwt.strategy.ts`):
    - Extract from Bearer Auth Header.
    - Validate: Check if `identity:blacklist:${payload.jti}` exists in Redis. If yes, throw Unauthorized.

2.  **Permissions Guard** (`permissions.guard.ts`):
    - Use logic:
      ```typescript
      const requiredPermissions = reflector.get<string[]>('permissions', context.getHandler());
      if (!requiredPermissions) return true;
      const { user } = context.switchToHttp().getRequest();
      
      // 1. Try get from Redis
      let userPerms = await redisCacheService.getUserPermissions(user.id);
      
      // 2. If Miss -> Fallback to DB -> Cache to Redis
      if (!userPerms) {
          userPerms = await authService.fetchUserPermissionsFromDB(user.id);
          await redisCacheService.cacheUserPermissions(user.id, userPerms);
      }
      
      // 3. Check logic
      return requiredPermissions.some(p => userPerms.includes(p));
      ```

3.  **Decorator** (`require-permissions.decorator.ts`):
    - `export const RequirePermissions = (...permissions: string[]) => SetMetadata('permissions', permissions);`

---

# PART 5: Auth Controller
**Action:** Create `src/modules/identity/auth.controller.ts`.

- **POST** `/auth/register`: Body `RegisterDto`.
- **POST** `/auth/login`: Body `LoginDto`. Get IP/User-Agent from `@Req()`.
- **POST** `/auth/refresh`: Body `RefreshTokenDto`.
- **POST** `/auth/logout`: Use `@UseGuards(JwtAuthGuard)`. Call service to revoke token & clear Redis perms.
- **GET** `/auth/profile`: `@UseGuards(JwtAuthGuard)`. Return user info.
- **GET** `/auth/check-permission`:
    - Decorators: `@UseGuards(JwtAuthGuard, PermissionsGuard)`, `@RequirePermissions('product:create')`.
    - Return: `{ message: 'You have access' }`.

---

# Instructions for Generation
1.  Analyze the prompt completely.
2.  Generate the code file by file in the exact order: **DTOs -> RedisService -> AuthService -> Strategies/Guards -> Controller**.
3.  Ensure all imports are correct (assuming relative paths).
4.  Do not stop until the Controller is finished.