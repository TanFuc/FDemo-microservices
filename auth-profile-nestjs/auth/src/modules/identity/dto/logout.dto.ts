import { ApiPropertyOptional } from '@nestjs/swagger';
import { IsOptional, IsString } from 'class-validator';

export class LogoutDto {
  @ApiPropertyOptional({
    description: 'Device ID to logout from specific device only',
    example: 'device-123-abc',
  })
  @IsString({ message: 'Device ID must be a string' })
  @IsOptional()
  deviceId?: string;
}
