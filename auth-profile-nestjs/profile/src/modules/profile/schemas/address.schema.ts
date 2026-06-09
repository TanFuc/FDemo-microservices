import { Prop, Schema, SchemaFactory } from '@nestjs/mongoose';
import { Document, HydratedDocument } from 'mongoose';

export enum AddressType {
  HOME = 'HOME',
  OFFICE = 'OFFICE',
  OTHER = 'OTHER',
}

export type AddressDocument = HydratedDocument<Address>;

@Schema({ _id: false })
export class Coordinates {
  @Prop({ type: Number })
  latitude?: number;

  @Prop({ type: Number })
  longitude?: number;
}

@Schema({ timestamps: true, collection: 'addresses' })
export class Address extends Document {
  @Prop({ required: true, index: true })
  userId!: string;

  @Prop({ required: true, maxlength: 100 })
  contactName!: string;

  @Prop({ required: true, maxlength: 20 })
  phone!: string;

  @Prop({ maxlength: 255 })
  email?: string;

  // Location Codes (for shipping calculation)
  @Prop({ required: true })
  countryCode!: string;

  @Prop({ required: true })
  provinceCode!: string;

  @Prop()
  provinceName?: string;

  @Prop({ required: true })
  districtCode!: string;

  @Prop()
  districtName?: string;

  @Prop({ required: true })
  wardCode!: string;

  @Prop()
  wardName?: string;

  @Prop({ required: true, maxlength: 500 })
  streetAddress!: string;

  @Prop({ maxlength: 100 })
  apartment?: string; // Unit, floor, building

  @Prop({ maxlength: 20 })
  postalCode?: string;

  // Computed full address
  @Prop({ maxlength: 1000 })
  fullAddress?: string;

  // Geolocation (for delivery optimization)
  @Prop({ type: Object })
  coordinates?: Coordinates;

  // Type & Flags
  @Prop({ type: String, enum: AddressType, default: AddressType.HOME })
  type!: AddressType;

  @Prop({ maxlength: 50 })
  label?: string; // Custom label like "Mom's house"

  @Prop({ default: false })
  isDefault!: boolean;

  @Prop({ default: false })
  isDefaultBilling?: boolean;

  @Prop({ default: false })
  isDefaultPickup?: boolean; // For sellers

  // Delivery Instructions
  @Prop({ maxlength: 500 })
  deliveryInstructions?: string;

  // Validation
  @Prop({ default: false })
  isVerified?: boolean;

  @Prop({ type: Date })
  verifiedAt?: Date;

  // Soft delete
  @Prop({ default: false })
  isDeleted?: boolean;

  @Prop({ type: Date })
  deletedAt?: Date;
}

export const AddressSchema = SchemaFactory.createForClass(Address);

// Indexes
AddressSchema.index({ userId: 1, isDeleted: 1 });
AddressSchema.index({ userId: 1, isDefault: 1 });
AddressSchema.index({ userId: 1, isDefaultBilling: 1 });
AddressSchema.index({ userId: 1, isDefaultPickup: 1 });
AddressSchema.index({ 'coordinates.latitude': 1, 'coordinates.longitude': 1 });
AddressSchema.index(
  { coordinates: '2dsphere' },
  { sparse: true, partialFilterExpression: { coordinates: { $exists: true } } },
);

// Pre-save hook to compute full address
AddressSchema.pre('save', function () {
  if (this.isModified('streetAddress') || this.isModified('wardName') ||
      this.isModified('districtName') || this.isModified('provinceName')) {
    const parts = [
      this.apartment,
      this.streetAddress,
      this.wardName,
      this.districtName,
      this.provinceName,
    ].filter(Boolean);
    this.fullAddress = parts.join(', ');
  }
});
