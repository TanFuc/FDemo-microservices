import {
  Controller,
  Get,
  Param,
  Headers,
  UnauthorizedException,
  NotFoundException,
} from '@nestjs/common';
import { ProfileService } from '../services/profile.service';

// Internal service key header
const INTERNAL_SERVICE_KEY_HEADER = 'x-internal-service-key';

interface UserInfoResponse {
  userId: string;
  displayName: string;
  avatarUrl: string;
  email?: string;
}

@Controller('internal')
export class InternalController {
  private readonly serviceKey: string;

  constructor(private readonly profileService: ProfileService) {
    this.serviceKey = process.env.INTERNAL_SERVICE_KEY || '';
  }

  @Get('users/:userId')
  async getUserInfo(
    @Param('userId') userId: string,
    @Headers(INTERNAL_SERVICE_KEY_HEADER) serviceKey: string,
  ): Promise<UserInfoResponse> {
    // Validate internal service key
    if (this.serviceKey && serviceKey !== this.serviceKey) {
      throw new UnauthorizedException('Invalid internal service key');
    }

    try {
      const profile = await this.profileService.getOrCreateProfile(userId);

      return {
        userId: profile.userId,
        displayName: profile.displayName || `User-${userId.slice(-4)}`,
        avatarUrl: profile.avatarUrl || '',
        email: profile.email,
      };
    } catch (error) {
      // Return default info if profile creation fails
      return {
        userId: userId,
        displayName: `User-${userId.slice(-4)}`,
        avatarUrl: '',
      };
    }
  }

  @Get('health')
  healthCheck(): { status: string } {
    return { status: 'ok' };
  }
}
