import { Document, HydratedDocument, Types } from 'mongoose';
export type ProfileDocument = HydratedDocument<Profile>;
export declare class NotificationPreferences {
    email: {
        orders: boolean;
        promotions: boolean;
        news: boolean;
    };
    push: {
        orders: boolean;
        promotions: boolean;
        chat: boolean;
    };
    sms: {
        orders: boolean;
        otp: boolean;
    };
}
export declare class PrivacySettings {
    showOnlineStatus: boolean;
    showLastSeen: boolean;
    allowMessages: string;
}
export declare class UserPreferences {
    language: string;
    currency: string;
    timezone: string;
    measurementUnit: string;
    notifications?: NotificationPreferences;
    privacy?: PrivacySettings;
}
export declare class BankAccount {
    bankName?: string;
    accountNumber?: string;
    accountHolder?: string;
    branch?: string;
}
export declare class ShopPolicies {
    returnPolicy?: string;
    shippingPolicy?: string;
}
export declare class OperatingHours {
    open?: string;
    close?: string;
    closed?: boolean;
}
export declare class SocialLinks {
    facebook?: string;
    instagram?: string;
    tiktok?: string;
    youtube?: string;
    website?: string;
}
export declare class ShopConfig {
    shopId?: string;
    shopName?: string;
    shopSlug?: string;
    description?: string;
    logoUrl?: string;
    bannerUrl?: string;
    pickupAddressId?: Types.ObjectId;
    businessType?: string;
    businessLicense?: string;
    taxId?: string;
    bankAccount?: BankAccount;
    policies?: ShopPolicies;
    operatingHours?: {
        monday?: OperatingHours;
        tuesday?: OperatingHours;
        wednesday?: OperatingHours;
        thursday?: OperatingHours;
        friday?: OperatingHours;
        saturday?: OperatingHours;
        sunday?: OperatingHours;
    };
    socialLinks?: SocialLinks;
    verificationStatus?: string;
    verifiedAt?: Date;
    rating?: number;
    totalReviews?: number;
    totalProducts?: number;
    totalOrders?: number;
    totalFollowers?: number;
    joinedAt?: Date;
}
export declare const ShopConfigSchema: import("mongoose").Schema<ShopConfig, import("mongoose").Model<ShopConfig, any, any, any, (Document<unknown, any, ShopConfig, any, import("mongoose").DefaultSchemaOptions> & ShopConfig & {
    _id: Types.ObjectId;
} & {
    __v: number;
} & {
    id: string;
}) | (Document<unknown, any, ShopConfig, any, import("mongoose").DefaultSchemaOptions> & ShopConfig & {
    _id: Types.ObjectId;
} & {
    __v: number;
}), any, ShopConfig>, {}, {}, {}, {}, import("mongoose").DefaultSchemaOptions, ShopConfig, Document<unknown, {}, ShopConfig, {
    id: string;
}, import("mongoose").ResolveSchemaOptions<import("mongoose").DefaultSchemaOptions>> & Omit<ShopConfig & {
    _id: Types.ObjectId;
} & {
    __v: number;
}, "id"> & {
    id: string;
}, {
    shopId?: import("mongoose").SchemaDefinitionProperty<string | undefined, ShopConfig, Document<unknown, {}, ShopConfig, {
        id: string;
    }, import("mongoose").ResolveSchemaOptions<import("mongoose").DefaultSchemaOptions>> & Omit<ShopConfig & {
        _id: Types.ObjectId;
    } & {
        __v: number;
    }, "id"> & {
        id: string;
    }> | undefined;
    shopName?: import("mongoose").SchemaDefinitionProperty<string | undefined, ShopConfig, Document<unknown, {}, ShopConfig, {
        id: string;
    }, import("mongoose").ResolveSchemaOptions<import("mongoose").DefaultSchemaOptions>> & Omit<ShopConfig & {
        _id: Types.ObjectId;
    } & {
        __v: number;
    }, "id"> & {
        id: string;
    }> | undefined;
    shopSlug?: import("mongoose").SchemaDefinitionProperty<string | undefined, ShopConfig, Document<unknown, {}, ShopConfig, {
        id: string;
    }, import("mongoose").ResolveSchemaOptions<import("mongoose").DefaultSchemaOptions>> & Omit<ShopConfig & {
        _id: Types.ObjectId;
    } & {
        __v: number;
    }, "id"> & {
        id: string;
    }> | undefined;
    description?: import("mongoose").SchemaDefinitionProperty<string | undefined, ShopConfig, Document<unknown, {}, ShopConfig, {
        id: string;
    }, import("mongoose").ResolveSchemaOptions<import("mongoose").DefaultSchemaOptions>> & Omit<ShopConfig & {
        _id: Types.ObjectId;
    } & {
        __v: number;
    }, "id"> & {
        id: string;
    }> | undefined;
    logoUrl?: import("mongoose").SchemaDefinitionProperty<string | undefined, ShopConfig, Document<unknown, {}, ShopConfig, {
        id: string;
    }, import("mongoose").ResolveSchemaOptions<import("mongoose").DefaultSchemaOptions>> & Omit<ShopConfig & {
        _id: Types.ObjectId;
    } & {
        __v: number;
    }, "id"> & {
        id: string;
    }> | undefined;
    bannerUrl?: import("mongoose").SchemaDefinitionProperty<string | undefined, ShopConfig, Document<unknown, {}, ShopConfig, {
        id: string;
    }, import("mongoose").ResolveSchemaOptions<import("mongoose").DefaultSchemaOptions>> & Omit<ShopConfig & {
        _id: Types.ObjectId;
    } & {
        __v: number;
    }, "id"> & {
        id: string;
    }> | undefined;
    pickupAddressId?: import("mongoose").SchemaDefinitionProperty<Types.ObjectId | undefined, ShopConfig, Document<unknown, {}, ShopConfig, {
        id: string;
    }, import("mongoose").ResolveSchemaOptions<import("mongoose").DefaultSchemaOptions>> & Omit<ShopConfig & {
        _id: Types.ObjectId;
    } & {
        __v: number;
    }, "id"> & {
        id: string;
    }> | undefined;
    businessType?: import("mongoose").SchemaDefinitionProperty<string | undefined, ShopConfig, Document<unknown, {}, ShopConfig, {
        id: string;
    }, import("mongoose").ResolveSchemaOptions<import("mongoose").DefaultSchemaOptions>> & Omit<ShopConfig & {
        _id: Types.ObjectId;
    } & {
        __v: number;
    }, "id"> & {
        id: string;
    }> | undefined;
    businessLicense?: import("mongoose").SchemaDefinitionProperty<string | undefined, ShopConfig, Document<unknown, {}, ShopConfig, {
        id: string;
    }, import("mongoose").ResolveSchemaOptions<import("mongoose").DefaultSchemaOptions>> & Omit<ShopConfig & {
        _id: Types.ObjectId;
    } & {
        __v: number;
    }, "id"> & {
        id: string;
    }> | undefined;
    taxId?: import("mongoose").SchemaDefinitionProperty<string | undefined, ShopConfig, Document<unknown, {}, ShopConfig, {
        id: string;
    }, import("mongoose").ResolveSchemaOptions<import("mongoose").DefaultSchemaOptions>> & Omit<ShopConfig & {
        _id: Types.ObjectId;
    } & {
        __v: number;
    }, "id"> & {
        id: string;
    }> | undefined;
    bankAccount?: import("mongoose").SchemaDefinitionProperty<BankAccount | undefined, ShopConfig, Document<unknown, {}, ShopConfig, {
        id: string;
    }, import("mongoose").ResolveSchemaOptions<import("mongoose").DefaultSchemaOptions>> & Omit<ShopConfig & {
        _id: Types.ObjectId;
    } & {
        __v: number;
    }, "id"> & {
        id: string;
    }> | undefined;
    policies?: import("mongoose").SchemaDefinitionProperty<ShopPolicies | undefined, ShopConfig, Document<unknown, {}, ShopConfig, {
        id: string;
    }, import("mongoose").ResolveSchemaOptions<import("mongoose").DefaultSchemaOptions>> & Omit<ShopConfig & {
        _id: Types.ObjectId;
    } & {
        __v: number;
    }, "id"> & {
        id: string;
    }> | undefined;
    operatingHours?: import("mongoose").SchemaDefinitionProperty<{
        monday?: OperatingHours;
        tuesday?: OperatingHours;
        wednesday?: OperatingHours;
        thursday?: OperatingHours;
        friday?: OperatingHours;
        saturday?: OperatingHours;
        sunday?: OperatingHours;
    } | undefined, ShopConfig, Document<unknown, {}, ShopConfig, {
        id: string;
    }, import("mongoose").ResolveSchemaOptions<import("mongoose").DefaultSchemaOptions>> & Omit<ShopConfig & {
        _id: Types.ObjectId;
    } & {
        __v: number;
    }, "id"> & {
        id: string;
    }> | undefined;
    socialLinks?: import("mongoose").SchemaDefinitionProperty<SocialLinks | undefined, ShopConfig, Document<unknown, {}, ShopConfig, {
        id: string;
    }, import("mongoose").ResolveSchemaOptions<import("mongoose").DefaultSchemaOptions>> & Omit<ShopConfig & {
        _id: Types.ObjectId;
    } & {
        __v: number;
    }, "id"> & {
        id: string;
    }> | undefined;
    verificationStatus?: import("mongoose").SchemaDefinitionProperty<string | undefined, ShopConfig, Document<unknown, {}, ShopConfig, {
        id: string;
    }, import("mongoose").ResolveSchemaOptions<import("mongoose").DefaultSchemaOptions>> & Omit<ShopConfig & {
        _id: Types.ObjectId;
    } & {
        __v: number;
    }, "id"> & {
        id: string;
    }> | undefined;
    verifiedAt?: import("mongoose").SchemaDefinitionProperty<Date | undefined, ShopConfig, Document<unknown, {}, ShopConfig, {
        id: string;
    }, import("mongoose").ResolveSchemaOptions<import("mongoose").DefaultSchemaOptions>> & Omit<ShopConfig & {
        _id: Types.ObjectId;
    } & {
        __v: number;
    }, "id"> & {
        id: string;
    }> | undefined;
    rating?: import("mongoose").SchemaDefinitionProperty<number | undefined, ShopConfig, Document<unknown, {}, ShopConfig, {
        id: string;
    }, import("mongoose").ResolveSchemaOptions<import("mongoose").DefaultSchemaOptions>> & Omit<ShopConfig & {
        _id: Types.ObjectId;
    } & {
        __v: number;
    }, "id"> & {
        id: string;
    }> | undefined;
    totalReviews?: import("mongoose").SchemaDefinitionProperty<number | undefined, ShopConfig, Document<unknown, {}, ShopConfig, {
        id: string;
    }, import("mongoose").ResolveSchemaOptions<import("mongoose").DefaultSchemaOptions>> & Omit<ShopConfig & {
        _id: Types.ObjectId;
    } & {
        __v: number;
    }, "id"> & {
        id: string;
    }> | undefined;
    totalProducts?: import("mongoose").SchemaDefinitionProperty<number | undefined, ShopConfig, Document<unknown, {}, ShopConfig, {
        id: string;
    }, import("mongoose").ResolveSchemaOptions<import("mongoose").DefaultSchemaOptions>> & Omit<ShopConfig & {
        _id: Types.ObjectId;
    } & {
        __v: number;
    }, "id"> & {
        id: string;
    }> | undefined;
    totalOrders?: import("mongoose").SchemaDefinitionProperty<number | undefined, ShopConfig, Document<unknown, {}, ShopConfig, {
        id: string;
    }, import("mongoose").ResolveSchemaOptions<import("mongoose").DefaultSchemaOptions>> & Omit<ShopConfig & {
        _id: Types.ObjectId;
    } & {
        __v: number;
    }, "id"> & {
        id: string;
    }> | undefined;
    totalFollowers?: import("mongoose").SchemaDefinitionProperty<number | undefined, ShopConfig, Document<unknown, {}, ShopConfig, {
        id: string;
    }, import("mongoose").ResolveSchemaOptions<import("mongoose").DefaultSchemaOptions>> & Omit<ShopConfig & {
        _id: Types.ObjectId;
    } & {
        __v: number;
    }, "id"> & {
        id: string;
    }> | undefined;
    joinedAt?: import("mongoose").SchemaDefinitionProperty<Date | undefined, ShopConfig, Document<unknown, {}, ShopConfig, {
        id: string;
    }, import("mongoose").ResolveSchemaOptions<import("mongoose").DefaultSchemaOptions>> & Omit<ShopConfig & {
        _id: Types.ObjectId;
    } & {
        __v: number;
    }, "id"> & {
        id: string;
    }> | undefined;
}, ShopConfig>;
export declare class ProfileStats {
    totalOrders: number;
    completedOrders: number;
    cancelledOrders: number;
    totalSpent: number;
    averageOrderValue: number;
    totalReviews: number;
    wishlistCount: number;
    cartItemCount: number;
}
export declare class IdentityDocument {
    type?: string;
    number?: string;
    frontImage?: string;
    backImage?: string;
    selfieImage?: string;
    verifiedAt?: Date;
}
export declare class Profile extends Document {
    userId: string;
    displayName: string;
    email?: string;
    phone?: string;
    avatarUrl?: string;
    coverUrl?: string;
    bio?: string;
    dateOfBirth?: Date;
    gender?: string;
    countryCode?: string;
    city?: string;
    preferences?: UserPreferences;
    shopConfig?: ShopConfig;
    stats?: ProfileStats;
    membershipTier?: string;
    loyaltyPoints?: number;
    membershipExpiresAt?: Date;
    isPhoneVerified?: boolean;
    isEmailVerified?: boolean;
    isIdentityVerified?: boolean;
    identityDocument?: IdentityDocument;
    followingShopIds?: string[];
    followersCount?: number;
    metadata?: Record<string, any>;
    lastActiveAt?: Date;
    isDeleted?: boolean;
    deletedAt?: Date;
}
export declare const ProfileSchema: import("mongoose").Schema<Profile, import("mongoose").Model<Profile, any, any, any, (Document<unknown, any, Profile, any, import("mongoose").DefaultSchemaOptions> & Profile & Required<{
    _id: Types.ObjectId;
}> & {
    __v: number;
} & {
    id: string;
}) | (Document<unknown, any, Profile, any, import("mongoose").DefaultSchemaOptions> & Profile & Required<{
    _id: Types.ObjectId;
}> & {
    __v: number;
}), any, Profile>, {}, {}, {}, {}, import("mongoose").DefaultSchemaOptions, Profile, Document<unknown, {}, Profile, {
    id: string;
}, import("mongoose").ResolveSchemaOptions<import("mongoose").DefaultSchemaOptions>> & Omit<Profile & Required<{
    _id: Types.ObjectId;
}> & {
    __v: number;
}, "id"> & {
    id: string;
}, {
    userId?: import("mongoose").SchemaDefinitionProperty<string, Profile, Document<unknown, {}, Profile, {
        id: string;
    }, import("mongoose").ResolveSchemaOptions<import("mongoose").DefaultSchemaOptions>> & Omit<Profile & Required<{
        _id: Types.ObjectId;
    }> & {
        __v: number;
    }, "id"> & {
        id: string;
    }> | undefined;
    displayName?: import("mongoose").SchemaDefinitionProperty<string, Profile, Document<unknown, {}, Profile, {
        id: string;
    }, import("mongoose").ResolveSchemaOptions<import("mongoose").DefaultSchemaOptions>> & Omit<Profile & Required<{
        _id: Types.ObjectId;
    }> & {
        __v: number;
    }, "id"> & {
        id: string;
    }> | undefined;
    email?: import("mongoose").SchemaDefinitionProperty<string | undefined, Profile, Document<unknown, {}, Profile, {
        id: string;
    }, import("mongoose").ResolveSchemaOptions<import("mongoose").DefaultSchemaOptions>> & Omit<Profile & Required<{
        _id: Types.ObjectId;
    }> & {
        __v: number;
    }, "id"> & {
        id: string;
    }> | undefined;
    phone?: import("mongoose").SchemaDefinitionProperty<string | undefined, Profile, Document<unknown, {}, Profile, {
        id: string;
    }, import("mongoose").ResolveSchemaOptions<import("mongoose").DefaultSchemaOptions>> & Omit<Profile & Required<{
        _id: Types.ObjectId;
    }> & {
        __v: number;
    }, "id"> & {
        id: string;
    }> | undefined;
    avatarUrl?: import("mongoose").SchemaDefinitionProperty<string | undefined, Profile, Document<unknown, {}, Profile, {
        id: string;
    }, import("mongoose").ResolveSchemaOptions<import("mongoose").DefaultSchemaOptions>> & Omit<Profile & Required<{
        _id: Types.ObjectId;
    }> & {
        __v: number;
    }, "id"> & {
        id: string;
    }> | undefined;
    coverUrl?: import("mongoose").SchemaDefinitionProperty<string | undefined, Profile, Document<unknown, {}, Profile, {
        id: string;
    }, import("mongoose").ResolveSchemaOptions<import("mongoose").DefaultSchemaOptions>> & Omit<Profile & Required<{
        _id: Types.ObjectId;
    }> & {
        __v: number;
    }, "id"> & {
        id: string;
    }> | undefined;
    bio?: import("mongoose").SchemaDefinitionProperty<string | undefined, Profile, Document<unknown, {}, Profile, {
        id: string;
    }, import("mongoose").ResolveSchemaOptions<import("mongoose").DefaultSchemaOptions>> & Omit<Profile & Required<{
        _id: Types.ObjectId;
    }> & {
        __v: number;
    }, "id"> & {
        id: string;
    }> | undefined;
    dateOfBirth?: import("mongoose").SchemaDefinitionProperty<Date | undefined, Profile, Document<unknown, {}, Profile, {
        id: string;
    }, import("mongoose").ResolveSchemaOptions<import("mongoose").DefaultSchemaOptions>> & Omit<Profile & Required<{
        _id: Types.ObjectId;
    }> & {
        __v: number;
    }, "id"> & {
        id: string;
    }> | undefined;
    gender?: import("mongoose").SchemaDefinitionProperty<string | undefined, Profile, Document<unknown, {}, Profile, {
        id: string;
    }, import("mongoose").ResolveSchemaOptions<import("mongoose").DefaultSchemaOptions>> & Omit<Profile & Required<{
        _id: Types.ObjectId;
    }> & {
        __v: number;
    }, "id"> & {
        id: string;
    }> | undefined;
    countryCode?: import("mongoose").SchemaDefinitionProperty<string | undefined, Profile, Document<unknown, {}, Profile, {
        id: string;
    }, import("mongoose").ResolveSchemaOptions<import("mongoose").DefaultSchemaOptions>> & Omit<Profile & Required<{
        _id: Types.ObjectId;
    }> & {
        __v: number;
    }, "id"> & {
        id: string;
    }> | undefined;
    city?: import("mongoose").SchemaDefinitionProperty<string | undefined, Profile, Document<unknown, {}, Profile, {
        id: string;
    }, import("mongoose").ResolveSchemaOptions<import("mongoose").DefaultSchemaOptions>> & Omit<Profile & Required<{
        _id: Types.ObjectId;
    }> & {
        __v: number;
    }, "id"> & {
        id: string;
    }> | undefined;
    preferences?: import("mongoose").SchemaDefinitionProperty<UserPreferences | undefined, Profile, Document<unknown, {}, Profile, {
        id: string;
    }, import("mongoose").ResolveSchemaOptions<import("mongoose").DefaultSchemaOptions>> & Omit<Profile & Required<{
        _id: Types.ObjectId;
    }> & {
        __v: number;
    }, "id"> & {
        id: string;
    }> | undefined;
    membershipTier?: import("mongoose").SchemaDefinitionProperty<string | undefined, Profile, Document<unknown, {}, Profile, {
        id: string;
    }, import("mongoose").ResolveSchemaOptions<import("mongoose").DefaultSchemaOptions>> & Omit<Profile & Required<{
        _id: Types.ObjectId;
    }> & {
        __v: number;
    }, "id"> & {
        id: string;
    }> | undefined;
    loyaltyPoints?: import("mongoose").SchemaDefinitionProperty<number | undefined, Profile, Document<unknown, {}, Profile, {
        id: string;
    }, import("mongoose").ResolveSchemaOptions<import("mongoose").DefaultSchemaOptions>> & Omit<Profile & Required<{
        _id: Types.ObjectId;
    }> & {
        __v: number;
    }, "id"> & {
        id: string;
    }> | undefined;
    membershipExpiresAt?: import("mongoose").SchemaDefinitionProperty<Date | undefined, Profile, Document<unknown, {}, Profile, {
        id: string;
    }, import("mongoose").ResolveSchemaOptions<import("mongoose").DefaultSchemaOptions>> & Omit<Profile & Required<{
        _id: Types.ObjectId;
    }> & {
        __v: number;
    }, "id"> & {
        id: string;
    }> | undefined;
    isPhoneVerified?: import("mongoose").SchemaDefinitionProperty<boolean | undefined, Profile, Document<unknown, {}, Profile, {
        id: string;
    }, import("mongoose").ResolveSchemaOptions<import("mongoose").DefaultSchemaOptions>> & Omit<Profile & Required<{
        _id: Types.ObjectId;
    }> & {
        __v: number;
    }, "id"> & {
        id: string;
    }> | undefined;
    isEmailVerified?: import("mongoose").SchemaDefinitionProperty<boolean | undefined, Profile, Document<unknown, {}, Profile, {
        id: string;
    }, import("mongoose").ResolveSchemaOptions<import("mongoose").DefaultSchemaOptions>> & Omit<Profile & Required<{
        _id: Types.ObjectId;
    }> & {
        __v: number;
    }, "id"> & {
        id: string;
    }> | undefined;
    isIdentityVerified?: import("mongoose").SchemaDefinitionProperty<boolean | undefined, Profile, Document<unknown, {}, Profile, {
        id: string;
    }, import("mongoose").ResolveSchemaOptions<import("mongoose").DefaultSchemaOptions>> & Omit<Profile & Required<{
        _id: Types.ObjectId;
    }> & {
        __v: number;
    }, "id"> & {
        id: string;
    }> | undefined;
    identityDocument?: import("mongoose").SchemaDefinitionProperty<IdentityDocument | undefined, Profile, Document<unknown, {}, Profile, {
        id: string;
    }, import("mongoose").ResolveSchemaOptions<import("mongoose").DefaultSchemaOptions>> & Omit<Profile & Required<{
        _id: Types.ObjectId;
    }> & {
        __v: number;
    }, "id"> & {
        id: string;
    }> | undefined;
    followingShopIds?: import("mongoose").SchemaDefinitionProperty<string[] | undefined, Profile, Document<unknown, {}, Profile, {
        id: string;
    }, import("mongoose").ResolveSchemaOptions<import("mongoose").DefaultSchemaOptions>> & Omit<Profile & Required<{
        _id: Types.ObjectId;
    }> & {
        __v: number;
    }, "id"> & {
        id: string;
    }> | undefined;
    followersCount?: import("mongoose").SchemaDefinitionProperty<number | undefined, Profile, Document<unknown, {}, Profile, {
        id: string;
    }, import("mongoose").ResolveSchemaOptions<import("mongoose").DefaultSchemaOptions>> & Omit<Profile & Required<{
        _id: Types.ObjectId;
    }> & {
        __v: number;
    }, "id"> & {
        id: string;
    }> | undefined;
    stats?: import("mongoose").SchemaDefinitionProperty<ProfileStats | undefined, Profile, Document<unknown, {}, Profile, {
        id: string;
    }, import("mongoose").ResolveSchemaOptions<import("mongoose").DefaultSchemaOptions>> & Omit<Profile & Required<{
        _id: Types.ObjectId;
    }> & {
        __v: number;
    }, "id"> & {
        id: string;
    }> | undefined;
    metadata?: import("mongoose").SchemaDefinitionProperty<Record<string, any> | undefined, Profile, Document<unknown, {}, Profile, {
        id: string;
    }, import("mongoose").ResolveSchemaOptions<import("mongoose").DefaultSchemaOptions>> & Omit<Profile & Required<{
        _id: Types.ObjectId;
    }> & {
        __v: number;
    }, "id"> & {
        id: string;
    }> | undefined;
    lastActiveAt?: import("mongoose").SchemaDefinitionProperty<Date | undefined, Profile, Document<unknown, {}, Profile, {
        id: string;
    }, import("mongoose").ResolveSchemaOptions<import("mongoose").DefaultSchemaOptions>> & Omit<Profile & Required<{
        _id: Types.ObjectId;
    }> & {
        __v: number;
    }, "id"> & {
        id: string;
    }> | undefined;
    isDeleted?: import("mongoose").SchemaDefinitionProperty<boolean | undefined, Profile, Document<unknown, {}, Profile, {
        id: string;
    }, import("mongoose").ResolveSchemaOptions<import("mongoose").DefaultSchemaOptions>> & Omit<Profile & Required<{
        _id: Types.ObjectId;
    }> & {
        __v: number;
    }, "id"> & {
        id: string;
    }> | undefined;
    deletedAt?: import("mongoose").SchemaDefinitionProperty<Date | undefined, Profile, Document<unknown, {}, Profile, {
        id: string;
    }, import("mongoose").ResolveSchemaOptions<import("mongoose").DefaultSchemaOptions>> & Omit<Profile & Required<{
        _id: Types.ObjectId;
    }> & {
        __v: number;
    }, "id"> & {
        id: string;
    }> | undefined;
    shopConfig?: import("mongoose").SchemaDefinitionProperty<ShopConfig | undefined, Profile, Document<unknown, {}, Profile, {
        id: string;
    }, import("mongoose").ResolveSchemaOptions<import("mongoose").DefaultSchemaOptions>> & Omit<Profile & Required<{
        _id: Types.ObjectId;
    }> & {
        __v: number;
    }, "id"> & {
        id: string;
    }> | undefined;
    _id?: import("mongoose").SchemaDefinitionProperty<Types.ObjectId, Profile, Document<unknown, {}, Profile, {
        id: string;
    }, import("mongoose").ResolveSchemaOptions<import("mongoose").DefaultSchemaOptions>> & Omit<Profile & Required<{
        _id: Types.ObjectId;
    }> & {
        __v: number;
    }, "id"> & {
        id: string;
    }> | undefined;
}, Profile>;
