import {
  IsString,
  IsNotEmpty,
  IsOptional,
  IsBoolean,
  IsEnum,
  Matches,
} from 'class-validator';
import { AddressType } from '../schemas/address.schema';

export class CreateAddressDto {
  @IsString()
  @IsNotEmpty()
  contactName!: string;

  @IsString()
  @IsNotEmpty()
  @Matches(/^(\+84|84|0)?[1-9]\d{8,9}$/, {
    message: 'Phone number must be a valid Vietnamese phone number',
  })
  phone!: string;

  @IsString()
  @IsNotEmpty()
  provinceCode!: string;

  @IsString()
  @IsNotEmpty()
  districtCode!: string;

  @IsString()
  @IsNotEmpty()
  wardCode!: string;

  @IsString()
  @IsNotEmpty()
  streetLine!: string;

  @IsString()
  @IsOptional()
  fullAddress?: string;

  @IsBoolean()
  @IsOptional()
  isDefault?: boolean;

  @IsEnum(AddressType)
  @IsOptional()
  type?: AddressType;
}

export class UpdateAddressDto {
  @IsString()
  @IsOptional()
  contactName?: string;

  @IsString()
  @IsOptional()
  @Matches(/^(\+84|84|0)?[1-9]\d{8,9}$/, {
    message: 'Phone number must be a valid Vietnamese phone number',
  })
  phone?: string;

  @IsString()
  @IsOptional()
  provinceCode?: string;

  @IsString()
  @IsOptional()
  districtCode?: string;

  @IsString()
  @IsOptional()
  wardCode?: string;

  @IsString()
  @IsOptional()
  streetLine?: string;

  @IsString()
  @IsOptional()
  fullAddress?: string;

  @IsBoolean()
  @IsOptional()
  isDefault?: boolean;

  @IsEnum(AddressType)
  @IsOptional()
  type?: AddressType;
}
