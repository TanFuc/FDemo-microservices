import {
  Injectable,
  NotFoundException,
  ConflictException,
} from '@nestjs/common';
import { PrismaService } from '../../../database/prisma.service';
import { Address, Prisma } from '../../../generated/client/client';
import { UpdateProfileDto } from '../dto/update-profile.dto';
import { CreateAddressDto, UpdateAddressDto } from '../dto/address.dto';
import { RegisterShopDto } from '../dto/register-shop.dto';

type ProfileWithShop = Prisma.ProfileGetPayload<{
  include: { shopConfig: true };
}>;

@Injectable()
export class ProfileService {
  constructor(private readonly prisma: PrismaService) {}

  async getOrCreateProfile(userId: string): Promise<ProfileWithShop> {
    let profile = await this.prisma.profile.findUnique({
      where: { userId },
      include: { shopConfig: true },
    });

    if (!profile) {
      const defaultDisplayName = `User-${userId.slice(-4)}`;
      profile = await this.prisma.profile.create({
        data: {
          userId,
          displayName: defaultDisplayName,
        },
        include: { shopConfig: true },
      });
    }

    return profile;
  }

  async createInitialProfile(
    userId: string,
    displayName: string,
    email: string,
  ): Promise<ProfileWithShop> {
    const existingProfile = await this.prisma.profile.findUnique({
      where: { userId },
      include: { shopConfig: true },
    });

    if (existingProfile) {
      return existingProfile;
    }

    return this.prisma.profile.create({
      data: {
        userId,
        displayName: displayName || `User-${userId.slice(-4)}`,
        email,
      },
      include: { shopConfig: true },
    });
  }

  async updateProfile(
    userId: string,
    dto: UpdateProfileDto,
  ): Promise<ProfileWithShop> {
    const profile = await this.getOrCreateProfile(userId);

    // Filter out undefined fields and handle nested objects if necessary
    // For simplicity assuming dto matches data structure or needs mapping
    const data: any = { ...dto };
    delete data.userId; // Prevent userId update if it exists in dto

    return this.prisma.profile.update({
      where: { id: profile.id },
      data: data,
      include: { shopConfig: true },
    });
  }

  async addAddress(
    userId: string,
    dto: CreateAddressDto,
  ): Promise<Address> {
    const profile = await this.getOrCreateProfile(userId);

    const existingAddressCount = await this.prisma.address.count({
      where: { profileId: profile.id },
    });

    let isDefault = dto.isDefault ?? false;

    if (existingAddressCount === 0) {
      isDefault = true;
    }

    if (isDefault && existingAddressCount > 0) {
      await this.prisma.address.updateMany({
        where: { profileId: profile.id },
        data: { isDefault: false },
      });
    }

    return this.prisma.address.create({
      data: {
        profileId: profile.id,
        contactName: dto.contactName,
        phone: dto.phone,
        countryCode: 'VN',
        provinceCode: dto.provinceCode,
        districtCode: dto.districtCode,
        wardCode: dto.wardCode,
        streetAddress: dto.streetLine,
        fullAddress: dto.fullAddress,
        type: dto.type,
        isDefault,
      },
    });
  }

  async getAddresses(userId: string): Promise<Address[]> {
    const profile = await this.getOrCreateProfile(userId);
    return this.prisma.address.findMany({
      where: { profileId: profile.id },
      orderBy: { isDefault: 'desc' },
    });
  }

  async getAddressById(
    userId: string,
    addressId: string,
  ): Promise<Address> {
    const profile = await this.getOrCreateProfile(userId);
    
    const address = await this.prisma.address.findFirst({
      where: {
        id: addressId,
        profileId: profile.id,
      },
    });

    if (!address) {
      throw new NotFoundException('Address not found');
    }

    return address;
  }

  async updateAddress(
    userId: string,
    addressId: string,
    dto: UpdateAddressDto,
  ): Promise<Address> {
    const profile = await this.getOrCreateProfile(userId);
    const address = await this.getAddressById(userId, addressId);

    if (dto.isDefault === true) {
      await this.prisma.address.updateMany({
        where: { profileId: profile.id },
        data: { isDefault: false },
      });
    }

    return this.prisma.address.update({
      where: { id: addressId },
      data: {
        contactName: dto.contactName,
        phone: dto.phone,
        provinceCode: dto.provinceCode,
        districtCode: dto.districtCode,
        wardCode: dto.wardCode,
        streetAddress: dto.streetLine,
        fullAddress: dto.fullAddress,
        type: dto.type,
        isDefault: dto.isDefault,
      },
    });
  }

  async setDefaultAddress(
    userId: string,
    addressId: string,
  ): Promise<Address> {
    const profile = await this.getOrCreateProfile(userId);
    
    // Check if address exists and belongs to user
    const addressExists = await this.prisma.address.findFirst({
      where: { id: addressId, profileId: profile.id }
    });
    
    if (!addressExists) {
      throw new NotFoundException('Address not found');
    }

    return this.prisma.$transaction(async (tx) => {
      await tx.address.updateMany({
        where: { profileId: profile.id },
        data: { isDefault: false },
      });

      return tx.address.update({
        where: { id: addressId },
        data: { isDefault: true },
      });
    });
  }

  async deleteAddress(userId: string, addressId: string): Promise<void> {
    const profile = await this.getOrCreateProfile(userId);
    const address = await this.getAddressById(userId, addressId);

    const wasDefault = address.isDefault;

    await this.prisma.address.delete({
      where: { id: addressId },
    });

    if (wasDefault) {
      const nextAddress = await this.prisma.address.findFirst({
        where: { profileId: profile.id },
        orderBy: { createdAt: 'desc' },
      });

      if (nextAddress) {
        await this.prisma.address.update({
          where: { id: nextAddress.id },
          data: { isDefault: true },
        });
      }
    }
  }

  async registerShop(
    userId: string,
    dto: RegisterShopDto,
  ): Promise<ProfileWithShop> {
    const existingShop = await this.prisma.shopConfig.findFirst({
      where: {
        shopName: dto.shopName,
        profile: {
          userId: { not: userId },
        },
      },
    });

    if (existingShop) {
      throw new ConflictException('Shop name is already taken');
    }

    const profile = await this.getOrCreateProfile(userId);

    if (profile.shopConfig) {
       throw new ConflictException('User already has a shop');
    }

    await this.prisma.shopConfig.create({
      data: {
        profileId: profile.id,
        shopName: dto.shopName,
        description: dto.description || '',
        logoUrl: dto.logoUrl || '',
      }
    });

    return this.getOrCreateProfile(userId);
  }

  async updateShop(
    userId: string,
    dto: Partial<RegisterShopDto>,
  ): Promise<ProfileWithShop> {
    const profile = await this.getOrCreateProfile(userId);

    if (!profile.shopConfig) {
      throw new NotFoundException('Shop not registered');
    }

    const shopConfig = profile.shopConfig as any; // Cast to any to access properties safely or define type better

    if (dto.shopName && dto.shopName !== shopConfig.shopName) {
      const existingShop = await this.prisma.shopConfig.findFirst({
        where: {
          shopName: dto.shopName,
          profile: {
             userId: { not: userId }
          }
        },
      });

      if (existingShop) {
        throw new ConflictException('Shop name is already taken');
      }
    }

    await this.prisma.shopConfig.update({
      where: { id: shopConfig.id },
      data: dto,
    });

    return this.getOrCreateProfile(userId);
  }
}
