import {
  Controller,
  Get,
  Post,
  Patch,
  Delete,
  Body,
  Param,
  HttpCode,
  HttpStatus,
} from '@nestjs/common';
import { ProfileService } from '../services/profile.service';
import { UpdateProfileDto } from '../dto/update-profile.dto';
import { CreateAddressDto } from '../dto/address.dto';
import { RegisterShopDto } from '../dto/register-shop.dto';
import { UserId } from '../decorators/user-id.decorator';
import { StandardResponse } from '../interfaces/response.interface';
import { Profile, Address } from '../../../generated/client/client';

@Controller('profiles')
export class ProfileController {
  constructor(private readonly profileService: ProfileService) {}

  @Get('me')
  async getMyProfile(
    @UserId() userId: string,
  ): Promise<StandardResponse<Profile>> {
    const profile = await this.profileService.getOrCreateProfile(userId);
    return {
      success: true,
      data: profile,
    };
  }

  @Patch('me')
  async updateMyProfile(
    @UserId() userId: string,
    @Body() dto: UpdateProfileDto,
  ): Promise<StandardResponse<Profile>> {
    const profile = await this.profileService.updateProfile(userId, dto);
    return {
      success: true,
      data: profile,
      message: 'Profile updated successfully',
    };
  }

  @Post('me/shop')
  async registerShop(
    @UserId() userId: string,
    @Body() dto: RegisterShopDto,
  ): Promise<StandardResponse<Profile>> {
    const profile = await this.profileService.registerShop(userId, dto);
    return {
      success: true,
      data: profile,
      message: 'Shop registered successfully',
    };
  }

  @Patch('me/shop')
  async updateShop(
    @UserId() userId: string,
    @Body() dto: Partial<RegisterShopDto>,
  ): Promise<StandardResponse<Profile>> {
    const profile = await this.profileService.updateShop(userId, dto);
    return {
      success: true,
      data: profile,
      message: 'Shop updated successfully',
    };
  }

  @Get('me/addresses')
  async getMyAddresses(
    @UserId() userId: string,
  ): Promise<StandardResponse<Address[]>> {
    const addresses = await this.profileService.getAddresses(userId);
    return {
      success: true,
      data: addresses,
    };
  }

  @Post('me/addresses')
  async addAddress(
    @UserId() userId: string,
    @Body() dto: CreateAddressDto,
  ): Promise<StandardResponse<Address>> {
    const address = await this.profileService.addAddress(userId, dto);
    return {
      success: true,
      data: address,
      message: 'Address added successfully',
    };
  }

  @Get('me/addresses/:id')
  async getAddressById(
    @UserId() userId: string,
    @Param('id') addressId: string,
  ): Promise<StandardResponse<Address>> {
    const address = await this.profileService.getAddressById(userId, addressId);
    return {
      success: true,
      data: address,
    };
  }

  @Patch('me/addresses/:id/set-default')
  async setDefaultAddress(
    @UserId() userId: string,
    @Param('id') addressId: string,
  ): Promise<StandardResponse<Address>> {
    const address = await this.profileService.setDefaultAddress(
      userId,
      addressId,
    );
    return {
      success: true,
      data: address,
      message: 'Default address updated successfully',
    };
  }

  @Delete('me/addresses/:id')
  @HttpCode(HttpStatus.OK)
  async deleteAddress(
    @UserId() userId: string,
    @Param('id') addressId: string,
  ): Promise<StandardResponse<null>> {
    await this.profileService.deleteAddress(userId, addressId);
    return {
      success: true,
      data: null,
      message: 'Address deleted successfully',
    };
  }
}
