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
exports.ProfileSchema = exports.Profile = exports.IdentityDocument = exports.ProfileStats = exports.ShopConfigSchema = exports.ShopConfig = exports.SocialLinks = exports.OperatingHours = exports.ShopPolicies = exports.BankAccount = exports.UserPreferences = exports.PrivacySettings = exports.NotificationPreferences = void 0;
const mongoose_1 = require("@nestjs/mongoose");
const mongoose_2 = require("mongoose");
let NotificationPreferences = class NotificationPreferences {
};
exports.NotificationPreferences = NotificationPreferences;
__decorate([
    (0, mongoose_1.Prop)({ type: Object, default: { orders: true, promotions: true, news: false } }),
    __metadata("design:type", Object)
], NotificationPreferences.prototype, "email", void 0);
__decorate([
    (0, mongoose_1.Prop)({ type: Object, default: { orders: true, promotions: true, chat: true } }),
    __metadata("design:type", Object)
], NotificationPreferences.prototype, "push", void 0);
__decorate([
    (0, mongoose_1.Prop)({ type: Object, default: { orders: true, otp: true } }),
    __metadata("design:type", Object)
], NotificationPreferences.prototype, "sms", void 0);
exports.NotificationPreferences = NotificationPreferences = __decorate([
    (0, mongoose_1.Schema)({ _id: false })
], NotificationPreferences);
let PrivacySettings = class PrivacySettings {
};
exports.PrivacySettings = PrivacySettings;
__decorate([
    (0, mongoose_1.Prop)({ default: true }),
    __metadata("design:type", Boolean)
], PrivacySettings.prototype, "showOnlineStatus", void 0);
__decorate([
    (0, mongoose_1.Prop)({ default: true }),
    __metadata("design:type", Boolean)
], PrivacySettings.prototype, "showLastSeen", void 0);
__decorate([
    (0, mongoose_1.Prop)({ type: String, enum: ['EVERYONE', 'FOLLOWING', 'NOBODY'], default: 'EVERYONE' }),
    __metadata("design:type", String)
], PrivacySettings.prototype, "allowMessages", void 0);
exports.PrivacySettings = PrivacySettings = __decorate([
    (0, mongoose_1.Schema)({ _id: false })
], PrivacySettings);
let UserPreferences = class UserPreferences {
};
exports.UserPreferences = UserPreferences;
__decorate([
    (0, mongoose_1.Prop)({ default: 'vi' }),
    __metadata("design:type", String)
], UserPreferences.prototype, "language", void 0);
__decorate([
    (0, mongoose_1.Prop)({ default: 'VND' }),
    __metadata("design:type", String)
], UserPreferences.prototype, "currency", void 0);
__decorate([
    (0, mongoose_1.Prop)({ default: 'Asia/Ho_Chi_Minh' }),
    __metadata("design:type", String)
], UserPreferences.prototype, "timezone", void 0);
__decorate([
    (0, mongoose_1.Prop)({ type: String, enum: ['METRIC', 'IMPERIAL'], default: 'METRIC' }),
    __metadata("design:type", String)
], UserPreferences.prototype, "measurementUnit", void 0);
__decorate([
    (0, mongoose_1.Prop)({ type: NotificationPreferences }),
    __metadata("design:type", NotificationPreferences)
], UserPreferences.prototype, "notifications", void 0);
__decorate([
    (0, mongoose_1.Prop)({ type: PrivacySettings }),
    __metadata("design:type", PrivacySettings)
], UserPreferences.prototype, "privacy", void 0);
exports.UserPreferences = UserPreferences = __decorate([
    (0, mongoose_1.Schema)({ _id: false })
], UserPreferences);
let BankAccount = class BankAccount {
};
exports.BankAccount = BankAccount;
__decorate([
    (0, mongoose_1.Prop)(),
    __metadata("design:type", String)
], BankAccount.prototype, "bankName", void 0);
__decorate([
    (0, mongoose_1.Prop)(),
    __metadata("design:type", String)
], BankAccount.prototype, "accountNumber", void 0);
__decorate([
    (0, mongoose_1.Prop)(),
    __metadata("design:type", String)
], BankAccount.prototype, "accountHolder", void 0);
__decorate([
    (0, mongoose_1.Prop)(),
    __metadata("design:type", String)
], BankAccount.prototype, "branch", void 0);
exports.BankAccount = BankAccount = __decorate([
    (0, mongoose_1.Schema)({ _id: false })
], BankAccount);
let ShopPolicies = class ShopPolicies {
};
exports.ShopPolicies = ShopPolicies;
__decorate([
    (0, mongoose_1.Prop)({ type: String }),
    __metadata("design:type", String)
], ShopPolicies.prototype, "returnPolicy", void 0);
__decorate([
    (0, mongoose_1.Prop)({ type: String }),
    __metadata("design:type", String)
], ShopPolicies.prototype, "shippingPolicy", void 0);
exports.ShopPolicies = ShopPolicies = __decorate([
    (0, mongoose_1.Schema)({ _id: false })
], ShopPolicies);
let OperatingHours = class OperatingHours {
};
exports.OperatingHours = OperatingHours;
__decorate([
    (0, mongoose_1.Prop)({ type: String }),
    __metadata("design:type", String)
], OperatingHours.prototype, "open", void 0);
__decorate([
    (0, mongoose_1.Prop)({ type: String }),
    __metadata("design:type", String)
], OperatingHours.prototype, "close", void 0);
__decorate([
    (0, mongoose_1.Prop)({ default: false }),
    __metadata("design:type", Boolean)
], OperatingHours.prototype, "closed", void 0);
exports.OperatingHours = OperatingHours = __decorate([
    (0, mongoose_1.Schema)({ _id: false })
], OperatingHours);
let SocialLinks = class SocialLinks {
};
exports.SocialLinks = SocialLinks;
__decorate([
    (0, mongoose_1.Prop)(),
    __metadata("design:type", String)
], SocialLinks.prototype, "facebook", void 0);
__decorate([
    (0, mongoose_1.Prop)(),
    __metadata("design:type", String)
], SocialLinks.prototype, "instagram", void 0);
__decorate([
    (0, mongoose_1.Prop)(),
    __metadata("design:type", String)
], SocialLinks.prototype, "tiktok", void 0);
__decorate([
    (0, mongoose_1.Prop)(),
    __metadata("design:type", String)
], SocialLinks.prototype, "youtube", void 0);
__decorate([
    (0, mongoose_1.Prop)(),
    __metadata("design:type", String)
], SocialLinks.prototype, "website", void 0);
exports.SocialLinks = SocialLinks = __decorate([
    (0, mongoose_1.Schema)({ _id: false })
], SocialLinks);
let ShopConfig = class ShopConfig {
};
exports.ShopConfig = ShopConfig;
__decorate([
    (0, mongoose_1.Prop)({ type: String }),
    __metadata("design:type", String)
], ShopConfig.prototype, "shopId", void 0);
__decorate([
    (0, mongoose_1.Prop)({ type: String }),
    __metadata("design:type", String)
], ShopConfig.prototype, "shopName", void 0);
__decorate([
    (0, mongoose_1.Prop)({ type: String }),
    __metadata("design:type", String)
], ShopConfig.prototype, "shopSlug", void 0);
__decorate([
    (0, mongoose_1.Prop)(),
    __metadata("design:type", String)
], ShopConfig.prototype, "description", void 0);
__decorate([
    (0, mongoose_1.Prop)(),
    __metadata("design:type", String)
], ShopConfig.prototype, "logoUrl", void 0);
__decorate([
    (0, mongoose_1.Prop)(),
    __metadata("design:type", String)
], ShopConfig.prototype, "bannerUrl", void 0);
__decorate([
    (0, mongoose_1.Prop)({ type: mongoose_2.Types.ObjectId, ref: 'Address' }),
    __metadata("design:type", mongoose_2.Types.ObjectId)
], ShopConfig.prototype, "pickupAddressId", void 0);
__decorate([
    (0, mongoose_1.Prop)({ type: String, enum: ['INDIVIDUAL', 'BUSINESS'], default: 'INDIVIDUAL' }),
    __metadata("design:type", String)
], ShopConfig.prototype, "businessType", void 0);
__decorate([
    (0, mongoose_1.Prop)(),
    __metadata("design:type", String)
], ShopConfig.prototype, "businessLicense", void 0);
__decorate([
    (0, mongoose_1.Prop)(),
    __metadata("design:type", String)
], ShopConfig.prototype, "taxId", void 0);
__decorate([
    (0, mongoose_1.Prop)({ type: BankAccount }),
    __metadata("design:type", BankAccount)
], ShopConfig.prototype, "bankAccount", void 0);
__decorate([
    (0, mongoose_1.Prop)({ type: ShopPolicies }),
    __metadata("design:type", ShopPolicies)
], ShopConfig.prototype, "policies", void 0);
__decorate([
    (0, mongoose_1.Prop)({ type: Object }),
    __metadata("design:type", Object)
], ShopConfig.prototype, "operatingHours", void 0);
__decorate([
    (0, mongoose_1.Prop)({ type: SocialLinks }),
    __metadata("design:type", SocialLinks)
], ShopConfig.prototype, "socialLinks", void 0);
__decorate([
    (0, mongoose_1.Prop)({ type: String, enum: ['PENDING', 'VERIFIED', 'REJECTED'], default: 'PENDING' }),
    __metadata("design:type", String)
], ShopConfig.prototype, "verificationStatus", void 0);
__decorate([
    (0, mongoose_1.Prop)({ type: Date }),
    __metadata("design:type", Date)
], ShopConfig.prototype, "verifiedAt", void 0);
__decorate([
    (0, mongoose_1.Prop)({ default: 0 }),
    __metadata("design:type", Number)
], ShopConfig.prototype, "rating", void 0);
__decorate([
    (0, mongoose_1.Prop)({ default: 0 }),
    __metadata("design:type", Number)
], ShopConfig.prototype, "totalReviews", void 0);
__decorate([
    (0, mongoose_1.Prop)({ default: 0 }),
    __metadata("design:type", Number)
], ShopConfig.prototype, "totalProducts", void 0);
__decorate([
    (0, mongoose_1.Prop)({ default: 0 }),
    __metadata("design:type", Number)
], ShopConfig.prototype, "totalOrders", void 0);
__decorate([
    (0, mongoose_1.Prop)({ default: 0 }),
    __metadata("design:type", Number)
], ShopConfig.prototype, "totalFollowers", void 0);
__decorate([
    (0, mongoose_1.Prop)({ type: Date }),
    __metadata("design:type", Date)
], ShopConfig.prototype, "joinedAt", void 0);
exports.ShopConfig = ShopConfig = __decorate([
    (0, mongoose_1.Schema)({ _id: false })
], ShopConfig);
exports.ShopConfigSchema = mongoose_1.SchemaFactory.createForClass(ShopConfig);
let ProfileStats = class ProfileStats {
};
exports.ProfileStats = ProfileStats;
__decorate([
    (0, mongoose_1.Prop)({ default: 0 }),
    __metadata("design:type", Number)
], ProfileStats.prototype, "totalOrders", void 0);
__decorate([
    (0, mongoose_1.Prop)({ default: 0 }),
    __metadata("design:type", Number)
], ProfileStats.prototype, "completedOrders", void 0);
__decorate([
    (0, mongoose_1.Prop)({ default: 0 }),
    __metadata("design:type", Number)
], ProfileStats.prototype, "cancelledOrders", void 0);
__decorate([
    (0, mongoose_1.Prop)({ default: 0 }),
    __metadata("design:type", Number)
], ProfileStats.prototype, "totalSpent", void 0);
__decorate([
    (0, mongoose_1.Prop)({ default: 0 }),
    __metadata("design:type", Number)
], ProfileStats.prototype, "averageOrderValue", void 0);
__decorate([
    (0, mongoose_1.Prop)({ default: 0 }),
    __metadata("design:type", Number)
], ProfileStats.prototype, "totalReviews", void 0);
__decorate([
    (0, mongoose_1.Prop)({ default: 0 }),
    __metadata("design:type", Number)
], ProfileStats.prototype, "wishlistCount", void 0);
__decorate([
    (0, mongoose_1.Prop)({ default: 0 }),
    __metadata("design:type", Number)
], ProfileStats.prototype, "cartItemCount", void 0);
exports.ProfileStats = ProfileStats = __decorate([
    (0, mongoose_1.Schema)({ _id: false })
], ProfileStats);
let IdentityDocument = class IdentityDocument {
};
exports.IdentityDocument = IdentityDocument;
__decorate([
    (0, mongoose_1.Prop)({ type: String, enum: ['ID_CARD', 'PASSPORT', 'DRIVER_LICENSE'] }),
    __metadata("design:type", String)
], IdentityDocument.prototype, "type", void 0);
__decorate([
    (0, mongoose_1.Prop)(),
    __metadata("design:type", String)
], IdentityDocument.prototype, "number", void 0);
__decorate([
    (0, mongoose_1.Prop)(),
    __metadata("design:type", String)
], IdentityDocument.prototype, "frontImage", void 0);
__decorate([
    (0, mongoose_1.Prop)(),
    __metadata("design:type", String)
], IdentityDocument.prototype, "backImage", void 0);
__decorate([
    (0, mongoose_1.Prop)(),
    __metadata("design:type", String)
], IdentityDocument.prototype, "selfieImage", void 0);
__decorate([
    (0, mongoose_1.Prop)({ type: Date }),
    __metadata("design:type", Date)
], IdentityDocument.prototype, "verifiedAt", void 0);
exports.IdentityDocument = IdentityDocument = __decorate([
    (0, mongoose_1.Schema)({ _id: false })
], IdentityDocument);
let Profile = class Profile extends mongoose_2.Document {
};
exports.Profile = Profile;
__decorate([
    (0, mongoose_1.Prop)({ required: true, unique: true, index: true }),
    __metadata("design:type", String)
], Profile.prototype, "userId", void 0);
__decorate([
    (0, mongoose_1.Prop)({ required: true, maxlength: 100 }),
    __metadata("design:type", String)
], Profile.prototype, "displayName", void 0);
__decorate([
    (0, mongoose_1.Prop)({ maxlength: 255 }),
    __metadata("design:type", String)
], Profile.prototype, "email", void 0);
__decorate([
    (0, mongoose_1.Prop)({ maxlength: 20 }),
    __metadata("design:type", String)
], Profile.prototype, "phone", void 0);
__decorate([
    (0, mongoose_1.Prop)({ maxlength: 500 }),
    __metadata("design:type", String)
], Profile.prototype, "avatarUrl", void 0);
__decorate([
    (0, mongoose_1.Prop)({ maxlength: 500 }),
    __metadata("design:type", String)
], Profile.prototype, "coverUrl", void 0);
__decorate([
    (0, mongoose_1.Prop)({ maxlength: 1000 }),
    __metadata("design:type", String)
], Profile.prototype, "bio", void 0);
__decorate([
    (0, mongoose_1.Prop)({ type: Date }),
    __metadata("design:type", Date)
], Profile.prototype, "dateOfBirth", void 0);
__decorate([
    (0, mongoose_1.Prop)({ type: String, enum: ['MALE', 'FEMALE', 'OTHER', 'PREFER_NOT_TO_SAY'] }),
    __metadata("design:type", String)
], Profile.prototype, "gender", void 0);
__decorate([
    (0, mongoose_1.Prop)(),
    __metadata("design:type", String)
], Profile.prototype, "countryCode", void 0);
__decorate([
    (0, mongoose_1.Prop)(),
    __metadata("design:type", String)
], Profile.prototype, "city", void 0);
__decorate([
    (0, mongoose_1.Prop)({ type: Object, default: {} }),
    __metadata("design:type", UserPreferences)
], Profile.prototype, "preferences", void 0);
__decorate([
    (0, mongoose_1.Prop)({ type: exports.ShopConfigSchema }),
    __metadata("design:type", ShopConfig)
], Profile.prototype, "shopConfig", void 0);
__decorate([
    (0, mongoose_1.Prop)({ type: Object, default: {} }),
    __metadata("design:type", ProfileStats)
], Profile.prototype, "stats", void 0);
__decorate([
    (0, mongoose_1.Prop)({ type: String, enum: ['BRONZE', 'SILVER', 'GOLD', 'PLATINUM', 'DIAMOND'], default: 'BRONZE' }),
    __metadata("design:type", String)
], Profile.prototype, "membershipTier", void 0);
__decorate([
    (0, mongoose_1.Prop)({ default: 0 }),
    __metadata("design:type", Number)
], Profile.prototype, "loyaltyPoints", void 0);
__decorate([
    (0, mongoose_1.Prop)({ type: Date }),
    __metadata("design:type", Date)
], Profile.prototype, "membershipExpiresAt", void 0);
__decorate([
    (0, mongoose_1.Prop)({ default: false }),
    __metadata("design:type", Boolean)
], Profile.prototype, "isPhoneVerified", void 0);
__decorate([
    (0, mongoose_1.Prop)({ default: false }),
    __metadata("design:type", Boolean)
], Profile.prototype, "isEmailVerified", void 0);
__decorate([
    (0, mongoose_1.Prop)({ default: false }),
    __metadata("design:type", Boolean)
], Profile.prototype, "isIdentityVerified", void 0);
__decorate([
    (0, mongoose_1.Prop)({ type: Object }),
    __metadata("design:type", IdentityDocument)
], Profile.prototype, "identityDocument", void 0);
__decorate([
    (0, mongoose_1.Prop)({ type: [String], default: [] }),
    __metadata("design:type", Array)
], Profile.prototype, "followingShopIds", void 0);
__decorate([
    (0, mongoose_1.Prop)({ default: 0 }),
    __metadata("design:type", Number)
], Profile.prototype, "followersCount", void 0);
__decorate([
    (0, mongoose_1.Prop)({ type: Object, default: {} }),
    __metadata("design:type", Object)
], Profile.prototype, "metadata", void 0);
__decorate([
    (0, mongoose_1.Prop)({ type: Date }),
    __metadata("design:type", Date)
], Profile.prototype, "lastActiveAt", void 0);
__decorate([
    (0, mongoose_1.Prop)({ default: false }),
    __metadata("design:type", Boolean)
], Profile.prototype, "isDeleted", void 0);
__decorate([
    (0, mongoose_1.Prop)({ type: Date }),
    __metadata("design:type", Date)
], Profile.prototype, "deletedAt", void 0);
exports.Profile = Profile = __decorate([
    (0, mongoose_1.Schema)({ timestamps: true, collection: 'profiles' })
], Profile);
exports.ProfileSchema = mongoose_1.SchemaFactory.createForClass(Profile);
exports.ProfileSchema.index({ userId: 1 }, { unique: true });
exports.ProfileSchema.index({ 'shopConfig.shopSlug': 1 }, {
    unique: true,
    sparse: true,
    partialFilterExpression: { 'shopConfig.shopSlug': { $exists: true, $ne: null } },
});
exports.ProfileSchema.index({ 'shopConfig.shopName': 'text' });
exports.ProfileSchema.index({ membershipTier: 1 });
exports.ProfileSchema.index({ lastActiveAt: -1 });
exports.ProfileSchema.index({ isDeleted: 1 });
exports.ProfileSchema.index({ 'shopConfig.verificationStatus': 1 });
exports.ProfileSchema.index({ followingShopIds: 1 });
//# sourceMappingURL=profile.schema.js.map