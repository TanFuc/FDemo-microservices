import { SetMetadata } from '@nestjs/common';

export const PERMISSION_METADATA_KEY = 'permissions';

export interface PermissionMetadata {
  resource: string;
  action: string;
}

export const RequirePermission = (resource: string, action: string) =>
  SetMetadata(PERMISSION_METADATA_KEY, { resource, action });
