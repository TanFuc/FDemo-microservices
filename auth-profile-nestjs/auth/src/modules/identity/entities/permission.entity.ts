import {
  Entity,
  PrimaryGeneratedColumn,
  Column,
  OneToMany,
  Index,
  BeforeInsert,
  BeforeUpdate,
} from 'typeorm';
import { RolePermission } from './role-permission.entity';

@Entity('permissions')
@Index('idx_permissions_resource_action', ['resource', 'action'], { unique: true })
@Index('idx_permissions_slug', ['slug'], { unique: true })
@Index('idx_permissions_category', ['category'])
export class Permission {
  @PrimaryGeneratedColumn()
  id!: number;

  @Column({ type: 'varchar', length: 50 })
  resource!: string; // product, order, user, shop, campaign, report, settings

  @Column({ type: 'varchar', length: 50 })
  action!: string; // create, read, update, delete, manage, approve, export

  @Column({ type: 'varchar', length: 101, unique: true })
  slug!: string; // product:create, order:read_own

  @Column({ name: 'display_name', type: 'varchar', length: 100 })
  displayName!: string;

  @Column({ type: 'text', nullable: true })
  description!: string | null;

  @Column({ type: 'varchar', length: 50, nullable: true })
  category!: string | null; // For grouping in UI

  @Column({ name: 'is_dangerous', type: 'boolean', default: false })
  isDangerous!: boolean; // Requires extra confirmation

  @OneToMany(
    () => RolePermission,
    (rolePermission) => rolePermission.permission,
  )
  rolePermissions!: RolePermission[];

  @BeforeInsert()
  @BeforeUpdate()
  generateSlug(): void {
    this.slug = `${this.resource}:${this.action}`;
  }
}
