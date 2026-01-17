"use strict";
var __decorate = (this && this.__decorate) || function (decorators, target, key, desc) {
    var c = arguments.length, r = c < 3 ? target : desc === null ? desc = Object.getOwnPropertyDescriptor(target, key) : desc, d;
    if (typeof Reflect === "object" && typeof Reflect.decorate === "function") r = Reflect.decorate(decorators, target, key, desc);
    else for (var i = decorators.length - 1; i >= 0; i--) if (d = decorators[i]) r = (c < 3 ? d(r) : c > 3 ? d(target, key, r) : d(target, key)) || r;
    return c > 3 && r && Object.defineProperty(target, key, r), r;
};
var __metadata = (this && this.__metadata) || function (k, v) {
    if (typeof Reflect === "object" && typeof Reflect.metadata === "function") return Reflect.metadata(k, v);
};
var __param = (this && this.__param) || function (paramIndex, decorator) {
    return function (target, key) { decorator(target, key, paramIndex); }
};
Object.defineProperty(exports, "__esModule", { value: true });
exports.ProfileService = void 0;
const common_1 = require("@nestjs/common");
const mongoose_1 = require("@nestjs/mongoose");
const mongoose_2 = require("mongoose");
const profile_schema_1 = require("./schemas/profile.schema");
const address_schema_1 = require("./schemas/address.schema");
let ProfileService = class ProfileService {
    constructor(profileModel, addressModel, connection) {
        this.profileModel = profileModel;
        this.addressModel = addressModel;
        this.connection = connection;
    }
    async getOrCreateProfile(userId) {
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
    async updateProfile(userId, dto) {
        const profile = await this.getOrCreateProfile(userId);
        Object.assign(profile, dto);
        await profile.save();
        return profile;
    }
    async addAddress(userId, dto) {
        const existingAddressCount = await this.addressModel.countDocuments({
            userId,
        });
        let isDefault = dto.isDefault ?? false;
        if (existingAddressCount === 0) {
            isDefault = true;
        }
        if (isDefault && existingAddressCount > 0) {
            await this.addressModel.updateMany({ userId }, { $set: { isDefault: false } });
        }
        const address = await this.addressModel.create({
            ...dto,
            userId,
            isDefault,
        });
        return address;
    }
    async getAddresses(userId) {
        return this.addressModel.find({ userId }).sort({ isDefault: -1 }).exec();
    }
    async getAddressById(userId, addressId) {
        if (!mongoose_2.Types.ObjectId.isValid(addressId)) {
            throw new common_1.NotFoundException('Address not found');
        }
        const address = await this.addressModel
            .findOne({
            _id: new mongoose_2.Types.ObjectId(addressId),
            userId,
        })
            .exec();
        if (!address) {
            throw new common_1.NotFoundException('Address not found');
        }
        return address;
    }
    async updateAddress(userId, addressId, dto) {
        const address = await this.getAddressById(userId, addressId);
        if (dto.isDefault === true) {
            await this.addressModel.updateMany({ userId }, { $set: { isDefault: false } });
        }
        Object.assign(address, dto);
        await address.save();
        return address;
    }
    async setDefaultAddress(userId, addressId) {
        if (!mongoose_2.Types.ObjectId.isValid(addressId)) {
            throw new common_1.NotFoundException('Address not found');
        }
        const session = await this.connection.startSession();
        session.startTransaction();
        try {
            await this.addressModel.updateMany({ userId }, { $set: { isDefault: false } }, { session });
            const address = await this.addressModel
                .findOneAndUpdate({ _id: new mongoose_2.Types.ObjectId(addressId), userId }, { $set: { isDefault: true } }, { new: true, session })
                .exec();
            if (!address) {
                await session.abortTransaction();
                throw new common_1.NotFoundException('Address not found or does not belong to user');
            }
            await session.commitTransaction();
            return address;
        }
        catch (error) {
            await session.abortTransaction();
            throw error;
        }
        finally {
            session.endSession();
        }
    }
    async deleteAddress(userId, addressId) {
        if (!mongoose_2.Types.ObjectId.isValid(addressId)) {
            throw new common_1.NotFoundException('Address not found');
        }
        const address = await this.addressModel
            .findOne({
            _id: new mongoose_2.Types.ObjectId(addressId),
            userId,
        })
            .exec();
        if (!address) {
            throw new common_1.NotFoundException('Address not found');
        }
        const wasDefault = address.isDefault;
        await this.addressModel.deleteOne({
            _id: new mongoose_2.Types.ObjectId(addressId),
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
    async registerShop(userId, dto) {
        const existingShop = await this.profileModel
            .findOne({
            'shopConfig.shopName': dto.shopName,
            userId: { $ne: userId },
        })
            .exec();
        if (existingShop) {
            throw new common_1.ConflictException('Shop name is already taken');
        }
        const profile = await this.getOrCreateProfile(userId);
        profile.shopConfig = {
            shopName: dto.shopName,
            description: dto.description || '',
            logoUrl: dto.logoUrl || '',
            pickupAddressId: undefined,
        };
        await profile.save();
        return profile;
    }
    async updateShop(userId, dto) {
        const profile = await this.profileModel.findOne({ userId }).exec();
        if (!profile) {
            throw new common_1.NotFoundException('Profile not found');
        }
        if (!profile.shopConfig) {
            throw new common_1.NotFoundException('Shop not registered');
        }
        if (dto.shopName && dto.shopName !== profile.shopConfig.shopName) {
            const existingShop = await this.profileModel
                .findOne({
                'shopConfig.shopName': dto.shopName,
                userId: { $ne: userId },
            })
                .exec();
            if (existingShop) {
                throw new common_1.ConflictException('Shop name is already taken');
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
};
exports.ProfileService = ProfileService;
exports.ProfileService = ProfileService = __decorate([
    (0, common_1.Injectable)(),
    __param(0, (0, mongoose_1.InjectModel)(profile_schema_1.Profile.name)),
    __param(1, (0, mongoose_1.InjectModel)(address_schema_1.Address.name)),
    __param(2, (0, mongoose_1.InjectConnection)()),
    __metadata("design:paramtypes", [mongoose_2.Model,
        mongoose_2.Model,
        mongoose_2.Connection])
], ProfileService);
//# sourceMappingURL=profile.service.js.map