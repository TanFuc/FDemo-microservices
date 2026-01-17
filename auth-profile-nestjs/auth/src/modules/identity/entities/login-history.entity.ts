import {
  Entity,
  PrimaryGeneratedColumn,
  Column,
  ManyToOne,
  JoinColumn,
  CreateDateColumn,
  Index,
} from 'typeorm';
import { User } from './user.entity';

@Entity('login_history')
@Index('idx_login_history_user_id_created_at', ['userId', 'createdAt'])
@Index('idx_login_history_status', ['status'])
@Index('idx_login_history_ip_address', ['ipAddress'])
@Index('idx_login_history_created_at', ['createdAt'])
export class LoginHistory {
  @PrimaryGeneratedColumn('uuid')
  id!: string;

  @Column({ name: 'user_id', type: 'uuid' })
  userId!: string;

  @Column({ type: 'varchar', length: 20 })
  status!: string; // SUCCESS, FAILED, BLOCKED, 2FA_REQUIRED, 2FA_SUCCESS, 2FA_FAILED

  @Column({ name: 'failure_reason', type: 'varchar', length: 100, nullable: true })
  failureReason!: string | null; // INVALID_PASSWORD, ACCOUNT_LOCKED, ACCOUNT_SUSPENDED, etc.

  @Column({ name: 'ip_address', type: 'varchar', length: 45 })
  ipAddress!: string;

  @Column({ name: 'user_agent', type: 'varchar', length: 500, nullable: true })
  userAgent!: string | null;

  @Column({ type: 'varchar', length: 100, nullable: true })
  location!: string | null;

  @Column({ name: 'auth_method', type: 'varchar', length: 20 })
  authMethod!: string; // PASSWORD, GOOGLE, FACEBOOK, APPLE, 2FA

  @Column({ name: 'device_type', type: 'varchar', length: 20, nullable: true })
  deviceType!: string | null; // mobile, desktop, tablet

  @Column({ type: 'varchar', length: 100, nullable: true })
  browser!: string | null;

  @Column({ type: 'varchar', length: 100, nullable: true })
  os!: string | null;

  @Column({ type: 'jsonb', nullable: true })
  metadata!: Record<string, any> | null;

  @CreateDateColumn({ name: 'created_at', type: 'timestamptz' })
  createdAt!: Date;

  @ManyToOne(() => User, (user) => user.loginHistory, {
    onDelete: 'CASCADE',
  })
  @JoinColumn({ name: 'user_id' })
  user!: User;
}
