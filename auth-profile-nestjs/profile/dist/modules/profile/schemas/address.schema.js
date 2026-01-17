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
exports.AddressSchema = exports.Address = exports.Coordinates = exports.AddressType = void 0;
const mongoose_1 = require("@nestjs/mongoose");
const mongoose_2 = require("mongoose");
var AddressType;
(function (AddressType) {
    AddressType["HOME"] = "HOME";
    AddressType["OFFICE"] = "OFFICE";
    AddressType["OTHER"] = "OTHER";
})(AddressType || (exports.AddressType = AddressType = {}));
let Coordinates = class Coordinates {
};
exports.Coordinates = Coordinates;
__decorate([
    (0, mongoose_1.Prop)({ type: Number }),
    __metadata("design:type", Number)
], Coordinates.prototype, "latitude", void 0);
__decorate([
    (0, mongoose_1.Prop)({ type: Number }),
    __metadata("design:type", Number)
], Coordinates.prototype, "longitude", void 0);
exports.Coordinates = Coordinates = __decorate([
    (0, mongoose_1.Schema)({ _id: false })
], Coordinates);
let Address = class Address extends mongoose_2.Document {
};
exports.Address = Address;
__decorate([
    (0, mongoose_1.Prop)({ required: true, index: true }),
    __metadata("design:type", String)
], Address.prototype, "userId", void 0);
__decorate([
    (0, mongoose_1.Prop)({ required: true, maxlength: 100 }),
    __metadata("design:type", String)
], Address.prototype, "contactName", void 0);
__decorate([
    (0, mongoose_1.Prop)({ required: true, maxlength: 20 }),
    __metadata("design:type", String)
], Address.prototype, "phone", void 0);
__decorate([
    (0, mongoose_1.Prop)({ maxlength: 255 }),
    __metadata("design:type", String)
], Address.prototype, "email", void 0);
__decorate([
    (0, mongoose_1.Prop)({ required: true }),
    __metadata("design:type", String)
], Address.prototype, "countryCode", void 0);
__decorate([
    (0, mongoose_1.Prop)({ required: true }),
    __metadata("design:type", String)
], Address.prototype, "provinceCode", void 0);
__decorate([
    (0, mongoose_1.Prop)(),
    __metadata("design:type", String)
], Address.prototype, "provinceName", void 0);
__decorate([
    (0, mongoose_1.Prop)({ required: true }),
    __metadata("design:type", String)
], Address.prototype, "districtCode", void 0);
__decorate([
    (0, mongoose_1.Prop)(),
    __metadata("design:type", String)
], Address.prototype, "districtName", void 0);
__decorate([
    (0, mongoose_1.Prop)({ required: true }),
    __metadata("design:type", String)
], Address.prototype, "wardCode", void 0);
__decorate([
    (0, mongoose_1.Prop)(),
    __metadata("design:type", String)
], Address.prototype, "wardName", void 0);
__decorate([
    (0, mongoose_1.Prop)({ required: true, maxlength: 500 }),
    __metadata("design:type", String)
], Address.prototype, "streetAddress", void 0);
__decorate([
    (0, mongoose_1.Prop)({ maxlength: 100 }),
    __metadata("design:type", String)
], Address.prototype, "apartment", void 0);
__decorate([
    (0, mongoose_1.Prop)({ maxlength: 20 }),
    __metadata("design:type", String)
], Address.prototype, "postalCode", void 0);
__decorate([
    (0, mongoose_1.Prop)({ maxlength: 1000 }),
    __metadata("design:type", String)
], Address.prototype, "fullAddress", void 0);
__decorate([
    (0, mongoose_1.Prop)({ type: Object }),
    __metadata("design:type", Coordinates)
], Address.prototype, "coordinates", void 0);
__decorate([
    (0, mongoose_1.Prop)({ type: String, enum: AddressType, default: AddressType.HOME }),
    __metadata("design:type", String)
], Address.prototype, "type", void 0);
__decorate([
    (0, mongoose_1.Prop)({ maxlength: 50 }),
    __metadata("design:type", String)
], Address.prototype, "label", void 0);
__decorate([
    (0, mongoose_1.Prop)({ default: false }),
    __metadata("design:type", Boolean)
], Address.prototype, "isDefault", void 0);
__decorate([
    (0, mongoose_1.Prop)({ default: false }),
    __metadata("design:type", Boolean)
], Address.prototype, "isDefaultBilling", void 0);
__decorate([
    (0, mongoose_1.Prop)({ default: false }),
    __metadata("design:type", Boolean)
], Address.prototype, "isDefaultPickup", void 0);
__decorate([
    (0, mongoose_1.Prop)({ maxlength: 500 }),
    __metadata("design:type", String)
], Address.prototype, "deliveryInstructions", void 0);
__decorate([
    (0, mongoose_1.Prop)({ default: false }),
    __metadata("design:type", Boolean)
], Address.prototype, "isVerified", void 0);
__decorate([
    (0, mongoose_1.Prop)({ type: Date }),
    __metadata("design:type", Date)
], Address.prototype, "verifiedAt", void 0);
__decorate([
    (0, mongoose_1.Prop)({ default: false }),
    __metadata("design:type", Boolean)
], Address.prototype, "isDeleted", void 0);
__decorate([
    (0, mongoose_1.Prop)({ type: Date }),
    __metadata("design:type", Date)
], Address.prototype, "deletedAt", void 0);
exports.Address = Address = __decorate([
    (0, mongoose_1.Schema)({ timestamps: true, collection: 'addresses' })
], Address);
exports.AddressSchema = mongoose_1.SchemaFactory.createForClass(Address);
exports.AddressSchema.index({ userId: 1, isDeleted: 1 });
exports.AddressSchema.index({ userId: 1, isDefault: 1 });
exports.AddressSchema.index({ userId: 1, isDefaultBilling: 1 });
exports.AddressSchema.index({ userId: 1, isDefaultPickup: 1 });
exports.AddressSchema.index({ 'coordinates.latitude': 1, 'coordinates.longitude': 1 });
exports.AddressSchema.index({ coordinates: '2dsphere' }, { sparse: true, partialFilterExpression: { coordinates: { $exists: true } } });
exports.AddressSchema.pre('save', function (next) {
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
    next();
});
//# sourceMappingURL=address.schema.js.map