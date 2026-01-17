import { ProfileService } from './profile.service';
import { UpdateProfileDto } from './dto/update-profile.dto';
import { CreateAddressDto } from './dto/address.dto';
import { RegisterShopDto } from './dto/register-shop.dto';
import { StandardResponse } from './interfaces/response.interface';
import { ProfileDocument } from './schemas/profile.schema';
import { AddressDocument } from './schemas/address.schema';
export declare class ProfileController {
    private readonly profileService;
    constructor(profileService: ProfileService);
    getMyProfile(userId: string): Promise<StandardResponse<ProfileDocument>>;
    updateMyProfile(userId: string, dto: UpdateProfileDto): Promise<StandardResponse<ProfileDocument>>;
    registerShop(userId: string, dto: RegisterShopDto): Promise<StandardResponse<ProfileDocument>>;
    updateShop(userId: string, dto: Partial<RegisterShopDto>): Promise<StandardResponse<ProfileDocument>>;
    getMyAddresses(userId: string): Promise<StandardResponse<AddressDocument[]>>;
    addAddress(userId: string, dto: CreateAddressDto): Promise<StandardResponse<AddressDocument>>;
    getAddressById(userId: string, addressId: string): Promise<StandardResponse<AddressDocument>>;
    setDefaultAddress(userId: string, addressId: string): Promise<StandardResponse<AddressDocument>>;
    deleteAddress(userId: string, addressId: string): Promise<StandardResponse<null>>;
}
