import {
  Injectable,
  NotFoundException,
  ConflictException,
} from '@nestjs/common';
import { InjectModel, InjectConnection } from '@nestjs/mongoose';
import { Model, Connection, Types } from 'mongoose';
import { Profile, ProfileDocument } from './schemas/profile.schema';
import { Address, AddressDocument } from './schemas/address.schema';
import { UpdateProfileDto } from './dto/update-profile.dto';
import { CreateAddressDto, UpdateAddressDto } from './dto/address.dto';
import { RegisterShopDto } from './dto/register-shop.dto';

@Injectable()
export class ProfileService {
  constructor(
    @InjectModel(Profile.name)
    private readonly profileModel: Model<ProfileDocument>,
    @InjectModel(Address.name)
    private readonly addressModel: Model<AddressDocument>,
    @InjectConnection()
    private readonly connection: Connection,
  ) {}

  async getOrCreateProfile(userId: string): Promise<ProfileDocument> {
    let profile = await this.profileModel.findOne({ userId }).exec();

    if (!profile) {
      const defaultDisplayName = `User-${userId.slice(-4)}`;
      profile = await this.profileModel.create({
        userId,
        displayName: defaultDisplayName,
      });
    }

    return profile;
  }

  async createInitialProfile(
    userId: string,
    displayName: string,
    email: string,
  ): Promise<ProfileDocument> {
    // Check if profile already exists
    const existingProfile = await this.profileModel.findOne({ userId }).exec();

    if (existingProfile) {
      // Profile already exists, skip creation
      return existingProfile;
    }

    // Create new profile with provided info
    const profile = await this.profileModel.create({
      userId,
      displayName: displayName || `User-${userId.slice(-4)}`,
      email,
    });

    return profile;
  }

  async updateProfile(
    userId: string,
    dto: UpdateProfileDto,
  ): Promise<ProfileDocument> {
    const profile = await this.getOrCreateProfile(userId);

    Object.assign(profile, dto);
    await profile.save();

    return profile;
  }

  async addAddress(
    userId: string,
    dto: CreateAddressDto,
  ): Promise<AddressDocument> {
    const existingAddressCount = await this.addressModel.countDocuments({
      userId,
    });

    let isDefault = dto.isDefault ?? false;

    if (existingAddressCount === 0) {
      isDefault = true;
    }

    if (isDefault && existingAddressCount > 0) {
      await this.addressModel.updateMany(
        { userId },
        { $set: { isDefault: false } },
      );
    }

    const address = await this.addressModel.create({
      ...dto,
      userId,
      isDefault,
    });

    return address;
  }

  async getAddresses(userId: string): Promise<AddressDocument[]> {
    return this.addressModel.find({ userId }).sort({ isDefault: -1 }).exec();
  }

  async getAddressById(
    userId: string,
    addressId: string,
  ): Promise<AddressDocument> {
    if (!Types.ObjectId.isValid(addressId)) {
      throw new NotFoundException('Address not found');
    }

    const address = await this.addressModel
      .findOne({
        _id: new Types.ObjectId(addressId),
        userId,
      })
      .exec();

    if (!address) {
      throw new NotFoundException('Address not found');
    }

    return address;
  }

  async updateAddress(
    userId: string,
    addressId: string,
    dto: UpdateAddressDto,
  ): Promise<AddressDocument> {
    const address = await this.getAddressById(userId, addressId);

    if (dto.isDefault === true) {
      await this.addressModel.updateMany(
        { userId },
        { $set: { isDefault: false } },
      );
    }

    Object.assign(address, dto);
    await address.save();

    return address;
  }

  async setDefaultAddress(
    userId: string,
    addressId: string,
  ): Promise<AddressDocument> {
    if (!Types.ObjectId.isValid(addressId)) {
      throw new NotFoundException('Address not found');
    }

    const session = await this.connection.startSession();
    session.startTransaction();

    try {
      await this.addressModel.updateMany(
        { userId },
        { $set: { isDefault: false } },
        { session },
      );

      const address = await this.addressModel
        .findOneAndUpdate(
          { _id: new Types.ObjectId(addressId), userId },
          { $set: { isDefault: true } },
          { new: true, session },
        )
        .exec();

      if (!address) {
        await session.abortTransaction();
        throw new NotFoundException(
          'Address not found or does not belong to user',
        );
      }

      await session.commitTransaction();
      return address;
    } catch (error) {
      await session.abortTransaction();
      throw error;
    } finally {
      session.endSession();
    }
  }

  async deleteAddress(userId: string, addressId: string): Promise<void> {
    if (!Types.ObjectId.isValid(addressId)) {
      throw new NotFoundException('Address not found');
    }

    const address = await this.addressModel
      .findOne({
        _id: new Types.ObjectId(addressId),
        userId,
      })
      .exec();

    if (!address) {
      throw new NotFoundException('Address not found');
    }

    const wasDefault = address.isDefault;
    await this.addressModel.deleteOne({
      _id: new Types.ObjectId(addressId),
    });

    if (wasDefault) {
      const nextAddress = await this.addressModel
        .findOne({ userId })
        .sort({ createdAt: -1 })
        .exec();

      if (nextAddress) {
        nextAddress.isDefault = true;
        await nextAddress.save();
      }
    }
  }

  async registerShop(
    userId: string,
    dto: RegisterShopDto,
  ): Promise<ProfileDocument> {
    const existingShop = await this.profileModel
      .findOne({
        'shopConfig.shopName': dto.shopName,
        userId: { $ne: userId },
      })
      .exec();

    if (existingShop) {
      throw new ConflictException('Shop name is already taken');
    }

    const profile = await this.getOrCreateProfile(userId);

    profile.shopConfig = {
      shopName: dto.shopName,
      description: dto.description || '',
      logoUrl: dto.logoUrl || '',
      pickupAddressId: undefined as unknown as Types.ObjectId,
    };

    await profile.save();

    return profile;
  }

  async updateShop(
    userId: string,
    dto: Partial<RegisterShopDto>,
  ): Promise<ProfileDocument> {
    const profile = await this.profileModel.findOne({ userId }).exec();

    if (!profile) {
      throw new NotFoundException('Profile not found');
    }

    if (!profile.shopConfig) {
      throw new NotFoundException('Shop not registered');
    }

    if (dto.shopName && dto.shopName !== profile.shopConfig.shopName) {
      const existingShop = await this.profileModel
        .findOne({
          'shopConfig.shopName': dto.shopName,
          userId: { $ne: userId },
        })
        .exec();

      if (existingShop) {
        throw new ConflictException('Shop name is already taken');
      }

      profile.shopConfig.shopName = dto.shopName;
    }

    if (dto.description !== undefined) {
      profile.shopConfig.description = dto.description;
    }

    if (dto.logoUrl !== undefined) {
      profile.shopConfig.logoUrl = dto.logoUrl;
    }

    await profile.save();

    return profile;
  }
}
