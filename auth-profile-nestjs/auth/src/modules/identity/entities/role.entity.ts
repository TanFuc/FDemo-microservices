import {
  Entity,
  PrimaryGeneratedColumn,
  Column,
  CreateDateColumn,
  OneToMany,
  Index,
} from 'typeorm';
import { RolePermission } from './role-permission.entity';
import { UserRole } from './user-role.entity';

@Entity('roles')
@Index('idx_roles_name', ['name'], { unique: true })
@Index('idx_roles_is_default', ['isDefault'])
@Index('idx_roles_priority', ['priority'])
export class Role {
  @PrimaryGeneratedColumn()
  id!: number;

  @Column({ type: 'varchar', length: 50, unique: true })
  name!: string; // CUSTOMER, SELLER, ADMIN, SUPER_ADMIN, SUPPORT, FINANCE, WAREHOUSE

  @Column({ name: 'display_name', type: 'varchar', length: 100 })
  displayName!: string;

  @Column({ type: 'text', nullable: true })
  description!: string | null;

  @Column({ name: 'is_system', type: 'boolean', default: false })
  isSystem!: boolean; // Cannot be deleted

  @Column({ name: 'is_default', type: 'boolean', default: false })
  isDefault!: boolean; // Auto-assigned to new users

  @Column({ type: 'int', default: 0 })
  priority!: number; // Higher = more privileges

  @Column({ type: 'jsonb', nullable: true })
  metadata!: Record<string, any> | null;

  @CreateDateColumn({ name: 'created_at', type: 'timestamptz' })
  createdAt!: Date;

  @OneToMany(() => RolePermission, (rolePermission) => rolePermission.role)
  rolePermissions!: RolePermission[];

  @OneToMany(() => UserRole, (userRole) => userRole.role)
  userRoles!: UserRole[];
}
