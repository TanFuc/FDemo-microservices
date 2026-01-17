import { Prop, Schema, SchemaFactory } from '@nestjs/mongoose';
import { Document, HydratedDocument, Types } from 'mongoose';

export type ProfileDocument = HydratedDocument<Profile>;

// ==================== Nested Schemas ====================

@Schema({ _id: false })
export class NotificationPreferences {
  @Prop({ type: Object, default: { orders: true, promotions: true, news: false } })
  email!: { orders: boolean; promotions: boolean; news: boolean };

  @Prop({ type: Object, default: { orders: true, promotions: true, chat: true } })
  push!: { orders: boolean; promotions: boolean; chat: boolean };

  @Prop({ type: Object, default: { orders: true, otp: true } })
  sms!: { orders: boolean; otp: boolean };
}

@Schema({ _id: false })
export class PrivacySettings {
  @Prop({ default: true })
  showOnlineStatus!: boolean;

  @Prop({ default: true })
  showLastSeen!: boolean;

  @Prop({ type: String, enum: ['EVERYONE', 'FOLLOWING', 'NOBODY'], default: 'EVERYONE' })
  allowMessages!: string;
}

@Schema({ _id: false })
export class UserPreferences {
  @Prop({ default: 'vi' })
  language!: string;

  @Prop({ default: 'VND' })
  currency!: string;

  @Prop({ default: 'Asia/Ho_Chi_Minh' })
  timezone!: string;

  @Prop({ type: String, enum: ['METRIC', 'IMPERIAL'], default: 'METRIC' })
  measurementUnit!: string;

  @Prop({ type: NotificationPreferences })
  notifications?: NotificationPreferences;

  @Prop({ type: PrivacySettings })
  privacy?: PrivacySettings;
}

@Schema({ _id: false })
export class BankAccount {
  @Prop()
  bankName?: string;

  @Prop()
  accountNumber?: string;

  @Prop()
  accountHolder?: string;

  @Prop()
  branch?: string;
}

@Schema({ _id: false })
export class ShopPolicies {
  @Prop({ type: String })
  returnPolicy?: string;

  @Prop({ type: String })
  shippingPolicy?: string;
}

@Schema({ _id: false })
export class OperatingHours {
  @Prop({ type: String })
  open?: string;

  @Prop({ type: String })
  close?: string;

  @Prop({ default: false })
  closed?: boolean;
}

@Schema({ _id: false })
export class SocialLinks {
  @Prop()
  facebook?: string;

  @Prop()
  instagram?: string;

  @Prop()
  tiktok?: string;

  @Prop()
  youtube?: string;

  @Prop()
  website?: string;
}

@Schema({ _id: false })
export class ShopConfig {
  @Prop({ type: String })
  shopId?: string;

  @Prop({ type: String })
  shopName?: string;

  @Prop({ type: String })
  shopSlug?: string;

  @Prop()
  description?: string;

  @Prop()
  logoUrl?: string;

  @Prop()
  bannerUrl?: string;

  @Prop({ type: Types.ObjectId, ref: 'Address' })
  pickupAddressId?: Types.ObjectId;

  @Prop({ type: String, enum: ['INDIVIDUAL', 'BUSINESS'], default: 'INDIVIDUAL' })
  businessType?: string;

  @Prop()
  businessLicense?: string;

  @Prop()
  taxId?: string;

  @Prop({ type: BankAccount })
  bankAccount?: BankAccount;

  @Prop({ type: ShopPolicies })
  policies?: ShopPolicies;

  @Prop({ type: Object })
  operatingHours?: {
    monday?: OperatingHours;
    tuesday?: OperatingHours;
    wednesday?: OperatingHours;
    thursday?: OperatingHours;
    friday?: OperatingHours;
    saturday?: OperatingHours;
    sunday?: OperatingHours;
  };

  @Prop({ type: SocialLinks })
  socialLinks?: SocialLinks;

  @Prop({ type: String, enum: ['PENDING', 'VERIFIED', 'REJECTED'], default: 'PENDING' })
  verificationStatus?: string;

  @Prop({ type: Date })
  verifiedAt?: Date;

  @Prop({ default: 0 })
  rating?: number;

  @Prop({ default: 0 })
  totalReviews?: number;

  @Prop({ default: 0 })
  totalProducts?: number;

  @Prop({ default: 0 })
  totalOrders?: number;

  @Prop({ default: 0 })
  totalFollowers?: number;

  @Prop({ type: Date })
  joinedAt?: Date;
}

export const ShopConfigSchema = SchemaFactory.createForClass(ShopConfig);

@Schema({ _id: false })
export class ProfileStats {
  @Prop({ default: 0 })
  totalOrders!: number;

  @Prop({ default: 0 })
  completedOrders!: number;

  @Prop({ default: 0 })
  cancelledOrders!: number;

  @Prop({ default: 0 })
  totalSpent!: number;

  @Prop({ default: 0 })
  averageOrderValue!: number;

  @Prop({ default: 0 })
  totalReviews!: number;

  @Prop({ default: 0 })
  wishlistCount!: number;

  @Prop({ default: 0 })
  cartItemCount!: number;
}

@Schema({ _id: false })
export class IdentityDocument {
  @Prop({ type: String, enum: ['ID_CARD', 'PASSPORT', 'DRIVER_LICENSE'] })
  type?: string;

  @Prop()
  number?: string;

  @Prop()
  frontImage?: string;

  @Prop()
  backImage?: string;

  @Prop()
  selfieImage?: string;

  @Prop({ type: Date })
  verifiedAt?: Date;
}

// ==================== Main Profile Schema ====================

@Schema({ timestamps: true, collection: 'profiles' })
export class Profile extends Document {
  @Prop({ required: true, unique: true, index: true })
  userId!: string; // UUID from Auth Service

  // Basic Info
  @Prop({ required: true, maxlength: 100 })
  displayName!: string;

  @Prop({ maxlength: 255 })
  email?: string;

  @Prop({ maxlength: 20 })
  phone?: string;

  @Prop({ maxlength: 500 })
  avatarUrl?: string;

  @Prop({ maxlength: 500 })
  coverUrl?: string;

  @Prop({ maxlength: 1000 })
  bio?: string;

  @Prop({ type: Date })
  dateOfBirth?: Date;

  @Prop({ type: String, enum: ['MALE', 'FEMALE', 'OTHER', 'PREFER_NOT_TO_SAY'] })
  gender?: string;

  // Location
  @Prop()
  countryCode?: string;

  @Prop()
  city?: string;

  // Preferences
  @Prop({ type: Object, default: {} })
  preferences?: UserPreferences;

  // Shop Configuration (for Sellers)
  @Prop({ type: ShopConfigSchema })
  shopConfig?: ShopConfig;

  // Stats (Denormalized)
  @Prop({ type: Object, default: {} })
  stats?: ProfileStats;

  // Membership
  @Prop({ type: String, enum: ['BRONZE', 'SILVER', 'GOLD', 'PLATINUM', 'DIAMOND'], default: 'BRONZE' })
  membershipTier?: string;

  @Prop({ default: 0 })
  loyaltyPoints?: number;

  @Prop({ type: Date })
  membershipExpiresAt?: Date;

  // Verification
  @Prop({ default: false })
  isPhoneVerified?: boolean;

  @Prop({ default: false })
  isEmailVerified?: boolean;

  @Prop({ default: false })
  isIdentityVerified?: boolean;

  @Prop({ type: Object })
  identityDocument?: IdentityDocument;

  // Following/Followers (for social features)
  @Prop({ type: [String], default: [] })
  followingShopIds?: string[];

  @Prop({ default: 0 })
  followersCount?: number;

  // Metadata
  @Prop({ type: Object, default: {} })
  metadata?: Record<string, any>;

  // Internal
  @Prop({ type: Date })
  lastActiveAt?: Date;

  @Prop({ default: false })
  isDeleted?: boolean;

  @Prop({ type: Date })
  deletedAt?: Date;
}

export const ProfileSchema = SchemaFactory.createForClass(Profile);

// Indexes
ProfileSchema.index({ userId: 1 }, { unique: true });
ProfileSchema.index(
  { 'shopConfig.shopSlug': 1 },
  {
    unique: true,
    sparse: true,
    partialFilterExpression: { 'shopConfig.shopSlug': { $exists: true, $ne: null } },
  },
);
ProfileSchema.index({ 'shopConfig.shopName': 'text' });
ProfileSchema.index({ membershipTier: 1 });
ProfileSchema.index({ lastActiveAt: -1 });
ProfileSchema.index({ isDeleted: 1 });
ProfileSchema.index({ 'shopConfig.verificationStatus': 1 });
ProfileSchema.index({ followingShopIds: 1 });
