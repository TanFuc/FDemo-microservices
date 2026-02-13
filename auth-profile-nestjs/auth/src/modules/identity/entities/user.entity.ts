import {
  Entity,
  PrimaryGeneratedColumn,
  Column,
  CreateDateColumn,
  UpdateDateColumn,
  DeleteDateColumn,
  VersionColumn,
  OneToMany,
  Index,
} from 'typeorm';
import { UserRole } from './user-role.entity';
import { RefreshToken } from './refresh-token.entity';
import { LoginHistory } from './login-history.entity';

@Entity('users')
@Index('idx_users_email', ['email'], { unique: true })
@Index('idx_users_phone', ['phone'], { unique: true, where: 'phone IS NOT NULL' })
@Index('idx_users_status_created_at', ['status', 'createdAt'])
@Index('idx_users_referral_code', ['referralCode'], { unique: true })
@Index('idx_users_google_id', ['googleId'], { where: 'google_id IS NOT NULL' })
@Index('idx_users_facebook_id', ['facebookId'], { where: 'facebook_id IS NOT NULL' })
@Index('idx_users_apple_id', ['appleId'], { where: 'apple_id IS NOT NULL' })
export class User {
  @PrimaryGeneratedColumn('uuid')
  id!: string;

  // ==================== Authentication ====================
  @Column({ type: 'varchar', length: 255, unique: true })
  email!: string;

  @Column({ name: 'email_verified', type: 'boolean', default: false })
  emailVerified!: boolean;

  @Column({ name: 'email_verified_at', type: 'timestamptz', nullable: true })
  emailVerifiedAt!: Date | null;

  @Column({ type: 'varchar', length: 20, nullable: true })
  phone!: string | null;

  @Column({ name: 'phone_verified', type: 'boolean', default: false })
  phoneVerified!: boolean;

  @Column({ name: 'password_hash', type: 'varchar', length: 255, select: false })
  passwordHash!: string;

  @Column({ name: 'password_changed_at', type: 'timestamptz', nullable: true })
  passwordChangedAt!: Date | null;

  // ==================== Profile ====================
  @Column({ name: 'full_name', type: 'varchar', length: 255 })
  fullName!: string;

  @Column({ name: 'display_name', type: 'varchar', length: 100, nullable: true })
  displayName!: string | null;

  @Column({ name: 'avatar_url', type: 'varchar', length: 500, nullable: true })
  avatarUrl!: string | null;

  @Column({ type: 'date', nullable: true })
  birthday!: Date | null;

  @Column({ type: 'varchar', length: 10, nullable: true })
  gender!: string | null; // MALE, FEMALE, OTHER

  // ==================== Account Status ====================
  @Column({ type: 'varchar', length: 20, default: 'ACTIVE' })
  status!: string; // ACTIVE, INACTIVE, SUSPENDED, BANNED, PENDING

  @Column({ name: 'suspension_reason', type: 'text', nullable: true })
  suspensionReason!: string | null;

  @Column({ name: 'suspended_until', type: 'timestamptz', nullable: true })
  suspendedUntil!: Date | null;

  // ==================== Security ====================
  @Column({ name: 'two_factor_enabled', type: 'boolean', default: false })
  twoFactorEnabled!: boolean;

  @Column({
    name: 'two_factor_secret',
    type: 'varchar',
    length: 255,
    nullable: true,
    select: false,
  })
  twoFactorSecret!: string | null;

  @Column({ name: 'backup_codes', type: 'jsonb', nullable: true, select: false })
  backupCodes!: string[] | null;

  @Column({ name: 'failed_login_attempts', type: 'int', default: 0 })
  failedLoginAttempts!: number;

  @Column({ name: 'locked_until', type: 'timestamptz', nullable: true })
  lockedUntil!: Date | null;

  @Column({ name: 'last_login_at', type: 'timestamptz', nullable: true })
  lastLoginAt!: Date | null;

  @Column({ name: 'last_login_ip', type: 'varchar', length: 45, nullable: true })
  lastLoginIp!: string | null;

  @Column({ name: 'login_count', type: 'int', default: 0 })
  loginCount!: number;

  // ==================== Preferences ====================
  @Column({ type: 'varchar', length: 10, default: 'vi' })
  language!: string;

  @Column({ type: 'varchar', length: 3, default: 'VND' })
  currency!: string;

  @Column({ type: 'varchar', length: 50, default: 'Asia/Ho_Chi_Minh' })
  timezone!: string;

  // ==================== Marketing ====================
  @Column({ name: 'accepts_marketing', type: 'boolean', default: false })
  acceptsMarketing!: boolean;

  @Column({ name: 'marketing_opted_in_at', type: 'timestamptz', nullable: true })
  marketingOptedInAt!: Date | null;

  // ==================== Referral ====================
  @Column({ name: 'referral_code', type: 'varchar', length: 20, unique: true })
  referralCode!: string;

  @Column({ name: 'referred_by', type: 'uuid', nullable: true })
  referredBy!: string | null;

  // ==================== OAuth ====================
  @Column({ name: 'google_id', type: 'varchar', length: 100, nullable: true })
  googleId!: string | null;

  @Column({ name: 'facebook_id', type: 'varchar', length: 100, nullable: true })
  facebookId!: string | null;

  @Column({ name: 'apple_id', type: 'varchar', length: 100, nullable: true })
  appleId!: string | null;

  // ==================== Metadata ====================
  @Column({ type: 'jsonb', nullable: true })
  metadata!: Record<string, any> | null;

  // ==================== Timestamps ====================
  @CreateDateColumn({ name: 'created_at', type: 'timestamptz' })
  createdAt!: Date;

  @UpdateDateColumn({ name: 'updated_at', type: 'timestamptz' })
  updatedAt!: Date;

  @DeleteDateColumn({ name: 'deleted_at', type: 'timestamptz' })
  deletedAt!: Date | null;

  // ==================== Version for Optimistic Locking ====================
  @VersionColumn()
  version!: number;

  // ==================== Relations ====================
  @OneToMany(() => UserRole, (userRole) => userRole.user)
  userRoles!: UserRole[];

  @OneToMany(() => RefreshToken, (refreshToken) => refreshToken.user)
  refreshTokens!: RefreshToken[];

  @OneToMany(() => LoginHistory, (loginHistory) => loginHistory.user)
  loginHistory!: LoginHistory[];
}
