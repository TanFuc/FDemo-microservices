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
Object.defineProperty(exports, "__esModule", { value: true });
exports.ProfileSchema = exports.Profile = exports.ShopConfigSchema = exports.ShopConfig = void 0;
const mongoose_1 = require("@nestjs/mongoose");
const mongoose_2 = require("mongoose");
let ShopConfig = class ShopConfig {
};
exports.ShopConfig = ShopConfig;
__decorate([
    (0, mongoose_1.Prop)({ type: String }),
    __metadata("design:type", String)
], ShopConfig.prototype, "shopName", void 0);
__decorate([
    (0, mongoose_1.Prop)(),
    __metadata("design:type", String)
], ShopConfig.prototype, "description", void 0);
__decorate([
    (0, mongoose_1.Prop)(),
    __metadata("design:type", String)
], ShopConfig.prototype, "logoUrl", void 0);
__decorate([
    (0, mongoose_1.Prop)({ type: mongoose_2.Types.ObjectId, ref: 'Address' }),
    __metadata("design:type", mongoose_2.Types.ObjectId)
], ShopConfig.prototype, "pickupAddressId", void 0);
exports.ShopConfig = ShopConfig = __decorate([
    (0, mongoose_1.Schema)({ _id: false })
], ShopConfig);
exports.ShopConfigSchema = mongoose_1.SchemaFactory.createForClass(ShopConfig);
let Profile = class Profile extends mongoose_2.Document {
};
exports.Profile = Profile;
__decorate([
    (0, mongoose_1.Prop)({ required: true }),
    __metadata("design:type", String)
], Profile.prototype, "userId", void 0);
__decorate([
    (0, mongoose_1.Prop)({ required: true }),
    __metadata("design:type", String)
], Profile.prototype, "displayName", void 0);
__decorate([
    (0, mongoose_1.Prop)(),
    __metadata("design:type", String)
], Profile.prototype, "email", void 0);
__decorate([
    (0, mongoose_1.Prop)(),
    __metadata("design:type", String)
], Profile.prototype, "avatarUrl", void 0);
__decorate([
    (0, mongoose_1.Prop)(),
    __metadata("design:type", String)
], Profile.prototype, "bio", void 0);
__decorate([
    (0, mongoose_1.Prop)({ type: exports.ShopConfigSchema }),
    __metadata("design:type", ShopConfig)
], Profile.prototype, "shopConfig", void 0);
exports.Profile = Profile = __decorate([
    (0, mongoose_1.Schema)({ timestamps: true })
], Profile);
exports.ProfileSchema = mongoose_1.SchemaFactory.createForClass(Profile);
exports.ProfileSchema.index({ userId: 1 }, { unique: true });
exports.ProfileSchema.index({ 'shopConfig.shopName': 1 }, {
    unique: true,
    sparse: true,
    partialFilterExpression: { 'shopConfig.shopName': { $exists: true } },
});
//# sourceMappingURL=profile.schema.js.map