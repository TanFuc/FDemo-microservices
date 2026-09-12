# API Documentation - Microservices E-Commerce Platform

> **Purpose**: Comprehensive API reference for frontend and client implementation
> **Last Updated**: 2026-01-13

---

## Table of Contents

1. [Auth Service](#1-auth-service)
2. [Profile Service](#2-profile-service)
3. [Catalog Service](#3-catalog-service)
4. [Cart Service](#4-cart-service)
5. [Order Service](#5-order-service)
6. [Payment Service](#6-payment-service)
7. [Inventory Service](#7-inventory-service)
8. [Logistic Service](#8-logistic-service)
9. [Media Service](#9-media-service)
10. [Review Service](#10-review-service)
11. [Search Service](#11-search-service)
12. [Campaign Service](#12-campaign-service)
13. [Analytics Service](#13-analytics-service)
14. [Notification Service](#14-notification-service)

---

## Base URL

```
Production: https://api.example.com
Development: http://localhost:8080
```

## Authentication

Most API endpoints require a JWT Bearer token in the Authorization header:

```
Authorization: Bearer <access_token>
```

---

## 1. Auth Service

### 1.1 Register Account

**POST** `/api/v1/identity/auth/register`

**Purpose**: Register a new user account with email and password credentials

**Usage Context**: Registration screen, new user onboarding

**Request Body (required)**:

```json
{
  "email": "user@example.com", // required, email format
  "password": "password123", // required, min 8, max 100 chars
  "fullName": "Nguyen Van A" // required, max 255 chars
}
```

**Response (201)**:

```json
{
  "success": true,
  "data": {
    "user": {
      "id": "uuid",
      "email": "user@example.com",
      "fullName": "Nguyen Van A",
      "isActive": true,
      "roles": ["user"],
      "createdAt": "2026-01-13T10:00:00Z",
      "updatedAt": "2026-01-13T10:00:00Z"
    }
  },
  "message": "User registered successfully"
}
```

**Errors**: 400 (validation), 409 (email already registered)

---

### 1.2 User Login

**POST** `/api/v1/identity/auth/login`

**Purpose**: Authenticate user credentials and return JWT access and refresh token pair

**Usage Context**: Login screen, authentication modal

**Headers (optional)**:

- `X-Device-ID`: Unique device identifier used for session management and device tracking

**Request Body (required)**:

```json
{
  "email": "user@example.com", // required
  "password": "password123" // required
}
```

**Response (200)**:

```json
{
  "success": true,
  "data": {
    "user": {
      "id": "uuid",
      "email": "user@example.com",
      "fullName": "Nguyen Van A",
      "isActive": true,
      "roles": ["user"],
      "createdAt": "2026-01-13T10:00:00Z",
      "updatedAt": "2026-01-13T10:00:00Z"
    },
    "tokens": {
      "accessToken": "jwt_access_token",
      "refreshToken": "jwt_refresh_token",
      "expiresIn": 3600
    }
  }
}
```

**Errors**: 400 (validation), 401 (sai credentials)

---

### 1.3 Refresh Token

**POST** `/api/v1/identity/auth/refresh`

**Purpose**: Obtain a fresh access token using a valid refresh token

**Usage Context**: Transparent token refresh via HTTP interceptor upon 401 Unauthorized

**Headers (optional)**:

- `X-Device-ID`: Device ID

**Request Body (required)**:

```json
{
  "refreshToken": "jwt_refresh_token" // required
}
```

**Response (200)**:

```json
{
  "success": true,
  "data": {
    "accessToken": "new_jwt_access_token",
    "refreshToken": "new_jwt_refresh_token",
    "expiresIn": 3600
  }
}
```

---

### 1.4 Logout

**POST** `/api/v1/identity/auth/logout`

**Auth**: Required

**Purpose**: Invalidate active session tokens and blacklist access token

**Usage Context**: User sign-out action

**Request Body (optional)**:

```json
{
  "deviceId": "device_uuid" // optional, logout specific device
}
```

**Response (200)**:

```json
{
  "success": true,
  "message": "Logged out successfully"
}
```

---

### 1.5 Revoke All Sessions / Logout Everywhere

**POST** `/api/v1/identity/auth/logout-all`

**Auth**: Required

**Purpose**: Revoke all active sessions and refresh tokens across all devices

**Usage Context**: Account security center, post-password reset

**Response (200)**:

```json
{
  "success": true,
  "message": "Logged out from all devices successfully"
}
```

---

### 1.6 Get Current User Identity

**GET** `/api/v1/identity/auth/profile`

**Auth**: Required

**Purpose**: Retrieve authenticated user identity and assigned roles

**Usage Context**: App bar, user greeting, profile overview

**Response (200)**:

```json
{
  "success": true,
  "data": {
    "id": "uuid",
    "email": "user@example.com",
    "fullName": "Nguyen Van A",
    "isActive": true,
    "roles": ["user"],
    "createdAt": "2026-01-13T10:00:00Z",
    "updatedAt": "2026-01-13T10:00:00Z"
  }
}
```

---

### 1.7 List Active Sessions

**GET** `/api/v1/identity/auth/sessions`

**Auth**: Required

**Purpose**: Retrieve all active login sessions and device footprints

**Usage Context**: Connected devices management, security audit

**Response (200)**:

```json
{
  "success": true,
  "data": [
    {
      "id": "session_uuid",
      "deviceId": "device_uuid",
      "deviceName": "Chrome on Windows",
      "browser": "Chrome",
      "os": "Windows 10",
      "ipAddress": "192.168.1.1",
      "location": "Ho Chi Minh, Vietnam",
      "createdAt": "2026-01-13T10:00:00Z",
      "expiresAt": "2026-01-14T10:00:00Z",
      "isCurrent": true
    }
  ]
}
```

---

## 2. Profile Service

### 2.1 Get My Profile

**GET** `/api/v1/profile`

**Auth**: Required

**Purpose**: Retrieve complete user profile details including address book and shop metadata

**Usage Context**: Account profile settings screen

**Response (200)**:

```json
{
  "success": true,
  "data": {
    "userId": "uuid",
    "displayName": "Nguyen Van A",
    "avatar": "https://...",
    "phone": "0901234567",
    "dateOfBirth": "1990-01-01",
    "gender": "male",
    "shop": null,
    "createdAt": "2026-01-13T10:00:00Z"
  }
}
```

---

### 2.2 Update Profile

**PUT** `/api/v1/profile`

**Auth**: Required

**Purpose**: Update user demographic details (name, avatar, bio)

**Usage Context**: Edit profile form

**Request Body (all optional)**:

```json
{
  "displayName": "Nguyen Van B",
  "phone": "0909876543",
  "dateOfBirth": "1990-01-01",
  "gender": "male",
  "avatar": "https://..."
}
```

**Response (200)**: Updated profile object

---

### 2.3 List Shipping Addresses

**GET** `/api/v1/profile/addresses`

**Auth**: Required

**Purpose**: Retrieve all saved delivery addresses for the authenticated user

**Usage Context**: Checkout address selector, address book management

**Response (200)**:

```json
{
  "success": true,
  "data": [
    {
      "id": "address_uuid",
      "fullName": "Nguyen Van A",
      "phone": "0901234567",
      "address": "123 Nguyen Hue",
      "ward": "Ben Nghe",
      "district": "Quan 1",
      "city": "Ho Chi Minh",
      "country": "Vietnam",
      "postalCode": "70000",
      "isDefault": true
    }
  ]
}
```

---

### 2.4 Add Shipping Address

**POST** `/api/v1/profile/addresses`

**Auth**: Required

**Purpose**: Add a new shipping destination address

**Usage Context**: Checkout shipping step, address book form

**Request Body (required)**:

```json
{
  "fullName": "Nguyen Van A", // required
  "phone": "0901234567", // required
  "address": "123 Nguyen Hue", // required
  "ward": "Ben Nghe", // optional
  "district": "Quan 1", // required
  "city": "Ho Chi Minh", // required
  "country": "Vietnam", // required
  "postalCode": "70000", // optional
  "isDefault": false // optional
}
```

**Response (201)**: Created address object

---

### 2.5 Update Shipping Address

**PUT** `/api/v1/profile/addresses/:id`

**Auth**: Required

**Path Params**: `id` - Address ID (required)

**Purpose**: Modify an existing delivery address

**Request Body**: Same as create (all optional)

---

### 2.6 Delete Shipping Address

**DELETE** `/api/v1/profile/addresses/:id`

**Auth**: Required

**Path Params**: `id` - Address ID (required)

**Purpose**: Remove a saved shipping address

**Response (200)**:

```json
{
  "success": true,
  "message": "Address deleted successfully"
}
```

---

### 2.7 Register Merchant / Shop Profile

**POST** `/api/v1/profile/shop`

**Auth**: Required

**Purpose**: Register and onboard as a marketplace seller

**Usage Context**: Seller onboarding application form

**Request Body (required)**:

```json
{
  "shopName": "My Shop", // required
  "description": "Shop description",
  "logo": "https://...",
  "address": "123 ABC Street", // required
  "phone": "0901234567", // required
  "taxCode": "123456789" // optional
}
```

---

## 3. Catalog Service

### 3.1 List Products

**GET** `/api/v1/catalog/products`

**Auth**: Not required

**Purpose**: Retrieve paginated list of catalog products with dynamic filtering

**Usage Context**: Home page storefront, category browsing, product catalog

**Query Params (all optional)**:
| Param | Type | Description |
|-------|------|-------------|
| categoryId | string | Filter theo category |
| brandId | string | Filter theo brand |
| status | string | Filter theo status (active/inactive) |
| limit | int | Number of items per page (default: 20) |
| offset | int | Offset for pagination |

**Response (200)**:

```json
{
  "products": [
    {
      "id": "product_id",
      "name": "Product Name",
      "slug": "product-name",
      "categoryId": "cat_id",
      "brandId": "brand_id",
      "thumbnail": "https://...",
      "images": ["https://..."],
      "description": "...",
      "specs": { "color": "red" },
      "variations": [
        {
          "skuId": "sku_123",
          "attributes": { "size": "M" },
          "price": 100000,
          "stock": 50
        }
      ],
      "status": "active",
      "createdAt": "2026-01-13T10:00:00Z"
    }
  ],
  "total": 100
}
```

---

### 3.2 Get Product Details

**GET** `/api/v1/catalog/products/:id`

**Auth**: Not required

**Path Params**: `id` - Product ID (required)

**Purpose**: Retrieve detailed specifications, variations, and rich media for a single product

**Usage Context**: Product detail view (PDP)

**Response (200)**: Single product object

---

### 3.3 Get Product by Slug

**GET** `/api/v1/catalog/products/slug/:slug`

**Auth**: Not required

**Path Params**: `slug` - Product slug (required)

**Purpose**: Retrieve product details using its SEO-friendly slug identifier

**Usage Context**: Canonical SEO URL routing, external link sharing

---

### 3.4 Create Product

**POST** `/api/v1/catalog/products`

**Auth**: Required (Seller)

**Purpose**: Create a new product entry with SKU variations and specs

**Usage Context**: Seller portal product creation wizard

**Request Body**:

```json
{
  "name": "Product Name", // required
  "categoryId": "cat_id", // required, ObjectId
  "brandId": "brand_id", // required, ObjectId
  "thumbnail": "https://...", // optional
  "images": ["https://..."], // optional
  "videoUrl": "https://...", // optional
  "description": "...", // optional
  "specs": { "color": "red" }, // optional
  "variations": [
    // optional
    {
      "attributes": { "size": "M" },
      "price": 100000,
      "stock": 50
    }
  ],
  "metadata": {} // optional
}
```

**Response (201)**: Created product

---

### 3.5 Update Product

**PUT** `/api/v1/catalog/products/:id`

**Auth**: Required (Seller)

**Request Body**: Same as create + `status` field

---

### 3.6 Delete Product

**DELETE** `/api/v1/catalog/products/:id`

**Auth**: Required (Seller)

**Response (204)**: No content

---

### 3.7 List Categories

**GET** `/api/v1/catalog/categories`

**Auth**: Not required

**Purpose**: Retrieve recursive category tree and attribute definitions

**Usage Context**: Store navigation navbar, category sidebar filter

**Response (200)**:

```json
[
  {
    "id": "cat_id",
    "name": "Electronics",
    "slug": "electronics",
    "imageUrl": "https://...",
    "parentId": null,
    "attributeDefinitions": [
      { "name": "RAM", "type": "select", "values": ["4GB", "8GB"] }
    ]
  }
]
```

---

### 3.8 Create Category

**POST** `/api/v1/catalog/categories`

**Auth**: Required (Admin)

**Request Body**:

```json
{
  "name": "Electronics", // required
  "imageUrl": "https://...", // optional
  "parentId": "parent_cat_id", // optional
  "attributeDefinitions": [
    // optional
    { "name": "RAM", "type": "select", "values": ["4GB", "8GB"] }
  ]
}
```

---

### 3.9 List Brands

**GET** `/api/v1/catalog/brands`

**Auth**: Not required (GET), Required (CUD operations)

---

## 4. Cart Service

### 4.1 Get Shopping Cart

**GET** `/api/v1/cart/:userId`

**Auth**: Required

**Path Params**: `userId` - User ID (required)

**Purpose**: Retrieve active shopping cart items and calculated pricing summary

**Usage Context**: Cart page, header mini-cart dropdown, checkout screen

**Response (200)**:

```json
{
  "userId": "user_uuid",
  "items": [
    {
      "skuId": "sku_123",
      "name": "Product Name - Size M",
      "price": 100000,
      "quantity": 2,
      "thumbnail": "https://...",
      "selected": true,
      "addedAt": 1704067200
    }
  ],
  "totalItems": 2,
  "updatedAt": "2026-01-13T10:00:00Z"
}
```

---

### 4.2 Add Item to Cart

**POST** `/api/v1/cart/:userId/items`

**Auth**: Required

**Path Params**: `userId` - User ID (required)

**Purpose**: Atomically increment item quantity or add new SKU to user cart

**Usage Context**: "Add to cart" button

**Request Body**:

```json
{
  "skuId": "sku_123", // required
  "name": "Product Name", // required
  "price": 100000, // required, > 0
  "quantity": 1, // optional, default 1
  "thumbnail": "https://...", // optional
  "selected": true // optional
}
```

**Response (201)**:

```json
{
  "message": "item added to cart"
}
```

---

### 4.3 Remove Item from Cart

**DELETE** `/api/v1/cart/:userId/items/:skuId`

**Auth**: Required

**Path Params**:

- `userId` - User ID (required)
- `skuId` - SKU ID (required)

**Response (200)**:

```json
{
  "message": "item removed from cart"
}
```

---

### 4.4 Update Item Quantity

**PUT** `/api/v1/cart/:userId/items/:skuId/quantity`

**Auth**: Required

**Request Body**:

```json
{
  "quantity": 3 // required, > 0
}
```

---

### 4.5 Update Item Selection

**PUT** `/api/v1/cart/:userId/items/:skuId/selection`

**Auth**: Required

**Purpose**: Select or deselect specific items to include in immediate checkout

**Request Body**:

```json
{
  "selected": true
}
```

---

### 4.6 Clear Shopping Cart

**DELETE** `/api/v1/cart/:userId`

**Auth**: Required

**Purpose**: Purge all items from the current user shopping cart

---

## 5. Order Service

### 5.1 Create Order

**POST** `/api/v1/orders`

**Auth**: Required

**Purpose**: Create a new pending order from selected cart items and initiate Saga reservation

**Usage Context**: Checkout "Place Order" confirmation button

**Request Body**:

```json
{
  "user_id": "uuid", // required
  "items": [
    // required, min 1
    {
      "product_id": "prod_id", // required
      "sku_id": "sku_id", // required
      "product_name": "Product Name", // required
      "sku_code": "SKU-001", // required
      "thumbnail": "https://...", // optional
      "quantity": 2, // required, min 1
      "unit_price": "100000" // required, decimal
    }
  ],
  "shipping_address": {
    // required
    "full_name": "Nguyen Van A", // required
    "phone": "0901234567", // required
    "address": "123 ABC", // required
    "ward": "Ward 1", // optional
    "district": "District 1", // required
    "city": "Ho Chi Minh", // required
    "country": "Vietnam", // required
    "postal_code": "70000" // optional
  },
  "payment_method": "MOMO", // required: MOMO|COD|STRIPE
  "shipping_fee": "30000", // optional, decimal
  "discount_amount": "10000" // optional, decimal
}
```

**Response (201)**:

```json
{
  "id": "order_uuid",
  "user_id": "user_uuid",
  "total_amount": "200000",
  "shipping_fee": "30000",
  "discount_amount": "10000",
  "final_amount": "220000",
  "status": "PENDING",
  "payment_method": "MOMO",
  "shipping_address": {...},
  "items": [...],
  "created_at": "2026-01-13T10:00:00Z",
  "updated_at": "2026-01-13T10:00:00Z"
}
```

---

### 5.2 Get Order Details

**GET** `/api/v1/orders/:id`

**Auth**: Required

**Path Params**: `id` - Order UUID (required)

**Purpose**: Retrieve complete order metadata, shipping progress, and item snapshots

**Usage Context**: Order tracking page, buyer order receipt

---

### 5.3 List Orders

**GET** `/api/v1/orders/user/:userId`

**Auth**: Required

**Path Params**: `userId` - User UUID (required)

**Query Params (optional)**:

- `limit`: int (default 10)
- `offset`: int (default 0)

**Purpose**: Retrieve paginated customer order history with status filters

**Response (200)**:

```json
{
  "orders": [...],
  "limit": 10,
  "offset": 0
}
```

---

### 5.4 Cancel Order

**POST** `/api/v1/orders/:id/cancel`

**Auth**: Required

**Purpose**: Cancel an order and trigger Saga stock release compensation (only valid when status is PENDING)

**Response (200)**:

```json
{
  "message": "Order cancelled successfully"
}
```

**Errors**: 404 (not found), 409 (cannot cancel)

---

### 5.5 Mark Order as Paid

**POST** `/api/v1/orders/:id/pay`

**Auth**: Required (Internal/Admin)

**Purpose**: Internal / admin transition to mark order status as PAID

---

### 5.6 Mark Order as Shipped

**POST** `/api/v1/orders/:id/ship`

**Auth**: Required (Admin/Seller)

**Purpose**: Internal transition updating order status to SHIPPED

---

### 5.7 Mark Order as Completed

**POST** `/api/v1/orders/:id/complete`

**Auth**: Required

**Purpose**: Finalize order lifecycle upon delivery confirmation

---

## 6. Payment Service

### 6.1 Create Payment Intent

**POST** `/api/v1/payments`

**Auth**: Required

**Purpose**: Initialize payment transaction and obtain client gateway checkout URL or token

**Usage Context**: Post-order placement redirect to payment gateway

**Request Body**:

```json
{
  "order_id": "order_uuid", // required
  "user_id": "user_uuid", // required
  "amount": "220000", // required, decimal string
  "currency": "VND", // required: VND|USD
  "provider": "MOMO", // required: MOMO|STRIPE|ZALOPAY
  "description": "Payment for...", // optional
  "callback_url": "https://...", // required
  "return_url": "https://...", // required
  "metadata": {} // optional
}
```

**Response (201)**:

```json
{
  "transaction_id": "tx_uuid",
  "payment_url": "https://payment-gateway.com/...",
  "provider_tx_id": "MOMO123456"
}
```

---

### 6.2 Get Payment Details

**GET** `/api/v1/payments/:id`

**Auth**: Required

**Path Params**: `id` - Transaction UUID (required)

---

### 6.3 Get Payment by Order ID

**GET** `/api/v1/payments/order/:orderId`

**Auth**: Required

**Path Params**: `orderId` - Order UUID (required)

**Response (200)**:

```json
{
  "transactions": [...],
  "count": 1
}
```

---

### 6.4 Payment Provider Webhook

**POST** `/api/v1/payments/webhook/:provider`

**Auth**: Not required (verified by signature)

**Path Params**: `provider` - Provider name (momo/stripe/zalopay)

**Purpose**: Ingest asynchronous payment status webhook callback from external provider

---

## 7. Inventory Service

### 7.1 Get Inventory Stock

**GET** `/api/v1/inventory/products/:skuId`

**Auth**: Required

**Path Params**: `skuId` - SKU ID (required)

**Purpose**: Query current available and reserved inventory counts for an SKU

**Response (200)**:

```json
{
  "sku_id": "sku_123",
  "total_stock": 100,
  "reserved_stock": 10,
  "available_stock": 90
}
```

---

### 7.2 Update Inventory Stock

**PUT** `/api/v1/inventory/products/:skuId`

**Auth**: Required (Admin/Seller)

**Request Body**:

```json
{
  "total_stock": 150
}
```

---

### 7.3 Get Stock Audit History

**GET** `/api/v1/inventory/products/:skuId/history`

**Auth**: Required

---

### 7.4 Reserve Stock (Internal)

**POST** `/api/v1/inventory/reserve`

**Purpose**: Atomically hold stock for an order via two-phase reservation

**Request Body**:

```json
{
  "order_id": "order_uuid",
  "items": [{ "sku_id": "sku_123", "quantity": 2 }]
}
```

---

### 7.5 Confirm Stock (Internal)

**POST** `/api/v1/inventory/confirm`

**Purpose**: Finalize deduction of previously reserved inventory upon payment confirmation

---

### 7.6 Release Stock (Internal)

**POST** `/api/v1/inventory/release`

**Purpose**: Release reserved inventory hold back to available stock on order cancellation

---

## 8. Logistic Service

### 8.1 Calculate Shipping Rates

**POST** `/api/v1/logistics/calculate-fee`

**Auth**: Required

**Purpose**: Calculate delivery fee across supported logistics carriers

**Usage Context**: Checkout shipping option selection

**Request Body**:

```json
{
  "provider": "GHN", // required: GHN|GHTK|VNPOST
  "from_district_id": 1454, // required
  "to_district_id": 1452, // required
  "weight_gram": 500, // required, min 1
  "insurance_value": 100000 // optional
}
```

**Response (200)**:

```json
{
  "success": true,
  "data": {
    "provider": "GHN",
    "fee": 25000,
    "from_cache": false
  }
}
```

---

### 8.2 Create Shipment Waybill

**POST** `/api/v1/logistics/shipments`

**Auth**: Required

**Request Body**:

```json
{
  "internal_order_id": "order_uuid",    // required
  "provider": "GHN",                     // required
  "sender": {                            // required
    "name": "Shop ABC",
    "phone": "0901234567",
    "address": "123 ABC",
    "ward_code": "20101",
    "district_id": 1454,
    "province_id": 202
  },
  "receiver": {...},                     // required (same structure)
  "parcels": [                           // required
    {
      "name": "Product 1",
      "quantity": 1,
      "weight_gram": 500,
      "value": 100000
    }
  ],
  "is_cod": true,                        // optional
  "cod_amount": 220000,                  // optional
  "note": "..."                          // optional
}
```

**Response (201)**:

```json
{
  "success": true,
  "data": {
    "id": "shipment_uuid",
    "tracking_code": "GHN123456",
    "label_url": "https://...",
    "shipping_fee": 25000,
    "provider": "GHN"
  }
}
```

---

### 8.3 Get Shipment Waybill Details

**GET** `/api/v1/logistics/shipments/:id`

**Auth**: Required

---

### 8.4 Get Shipment by Order ID

**GET** `/api/v1/logistics/shipments/order/:orderId`

**Auth**: Required

---

### 8.5 Track Shipment Progress

**GET** `/api/v1/logistics/shipments/:id/tracking`

**Auth**: Required

**Purpose**: Retrieve real-time carrier tracking checkpoints and delivery status

---

### 8.6 Carrier Status Webhook

**POST** `/api/v1/logistics/webhook/:provider`

**Auth**: Not required

---

## 9. Media Service

### 9.1 Get Presigned Upload URL

**POST** `/api/v1/media/presigned-url`

**Auth**: Required

**Purpose**: Generate a signed S3/MinIO PUT URL for direct client upload

**Usage Context**: Media upload widget, avatar change, product image gallery

**Request Body**:

```json
{
  "user_id": "user_uuid", // required
  "file_type": "image/jpeg", // required: image/jpeg, image/png, image/webp
  "purpose": "product" // optional: product, avatar, review
}
```

**Response (200)**:

```json
{
  "upload_url": "https://s3.../presigned-url",
  "file_key": "uploads/abc123.jpg",
  "public_url": "https://cdn.../uploads/abc123.jpg",
  "expires_at": "2026-01-13T11:00:00Z"
}
```

---

### 9.2 Confirm Upload Completion

**POST** `/api/v1/media/confirm`

**Auth**: Required

**Purpose**: Verify uploaded object in storage and trigger asynchronous image optimization

**Request Body**:

```json
{
  "file_key": "uploads/abc123.jpg" // required
}
```

---

### 9.3 Get Media Metadata

**GET** `/api/v1/media/:id`

**Auth**: Required

---

### 9.4 Delete Media Asset

**DELETE** `/api/v1/media/:id`

**Auth**: Required

---

## 10. Review Service

### 10.1 Create Verified Product Review

**POST** `/api/v1/reviews`

**Auth**: Required

**Purpose**: Submit rating and text review for an item from a verified completed order

**Usage Context**: Order history "Write Review" button

**Request Body**:

```json
{
  "userId": "uuid", // required
  "userName": "Nguyen Van A", // required
  "userAvatar": "https://...", // optional
  "productId": "product_uuid", // required
  "orderId": "order_uuid", // required
  "rating": 5, // required: 1-5
  "content": "Excellent build quality and fast shipping...", // required, 1-5000 chars
  "images": ["https://..."] // optional
}
```

**Response (201)**:

```json
{
  "success": true,
  "data": {...},
  "message": "Review created successfully"
}
```

---

### 10.2 Get Product Reviews

**GET** `/api/v1/reviews/products/:productId`

**Auth**: Not required

**Query Params (optional)**:

- `page`: int (default 1)
- `limit`: int (default 10, max 100)
- `sortBy`: string (createdAt|rating)

**Response (200)**:

```json
{
  "data": [...],
  "page": 1,
  "limit": 10,
  "totalItems": 50,
  "totalPages": 5
}
```

---

### 10.3 Get Product Rating Summary

**GET** `/api/v1/reviews/products/:productId/rating`

**Auth**: Not required

**Purpose**: Retrieve average star rating and breakdown distribution for a product

**Response (200)**:

```json
{
  "success": true,
  "data": {
    "averageRating": 4.5,
    "totalReviews": 50,
    "distribution": {
      "1": 2,
      "2": 3,
      "3": 5,
      "4": 15,
      "5": 25
    }
  }
}
```

---

### 10.4 Reply to Review (Seller)

**POST** `/api/v1/reviews/:id/reply`

**Auth**: Required (Seller)

**Request Body**:

```json
{
  "content": "Thank you for shopping with us! We appreciate your feedback." // required
}
```

---

### 10.5 Get My Reviews

**GET** `/api/v1/reviews/my`

**Auth**: Required

---

### 10.6 Update Review

**PUT** `/api/v1/reviews/:id`

**Auth**: Required

---

### 10.7 Delete Review

**DELETE** `/api/v1/reviews/:id`

**Auth**: Required

---

## 11. Search Service

### 11.1 Search Products (GET)

**GET** `/api/v1/search/products`

**Auth**: Not required

**Purpose**: Full-text product search with faceted filters and sorting

**Usage Context**: Global search bar, search catalog result page

**Query Params (all optional)**:
| Param | Type | Description |
|-------|------|-------------|
| keyword | string | Search query keyword |
| categoryId | string | Filter category |
| brandId | string | Filter brand |
| priceMin | float | Minimum price filter |
| priceMax | float | Maximum price filter |
| sortBy | string | Field sort (price, createdAt, rating) |
| sortOrder | string | asc/desc |
| page | int | Trang (default 1) |
| limit | int | Items/page (default 20) |

**Response (200)**:

```json
{
  "success": true,
  "data": {
    "products": [...],
    "total": 100,
    "page": 1,
    "limit": 20,
    "facets": {
      "categories": [...],
      "brands": [...],
      "priceRanges": [...]
    }
  }
}
```

---

### 11.2 Autocomplete Suggestions

**GET** `/api/v1/search/suggest`

**Auth**: Not required

**Query Params**:

- `keyword`: string (required)

**Purpose**: Instant autocomplete suggestions as user types in search input

---

### 11.3 Search Products by Category

**GET** `/api/v1/search/categories/:categoryId/products`

**Auth**: Not required

---

## 12. Campaign Service

### 12.1 List Active Campaigns

**GET** `/api/v1/campaigns`

**Auth**: Not required

**Purpose**: Retrieve list of active marketing campaigns and flash sales

**Usage Context**: Storefront promotional banner carousels, flash sales widget

---

### 12.2 Get Campaign Details

**GET** `/api/v1/campaigns/:id`

**Auth**: Not required

---

### 12.3 List Public Vouchers

**GET** `/api/v1/campaigns/vouchers/public`

**Auth**: Not required

**Purpose**: List vouchers available for customers to claim

---

### 12.4 Claim voucher

**POST** `/api/v1/campaigns/vouchers/claim`

**Auth**: Required

**Purpose**: Atomically claim and bind a voucher code to user wallet

**Request Body**:

```json
{
  "code": "SUMMER2026"
}
```

---

### 12.5 List My Vouchers

**GET** `/api/v1/campaigns/vouchers/my`

**Auth**: Required

**Purpose**: Retrieve all claimed and valid vouchers for current user

---

### 12.6 Apply voucher

**POST** `/api/v1/campaigns/vouchers/apply`

**Auth**: Required

**Purpose**: Evaluate voucher discount against current cart items

**Request Body**:

```json
{
  "code": "SUMMER2026",
  "items": [
    {
      "sku": "sku_123",
      "category": "electronics",
      "quantity": 2,
      "unit_price": 100000
    }
  ]
}
```

**Response (200)**:

```json
{
  "original_total": 200000,
  "discount": 20000,
  "final_total": 180000,
  "voucher_code": "SUMMER2026"
}
```

---

### 12.7 Create Campaign (Admin)

**POST** `/api/v1/campaigns`

**Auth**: Required (Admin)

---

### 12.8 Update Campaign (Admin)

**PUT** `/api/v1/campaigns/:id`

**Auth**: Required (Admin)

---

### 12.9 Delete Campaign (Admin)

**DELETE** `/api/v1/campaigns/:id`

**Auth**: Required (Admin)

---

### 12.10 Create Voucher for Campaign (Admin)

**POST** `/api/v1/campaigns/:id/vouchers`

**Auth**: Required (Admin)

---

## 13. Analytics Service

### 13.1 Dashboard Overview

**GET** `/api/v1/analytics/dashboard`

**Auth**: Required (Admin/Seller)

**Purpose**: Aggregated revenue, active order counts, and GMV metrics for analytics dashboard

---

### 13.2 Sales Analytics

**GET** `/api/v1/analytics/sales`

**Auth**: Required (Admin/Seller)

**Query Params (optional)**:

- `start`: ISO date
- `end`: ISO date

---

### 13.3 Product Analytics

**GET** `/api/v1/analytics/products/:productId`

**Auth**: Required

---

### 13.4 User Analytics

**GET** `/api/v1/analytics/users/:userId`

**Auth**: Required (Admin)

---

### 13.5 Track Event

**POST** `/api/v1/analytics/events`

**Auth**: Required

**Purpose**: Ingest real-time user behavior events into the analytics pipeline

**Usage Context**: Client-side event telemetry (page views, impressions, add-to-cart clicks)

**Request Body**:

```json
{
  "event_type": "product_view", // required
  "user_id": "user_uuid", // optional
  "session_id": "session_id", // optional
  "sku_id": "sku_123", // optional
  "metadata": "{...}" // optional, JSON string
}
```

**Response (202)**:

```json
{
  "status": "accepted",
  "event_id": "event_uuid"
}
```

---

## 14. Notification Service

### 14.1 List Notifications

**GET** `/api/v1/notifications`

**Auth**: Required

**Purpose**: Retrieve chronological notifications for authenticated user

**Query Params (optional)**:

- `page`: int
- `limit`: int

---

### 14.2 Get Unread Notification Count

**GET** `/api/v1/notifications/unread-count`

**Auth**: Required

**Response (200)**:

```json
{
  "count": 5
}
```

---

### 14.3 Mark Notification as Read

**PUT** `/api/v1/notifications/:id/read`

**Auth**: Required

---

### 14.4 Mark All Notifications as Read

**PUT** `/api/v1/notifications/read-all`

**Auth**: Required

---

### 14.5 Get Notification Preferences

**GET** `/api/v1/notifications/preferences`

**Auth**: Required

---

### 14.6 Update Notification Preferences

**PUT** `/api/v1/notifications/preferences`

**Auth**: Required

**Request Body**:

```json
{
  "email_enabled": true,
  "push_enabled": true,
  "sms_enabled": false,
  "order_updates": true,
  "promotions": true
}
```

---

## Error Response Format

All error responses adhere to the following standard JSON envelope:

```json
{
  "success": false,
  "error": {
    "code": "ERROR_CODE",
    "message": "Human readable message"
  }
}
```

### Common HTTP Status Codes

| Code | Meaning                        |
| ---- | ------------------------------ |
| 200  | Success                        |
| 201  | Created                        |
| 202  | Accepted (async)               |
| 204  | No Content                     |
| 400  | Bad Request (validation error) |
| 401  | Unauthorized                   |
| 403  | Forbidden                      |
| 404  | Not Found                      |
| 409  | Conflict                       |
| 422  | Unprocessable Entity           |
| 500  | Internal Server Error          |
| 503  | Service Unavailable            |

---

## Rate Limiting

- Default: 100 requests/minute per user
- Auth endpoints: 10 requests/minute per IP
- Search: 30 requests/minute per user

Headers returned:

```
X-RateLimit-Limit: 100
X-RateLimit-Remaining: 95
X-RateLimit-Reset: 1704067200
```

---

## Pagination

Standard pagination params:

- `page` or `offset`: Starting pagination index / offset
- `limit`: Number of items per page (maximum 100)

Response format:

```json
{
  "data": [...],
  "page": 1,
  "limit": 20,
  "total": 100,
  "totalPages": 5
}
```

---

_Document generated: 2026-01-13_
