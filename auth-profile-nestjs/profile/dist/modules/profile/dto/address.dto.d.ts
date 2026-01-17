import { AddressType } from '../schemas/address.schema';
export declare class CreateAddressDto {
    contactName: string;
    phone: string;
    provinceCode: string;
    districtCode: string;
    wardCode: string;
    streetLine: string;
    fullAddress?: string;
    isDefault?: boolean;
    type?: AddressType;
}
export declare class UpdateAddressDto {
    contactName?: string;
    phone?: string;
    provinceCode?: string;
    districtCode?: string;
    wardCode?: string;
    streetLine?: string;
    fullAddress?: string;
    isDefault?: boolean;
    type?: AddressType;
}
