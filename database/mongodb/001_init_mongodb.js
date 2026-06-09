// ============================================================================
// TAFU MongoDB Collections Initialization Script
// Version: 1.0.0
// Date: 2026-01-07
// Purpose: Initialize all MongoDB collections with proper indexes and validation
// Usage: Run with: mongosh < 001_init_mongodb.js
// ============================================================================

// ===================
// Database: tafu_profile
// ===================
db = db.getSiblingDB("tafu_profile");

// Collection: profiles
db.createCollection("profiles", {
    validator: {
        $jsonSchema: {
            bsonType: "object",
            required: ["userId", "displayName"],
            properties: {
                userId: {
                    bsonType: "string",
                    description: "UUID from Identity Service"
                },
                displayName: {
                    bsonType: "string",
                    minLength: 1,
                    maxLength: 100
                },
                email: {
                    bsonType: "string"
                },
                avatarUrl: {
                    bsonType: "string"
                },
                bio: {
                    bsonType: "string",
                    maxLength: 500
                },
                shopConfig: {
                    bsonType: "object",
                    properties: {
                        shopName: { bsonType: "string" },
                        description: { bsonType: "string" },
                        logoUrl: { bsonType: "string" },
                        pickupAddressId: { bsonType: "objectId" }
                    }
                }
            }
        }
    }
});

db.profiles.createIndex({ userId: 1 }, { unique: true });
db.profiles.createIndex(
    { "shopConfig.shopName": 1 },
    { unique: true, partialFilterExpression: { "shopConfig.shopName": { $exists: true } } }
);
db.profiles.createIndex({ email: 1 }, { sparse: true });
db.profiles.createIndex({ createdAt: -1 });

// Collection: addresses
db.createCollection("addresses", {
    validator: {
        $jsonSchema: {
            bsonType: "object",
            required: ["userId", "contactName", "phone", "provinceCode", "districtCode", "wardCode", "streetLine"],
            properties: {
                userId: { bsonType: "string" },
                contactName: { bsonType: "string", minLength: 1 },
                phone: { bsonType: "string", pattern: "^[0-9]{10,11}$" },
                provinceCode: { bsonType: "string" },
                districtCode: { bsonType: "string" },
                wardCode: { bsonType: "string" },
                streetLine: { bsonType: "string" },
                fullAddress: { bsonType: "string" },
                isDefault: { bsonType: "bool" },
                type: { enum: ["HOME", "OFFICE"] }
            }
        }
    }
});

db.addresses.createIndex({ userId: 1 });
db.addresses.createIndex({ userId: 1, isDefault: 1 });

print("tafu_profile database initialized");

// ===================
// Database: tafu_catalog
// ===================
db = db.getSiblingDB("tafu_catalog");

// Collection: categories
db.createCollection("categories");
db.categories.createIndex({ slug: 1 }, { unique: true });
db.categories.createIndex({ parentId: 1 });
db.categories.createIndex({ name: "text" });

// Collection: brands
db.createCollection("brands");
db.brands.createIndex({ slug: 1 }, { unique: true });
db.brands.createIndex({ status: 1 });
db.brands.createIndex({ name: "text" });

// Collection: products
db.createCollection("products", {
    validator: {
        $jsonSchema: {
            bsonType: "object",
            required: ["name", "slug", "categoryId", "status"],
            properties: {
                name: { bsonType: "string", minLength: 1, maxLength: 255 },
                slug: { bsonType: "string" },
                categoryId: { bsonType: "objectId" },
                brandId: { bsonType: "objectId" },
                thumbnail: { bsonType: "string" },
                images: { bsonType: "array", items: { bsonType: "string" } },
                videoUrl: { bsonType: "string" },
                description: { bsonType: "string" },
                specs: { bsonType: "object" },
                variations: {
                    bsonType: "array",
                    items: {
                        bsonType: "object",
                        required: ["sku", "price"],
                        properties: {
                            sku: { bsonType: "string" },
                            price: { bsonType: "number", minimum: 0 },
                            stock: { bsonType: "int", minimum: 0 },
                            imageUrl: { bsonType: "string" },
                            attributes: { bsonType: "object" },
                            tierIndex: { bsonType: "array" },
                            metadata: { bsonType: "object" }
                        }
                    }
                },
                metadata: { bsonType: "object" },
                status: { enum: ["DRAFT", "PUBLISHED", "ARCHIVED"] }
            }
        }
    }
});

db.products.createIndex({ slug: 1 }, { unique: true });
db.products.createIndex({ categoryId: 1 });
db.products.createIndex({ brandId: 1 });
db.products.createIndex({ status: 1, createdAt: -1 });
db.products.createIndex({ "variations.sku": 1 });
db.products.createIndex({ name: "text", description: "text" });
db.products.createIndex({ "metadata.isFlashSale": 1 }, { sparse: true });
db.products.createIndex({ "metadata.campaignId": 1 }, { sparse: true });

print("tafu_catalog database initialized");

// ===================
// Database: tafu_cart
// ===================
db = db.getSiblingDB("tafu_cart");

// Collection: carts (MongoDB backup for Redis)
db.createCollection("carts");
db.carts.createIndex({ _id: 1 });  // _id = userId
db.carts.createIndex({ updatedAt: 1 }, { expireAfterSeconds: 2592000 });  // 30 days TTL

print("tafu_cart database initialized");

// ===================
// Database: tafu_review
// ===================
db = db.getSiblingDB("tafu_review");

// Collection: reviews
db.createCollection("reviews", {
    validator: {
        $jsonSchema: {
            bsonType: "object",
            required: ["userId", "productId", "orderId", "rating"],
            properties: {
                userId: { bsonType: "string" },
                userName: { bsonType: "string" },
                userAvatar: { bsonType: "string" },
                productId: { bsonType: "string" },
                orderId: { bsonType: "string" },
                rating: { bsonType: "int", minimum: 1, maximum: 5 },
                content: { bsonType: "string", maxLength: 2000 },
                images: { bsonType: "array", items: { bsonType: "string" }, maxItems: 5 },
                isPurchased: { bsonType: "bool" },
                reply: {
                    bsonType: "object",
                    properties: {
                        content: { bsonType: "string" },
                        repliedAt: { bsonType: "date" }
                    }
                },
                status: { enum: ["VISIBLE", "HIDDEN", "PENDING"] }
            }
        }
    }
});

db.reviews.createIndex({ productId: 1, createdAt: -1 });
db.reviews.createIndex({ userId: 1 });
db.reviews.createIndex({ orderId: 1, productId: 1 }, { unique: true });
db.reviews.createIndex({ status: 1, productId: 1 });
db.reviews.createIndex({ rating: 1 });

// Collection: product_ratings (Materialized View)
db.createCollection("product_ratings");
db.product_ratings.createIndex({ _id: 1 });  // _id = productId

print("tafu_review database initialized");

// ===================
// Database: tafu_notification
// ===================
db = db.getSiblingDB("tafu_notification");

// Collection: notification_logs
db.createCollection("notification_logs", {
    validator: {
        $jsonSchema: {
            bsonType: "object",
            required: ["type", "status", "recipient"],
            properties: {
                userId: { bsonType: "string" },
                type: { enum: ["EMAIL", "PUSH", "SMS"] },
                status: { enum: ["PENDING", "SENT", "FAILED", "RETRYING"] },
                recipient: { bsonType: "string" },
                template: { bsonType: "string" },
                subject: { bsonType: "string" },
                payload: { bsonType: "object" },
                error: { bsonType: "string" },
                retryCount: { bsonType: "int" },
                sentAt: { bsonType: "date" }
            }
        }
    }
});

db.notification_logs.createIndex({ userId: 1 });
db.notification_logs.createIndex({ status: 1 });
db.notification_logs.createIndex({ createdAt: -1 });
db.notification_logs.createIndex({ type: 1, status: 1 });
db.notification_logs.createIndex(
    { createdAt: 1 },
    { expireAfterSeconds: 7776000 }  // 90 days TTL
);

// Collection: notification_templates
db.createCollection("notification_templates");
db.notification_templates.createIndex({ name: 1 }, { unique: true });
db.notification_templates.createIndex({ type: 1 });

print("tafu_notification database initialized");

// ===================
// Database: tafu_media
// ===================
db = db.getSiblingDB("tafu_media");

// Collection: media_files
db.createCollection("media_files", {
    validator: {
        $jsonSchema: {
            bsonType: "object",
            required: ["fileKey", "userId", "mimeType"],
            properties: {
                fileKey: { bsonType: "string" },
                userId: { bsonType: "string" },
                fileName: { bsonType: "string" },
                mimeType: { bsonType: "string" },
                size: { bsonType: "long" },
                purpose: { enum: ["product_image", "avatar", "shop_logo", "review_image", "other"] },
                variants: {
                    bsonType: "object",
                    properties: {
                        original: { bsonType: "string" },
                        thumbnail: { bsonType: "string" },
                        medium: { bsonType: "string" }
                    }
                },
                status: { enum: ["PENDING", "PROCESSED", "FAILED"] }
            }
        }
    }
});

db.media_files.createIndex({ fileKey: 1 }, { unique: true });
db.media_files.createIndex({ userId: 1 });
db.media_files.createIndex({ purpose: 1 });
db.media_files.createIndex({ status: 1 });
db.media_files.createIndex({ createdAt: -1 });

print("tafu_media database initialized");

print("\n========================================");
print("All MongoDB databases initialized.");
print("========================================");
