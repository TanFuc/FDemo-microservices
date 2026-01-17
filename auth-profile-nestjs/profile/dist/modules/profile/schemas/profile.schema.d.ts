import { Document, HydratedDocument, Types } from 'mongoose';
export type ProfileDocument = HydratedDocument<Profile>;
export declare class ShopConfig {
    shopName: string;
    description?: string;
    logoUrl?: string;
    pickupAddressId?: Types.ObjectId;
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
    shopName?: import("mongoose").SchemaDefinitionProperty<string, ShopConfig, Document<unknown, {}, ShopConfig, {
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
    pickupAddressId?: import("mongoose").SchemaDefinitionProperty<Types.ObjectId | undefined, ShopConfig, Document<unknown, {}, ShopConfig, {
        id: string;
    }, import("mongoose").ResolveSchemaOptions<import("mongoose").DefaultSchemaOptions>> & Omit<ShopConfig & {
        _id: Types.ObjectId;
    } & {
        __v: number;
    }, "id"> & {
        id: string;
    }> | undefined;
}, ShopConfig>;
export declare class Profile extends Document {
    userId: string;
    displayName: string;
    email?: string;
    avatarUrl?: string;
    bio?: string;
    shopConfig?: ShopConfig;
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
    avatarUrl?: import("mongoose").SchemaDefinitionProperty<string | undefined, Profile, Document<unknown, {}, Profile, {
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
