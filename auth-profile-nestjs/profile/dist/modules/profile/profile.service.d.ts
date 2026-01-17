import { Model, Connection } from 'mongoose';
import { ProfileDocument } from './schemas/profile.schema';
import { AddressDocument } from './schemas/address.schema';
import { UpdateProfileDto } from './dto/update-profile.dto';
import { CreateAddressDto, UpdateAddressDto } from './dto/address.dto';
import { RegisterShopDto } from './dto/register-shop.dto';
export declare class ProfileService {
    private readonly profileModel;
    private readonly addressModel;
    private readonly connection;
    constructor(profileModel: Model<ProfileDocument>, addressModel: Model<AddressDocument>, connection: Connection);
    getOrCreateProfile(userId: string): Promise<ProfileDocument>;
    updateProfile(userId: string, dto: UpdateProfileDto): Promise<ProfileDocument>;
    addAddress(userId: string, dto: CreateAddressDto): Promise<AddressDocument>;
    getAddresses(userId: string): Promise<AddressDocument[]>;
    getAddressById(userId: string, addressId: string): Promise<AddressDocument>;
    updateAddress(userId: string, addressId: string, dto: UpdateAddressDto): Promise<AddressDocument>;
    setDefaultAddress(userId: string, addressId: string): Promise<AddressDocument>;
    deleteAddress(userId: string, addressId: string): Promise<void>;
    registerShop(userId: string, dto: RegisterShopDto): Promise<ProfileDocument>;
    updateShop(userId: string, dto: Partial<RegisterShopDto>): Promise<ProfileDocument>;
}
