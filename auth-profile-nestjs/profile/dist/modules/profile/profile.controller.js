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
exports.ProfileController = void 0;
const common_1 = require("@nestjs/common");
const profile_service_1 = require("./profile.service");
const update_profile_dto_1 = require("./dto/update-profile.dto");
const address_dto_1 = require("./dto/address.dto");
const register_shop_dto_1 = require("./dto/register-shop.dto");
const user_id_decorator_1 = require("./decorators/user-id.decorator");
let ProfileController = class ProfileController {
    constructor(profileService) {
        this.profileService = profileService;
    }
    async getMyProfile(userId) {
        const profile = await this.profileService.getOrCreateProfile(userId);
        return {
            success: true,
            data: profile,
        };
    }
    async updateMyProfile(userId, dto) {
        const profile = await this.profileService.updateProfile(userId, dto);
        return {
            success: true,
            data: profile,
            message: 'Profile updated successfully',
        };
    }
    async registerShop(userId, dto) {
        const profile = await this.profileService.registerShop(userId, dto);
        return {
            success: true,
            data: profile,
            message: 'Shop registered successfully',
        };
    }
    async updateShop(userId, dto) {
        const profile = await this.profileService.updateShop(userId, dto);
        return {
            success: true,
            data: profile,
            message: 'Shop updated successfully',
        };
    }
    async getMyAddresses(userId) {
        const addresses = await this.profileService.getAddresses(userId);
        return {
            success: true,
            data: addresses,
        };
    }
    async addAddress(userId, dto) {
        const address = await this.profileService.addAddress(userId, dto);
        return {
            success: true,
            data: address,
            message: 'Address added successfully',
        };
    }
    async getAddressById(userId, addressId) {
        const address = await this.profileService.getAddressById(userId, addressId);
        return {
            success: true,
            data: address,
        };
    }
    async setDefaultAddress(userId, addressId) {
        const address = await this.profileService.setDefaultAddress(userId, addressId);
        return {
            success: true,
            data: address,
            message: 'Default address updated successfully',
        };
    }
    async deleteAddress(userId, addressId) {
        await this.profileService.deleteAddress(userId, addressId);
        return {
            success: true,
            data: null,
            message: 'Address deleted successfully',
        };
    }
};
exports.ProfileController = ProfileController;
__decorate([
    (0, common_1.Get)('me'),
    __param(0, (0, user_id_decorator_1.UserId)()),
    __metadata("design:type", Function),
    __metadata("design:paramtypes", [String]),
    __metadata("design:returntype", Promise)
], ProfileController.prototype, "getMyProfile", null);
__decorate([
    (0, common_1.Patch)('me'),
    __param(0, (0, user_id_decorator_1.UserId)()),
    __param(1, (0, common_1.Body)()),
    __metadata("design:type", Function),
    __metadata("design:paramtypes", [String, update_profile_dto_1.UpdateProfileDto]),
    __metadata("design:returntype", Promise)
], ProfileController.prototype, "updateMyProfile", null);
__decorate([
    (0, common_1.Post)('me/shop'),
    __param(0, (0, user_id_decorator_1.UserId)()),
    __param(1, (0, common_1.Body)()),
    __metadata("design:type", Function),
    __metadata("design:paramtypes", [String, register_shop_dto_1.RegisterShopDto]),
    __metadata("design:returntype", Promise)
], ProfileController.prototype, "registerShop", null);
__decorate([
    (0, common_1.Patch)('me/shop'),
    __param(0, (0, user_id_decorator_1.UserId)()),
    __param(1, (0, common_1.Body)()),
    __metadata("design:type", Function),
    __metadata("design:paramtypes", [String, Object]),
    __metadata("design:returntype", Promise)
], ProfileController.prototype, "updateShop", null);
__decorate([
    (0, common_1.Get)('me/addresses'),
    __param(0, (0, user_id_decorator_1.UserId)()),
    __metadata("design:type", Function),
    __metadata("design:paramtypes", [String]),
    __metadata("design:returntype", Promise)
], ProfileController.prototype, "getMyAddresses", null);
__decorate([
    (0, common_1.Post)('me/addresses'),
    __param(0, (0, user_id_decorator_1.UserId)()),
    __param(1, (0, common_1.Body)()),
    __metadata("design:type", Function),
    __metadata("design:paramtypes", [String, address_dto_1.CreateAddressDto]),
    __metadata("design:returntype", Promise)
], ProfileController.prototype, "addAddress", null);
__decorate([
    (0, common_1.Get)('me/addresses/:id'),
    __param(0, (0, user_id_decorator_1.UserId)()),
    __param(1, (0, common_1.Param)('id')),
    __metadata("design:type", Function),
    __metadata("design:paramtypes", [String, String]),
    __metadata("design:returntype", Promise)
], ProfileController.prototype, "getAddressById", null);
__decorate([
    (0, common_1.Patch)('me/addresses/:id/set-default'),
    __param(0, (0, user_id_decorator_1.UserId)()),
    __param(1, (0, common_1.Param)('id')),
    __metadata("design:type", Function),
    __metadata("design:paramtypes", [String, String]),
    __metadata("design:returntype", Promise)
], ProfileController.prototype, "setDefaultAddress", null);
__decorate([
    (0, common_1.Delete)('me/addresses/:id'),
    (0, common_1.HttpCode)(common_1.HttpStatus.OK),
    __param(0, (0, user_id_decorator_1.UserId)()),
    __param(1, (0, common_1.Param)('id')),
    __metadata("design:type", Function),
    __metadata("design:paramtypes", [String, String]),
    __metadata("design:returntype", Promise)
], ProfileController.prototype, "deleteAddress", null);
exports.ProfileController = ProfileController = __decorate([
    (0, common_1.Controller)('profiles'),
    __metadata("design:paramtypes", [profile_service_1.ProfileService])
], ProfileController);
//# sourceMappingURL=profile.controller.js.map