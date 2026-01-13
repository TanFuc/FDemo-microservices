# API Documentation - Microservices E-Commerce Platform

> **Mục đích**: Tài liệu API đầy đủ cho việc implement frontend
> **Cập nhật**: 2026-01-13

---

## Mục lục

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

Hầu hết các API yêu cầu JWT token trong header:

```
Authorization: Bearer <access_token>
```

---

## 1. Auth Service

### 1.1 Đăng ký tài khoản

**POST** `/api/v1/identity/auth/register`

**Tác dụng**: Đăng ký người dùng mới với email và password

**Bối cảnh sử dụng**: Trang đăng ký, onboarding user mới

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

**Errors**: 400 (validation), 409 (email đã tồn tại)

---

### 1.2 Đăng nhập

**POST** `/api/v1/identity/auth/login`

**Tác dụng**: Xác thực user và trả về JWT tokens

**Bối cảnh sử dụng**: Trang đăng nhập

**Headers (optional)**:

- `X-Device-ID`: Device ID để quản lý sessions

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

**Tác dụng**: Lấy access token mới từ refresh token

**Bối cảnh sử dụng**: Khi access token hết hạn, automatic refresh

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

### 1.4 Đăng xuất

**POST** `/api/v1/identity/auth/logout`

**Auth**: Required

**Tác dụng**: Đăng xuất và vô hiệu hóa tokens

**Bối cảnh sử dụng**: Nút logout trong app

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

### 1.5 Đăng xuất tất cả thiết bị

**POST** `/api/v1/identity/auth/logout-all`

**Auth**: Required

**Tác dụng**: Đăng xuất khỏi tất cả thiết bị

**Bối cảnh sử dụng**: Security settings, đổi mật khẩu

**Response (200)**:

```json
{
  "success": true,
  "message": "Logged out from all devices successfully"
}
```

---

### 1.6 Lấy Profile

**GET** `/api/v1/identity/auth/profile`

**Auth**: Required

**Tác dụng**: Lấy thông tin user đang đăng nhập

**Bối cảnh sử dụng**: Header user info, profile page

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

### 1.7 Lấy danh sách Sessions

**GET** `/api/v1/identity/auth/sessions`

**Auth**: Required

**Tác dụng**: Lấy tất cả sessions đang active

**Bối cảnh sử dụng**: Security settings, quản lý devices

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

### 2.1 Lấy Profile của tôi

**GET** `/api/v1/profile`

**Auth**: Required

**Tác dụng**: Lấy profile đầy đủ của user

**Bối cảnh sử dụng**: Profile page, account settings

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

### 2.2 Cập nhật Profile

**PUT** `/api/v1/profile`

**Auth**: Required

**Tác dụng**: Cập nhật thông tin profile

**Bối cảnh sử dụng**: Edit profile page

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

### 2.3 Lấy danh sách địa chỉ

**GET** `/api/v1/profile/addresses`

**Auth**: Required

**Tác dụng**: Lấy tất cả địa chỉ của user

**Bối cảnh sử dụng**: Checkout, address management

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

### 2.4 Thêm địa chỉ mới

**POST** `/api/v1/profile/addresses`

**Auth**: Required

**Tác dụng**: Thêm địa chỉ giao hàng mới

**Bối cảnh sử dụng**: Add address trong checkout hoặc settings

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

### 2.5 Cập nhật địa chỉ

**PUT** `/api/v1/profile/addresses/:id`

**Auth**: Required

**Path Params**: `id` - Address ID (required)

**Tác dụng**: Cập nhật địa chỉ đã có

**Request Body**: Same as create (all optional)

---

### 2.6 Xóa địa chỉ

**DELETE** `/api/v1/profile/addresses/:id`

**Auth**: Required

**Path Params**: `id` - Address ID (required)

**Tác dụng**: Xóa địa chỉ

**Response (200)**:

```json
{
  "success": true,
  "message": "Address deleted successfully"
}
```

---

### 2.7 Đăng ký Shop

**POST** `/api/v1/profile/shop`

**Auth**: Required

**Tác dụng**: Đăng ký trở thành seller

**Bối cảnh sử dụng**: Seller registration flow

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

### 3.1 Lấy danh sách sản phẩm

**GET** `/api/v1/catalog/products`

**Auth**: Not required

**Tác dụng**: Lấy danh sách sản phẩm có phân trang và filter

**Bối cảnh sử dụng**: Product listing, category page, home page

**Query Params (all optional)**:
| Param | Type | Description |
|-------|------|-------------|
| categoryId | string | Filter theo category |
| brandId | string | Filter theo brand |
| status | string | Filter theo status (active/inactive) |
| limit | int | Số items/page (default: 20) |
| offset | int | Offset để phân trang |

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

### 3.2 Lấy chi tiết sản phẩm

**GET** `/api/v1/catalog/products/:id`

**Auth**: Not required

**Path Params**: `id` - Product ID (required)

**Tác dụng**: Lấy thông tin chi tiết 1 sản phẩm

**Bối cảnh sử dụng**: Product detail page

**Response (200)**: Single product object

---

### 3.3 Lấy sản phẩm theo slug

**GET** `/api/v1/catalog/products/slug/:slug`

**Auth**: Not required

**Path Params**: `slug` - Product slug (required)

**Tác dụng**: Lấy sản phẩm bằng SEO-friendly URL

**Bối cảnh sử dụng**: Direct link share, SEO

---

### 3.4 Tạo sản phẩm

**POST** `/api/v1/catalog/products`

**Auth**: Required (Seller)

**Tác dụng**: Tạo sản phẩm mới

**Bối cảnh sử dụng**: Seller dashboard - add product

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

### 3.5 Cập nhật sản phẩm

**PUT** `/api/v1/catalog/products/:id`

**Auth**: Required (Seller)

**Request Body**: Same as create + `status` field

---

### 3.6 Xóa sản phẩm

**DELETE** `/api/v1/catalog/products/:id`

**Auth**: Required (Seller)

**Response (204)**: No content

---

### 3.7 Lấy danh sách Categories

**GET** `/api/v1/catalog/categories`

**Auth**: Not required

**Tác dụng**: Lấy tất cả categories

**Bối cảnh sử dụng**: Navigation menu, filter sidebar

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

### 3.8 Tạo Category

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

### 3.9 Lấy danh sách Brands

**GET** `/api/v1/catalog/brands`

**Auth**: Not required (GET), Required (CUD operations)

---

## 4. Cart Service

### 4.1 Lấy giỏ hàng

**GET** `/api/v1/cart/:userId`

**Auth**: Required

**Path Params**: `userId` - User ID (required)

**Tác dụng**: Lấy giỏ hàng của user

**Bối cảnh sử dụng**: Cart page, mini cart, checkout

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

### 4.2 Thêm vào giỏ hàng

**POST** `/api/v1/cart/:userId/items`

**Auth**: Required

**Path Params**: `userId` - User ID (required)

**Tác dụng**: Thêm sản phẩm vào giỏ hàng

**Bối cảnh sử dụng**: "Add to cart" button

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

### 4.3 Xóa item khỏi giỏ hàng

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

### 4.4 Cập nhật số lượng

**PUT** `/api/v1/cart/:userId/items/:skuId/quantity`

**Auth**: Required

**Request Body**:

```json
{
  "quantity": 3 // required, > 0
}
```

---

### 4.5 Cập nhật selection

**PUT** `/api/v1/cart/:userId/items/:skuId/selection`

**Auth**: Required

**Tác dụng**: Chọn/bỏ chọn item để checkout

**Request Body**:

```json
{
  "selected": true
}
```

---

### 4.6 Xóa toàn bộ giỏ hàng

**DELETE** `/api/v1/cart/:userId`

**Auth**: Required

**Tác dụng**: Clear cart

---

## 5. Order Service

### 5.1 Tạo đơn hàng

**POST** `/api/v1/orders`

**Auth**: Required

**Tác dụng**: Tạo đơn hàng mới từ cart

**Bối cảnh sử dụng**: Checkout - place order

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

### 5.2 Lấy chi tiết đơn hàng

**GET** `/api/v1/orders/:id`

**Auth**: Required

**Path Params**: `id` - Order UUID (required)

**Tác dụng**: Xem chi tiết đơn hàng

**Bối cảnh sử dụng**: Order detail page, order tracking

---

### 5.3 Lấy danh sách đơn hàng

**GET** `/api/v1/orders/user/:userId`

**Auth**: Required

**Path Params**: `userId` - User UUID (required)

**Query Params (optional)**:

- `limit`: int (default 10)
- `offset`: int (default 0)

**Tác dụng**: Xem lịch sử đơn hàng

**Response (200)**:

```json
{
  "orders": [...],
  "limit": 10,
  "offset": 0
}
```

---

### 5.4 Hủy đơn hàng

**POST** `/api/v1/orders/:id/cancel`

**Auth**: Required

**Tác dụng**: Hủy đơn hàng (chỉ khi status là PENDING)

**Response (200)**:

```json
{
  "message": "Order cancelled successfully"
}
```

**Errors**: 404 (not found), 409 (cannot cancel)

---

### 5.5 Đánh dấu đã thanh toán

**POST** `/api/v1/orders/:id/pay`

**Auth**: Required (Internal/Admin)

**Tác dụng**: Cập nhật trạng thái PAID

---

### 5.6 Đánh dấu đang giao

**POST** `/api/v1/orders/:id/ship`

**Auth**: Required (Admin/Seller)

**Tác dụng**: Cập nhật trạng thái SHIPPED

---

### 5.7 Đánh dấu hoàn thành

**POST** `/api/v1/orders/:id/complete`

**Auth**: Required

**Tác dụng**: Cập nhật trạng thái COMPLETED

---

## 6. Payment Service

### 6.1 Tạo thanh toán

**POST** `/api/v1/payments`

**Auth**: Required

**Tác dụng**: Khởi tạo giao dịch thanh toán

**Bối cảnh sử dụng**: Sau khi tạo order, redirect đến payment

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

### 6.2 Lấy thông tin thanh toán

**GET** `/api/v1/payments/:id`

**Auth**: Required

**Path Params**: `id` - Transaction UUID (required)

---

### 6.3 Lấy thanh toán theo order

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

### 6.4 Webhook từ Provider

**POST** `/api/v1/payments/webhook/:provider`

**Auth**: Not required (verified by signature)

**Path Params**: `provider` - Provider name (momo/stripe/zalopay)

**Tác dụng**: Nhận callback từ payment provider

---

## 7. Inventory Service

### 7.1 Lấy thông tin tồn kho

**GET** `/api/v1/inventory/products/:skuId`

**Auth**: Required

**Path Params**: `skuId` - SKU ID (required)

**Tác dụng**: Kiểm tra số lượng tồn kho

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

### 7.2 Cập nhật tồn kho

**PUT** `/api/v1/inventory/products/:skuId`

**Auth**: Required (Admin/Seller)

**Request Body**:

```json
{
  "total_stock": 150
}
```

---

### 7.3 Lấy lịch sử tồn kho

**GET** `/api/v1/inventory/products/:skuId/history`

**Auth**: Required

---

### 7.4 Reserve Stock (Internal)

**POST** `/api/v1/inventory/reserve`

**Tác dụng**: Đặt trước tồn kho khi tạo order

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

**Tác dụng**: Xác nhận reserve sau khi thanh toán

---

### 7.6 Release Stock (Internal)

**POST** `/api/v1/inventory/release`

**Tác dụng**: Giải phóng stock khi hủy order

---

## 8. Logistic Service

### 8.1 Tính phí vận chuyển

**POST** `/api/v1/logistics/calculate-fee`

**Auth**: Required

**Tác dụng**: Tính phí ship dựa trên provider

**Bối cảnh sử dụng**: Checkout - select shipping method

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

### 8.2 Tạo vận đơn

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

### 8.3 Lấy thông tin vận đơn

**GET** `/api/v1/logistics/shipments/:id`

**Auth**: Required

---

### 8.4 Lấy vận đơn theo order

**GET** `/api/v1/logistics/shipments/order/:orderId`

**Auth**: Required

---

### 8.5 Tracking vận đơn

**GET** `/api/v1/logistics/shipments/:id/tracking`

**Auth**: Required

**Tác dụng**: Lấy lịch sử tracking

---

### 8.6 Webhook từ đơn vị vận chuyển

**POST** `/api/v1/logistics/webhook/:provider`

**Auth**: Not required

---

## 9. Media Service

### 9.1 Lấy Presigned URL để upload

**POST** `/api/v1/media/presigned-url`

**Auth**: Required

**Tác dụng**: Lấy URL để upload file trực tiếp lên S3/MinIO

**Bối cảnh sử dụng**: Upload ảnh sản phẩm, avatar

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

### 9.2 Xác nhận upload hoàn tất

**POST** `/api/v1/media/confirm`

**Auth**: Required

**Tác dụng**: Xác nhận file đã upload xong

**Request Body**:

```json
{
  "file_key": "uploads/abc123.jpg" // required
}
```

---

### 9.3 Lấy thông tin media

**GET** `/api/v1/media/:id`

**Auth**: Required

---

### 9.4 Xóa media

**DELETE** `/api/v1/media/:id`

**Auth**: Required

---

## 10. Review Service

### 10.1 Tạo đánh giá

**POST** `/api/v1/reviews`

**Auth**: Required

**Tác dụng**: Đánh giá sản phẩm sau khi mua

**Bối cảnh sử dụng**: Order completed -> write review

**Request Body**:

```json
{
  "userId": "uuid", // required
  "userName": "Nguyen Van A", // required
  "userAvatar": "https://...", // optional
  "productId": "product_uuid", // required
  "orderId": "order_uuid", // required
  "rating": 5, // required: 1-5
  "content": "Sản phẩm rất tốt...", // required, 1-5000 chars
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

### 10.2 Lấy reviews của sản phẩm

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

### 10.3 Lấy rating summary

**GET** `/api/v1/reviews/products/:productId/rating`

**Auth**: Not required

**Tác dụng**: Lấy tổng hợp rating

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

### 10.4 Reply đánh giá (Seller)

**POST** `/api/v1/reviews/:id/reply`

**Auth**: Required (Seller)

**Request Body**:

```json
{
  "content": "Cảm ơn bạn đã ủng hộ..." // required
}
```

---

### 10.5 Lấy reviews của tôi

**GET** `/api/v1/reviews/my`

**Auth**: Required

---

### 10.6 Cập nhật đánh giá

**PUT** `/api/v1/reviews/:id`

**Auth**: Required

---

### 10.7 Xóa đánh giá

**DELETE** `/api/v1/reviews/:id`

**Auth**: Required

---

## 11. Search Service

### 11.1 Tìm kiếm sản phẩm (GET)

**GET** `/api/v1/search/products`

**Auth**: Not required

**Tác dụng**: Full-text search sản phẩm

**Bối cảnh sử dụng**: Search bar, search results page

**Query Params (all optional)**:
| Param | Type | Description |
|-------|------|-------------|
| keyword | string | Từ khóa tìm kiếm |
| categoryId | string | Filter category |
| brandId | string | Filter brand |
| priceMin | float | Giá tối thiểu |
| priceMax | float | Giá tối đa |
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

### 11.2 Gợi ý tìm kiếm

**GET** `/api/v1/search/suggest`

**Auth**: Not required

**Query Params**:

- `keyword`: string (required)

**Tác dụng**: Autocomplete suggestions

---

### 11.3 Sản phẩm theo category

**GET** `/api/v1/search/categories/:categoryId/products`

**Auth**: Not required

---

## 12. Campaign Service

### 12.1 Lấy danh sách campaigns

**GET** `/api/v1/campaigns`

**Auth**: Not required

**Tác dụng**: Lấy các campaigns đang active

**Bối cảnh sử dụng**: Home page banners, flash sales

---

### 12.2 Lấy chi tiết campaign

**GET** `/api/v1/campaigns/:id`

**Auth**: Not required

---

### 12.3 Lấy vouchers public

**GET** `/api/v1/campaigns/vouchers/public`

**Auth**: Not required

**Tác dụng**: Danh sách vouchers có thể claim

---

### 12.4 Claim voucher

**POST** `/api/v1/campaigns/vouchers/claim`

**Auth**: Required

**Tác dụng**: Lưu voucher vào tài khoản

**Request Body**:

```json
{
  "code": "SUMMER2026"
}
```

---

### 12.5 Lấy vouchers của tôi

**GET** `/api/v1/campaigns/vouchers/my`

**Auth**: Required

**Tác dụng**: Danh sách vouchers đã claim

---

### 12.6 Apply voucher

**POST** `/api/v1/campaigns/vouchers/apply`

**Auth**: Required

**Tác dụng**: Áp dụng voucher vào cart để tính discount

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

### 12.7 Tạo Campaign (Admin)

**POST** `/api/v1/campaigns`

**Auth**: Required (Admin)

---

### 12.8 Cập nhật Campaign (Admin)

**PUT** `/api/v1/campaigns/:id`

**Auth**: Required (Admin)

---

### 12.9 Xóa Campaign (Admin)

**DELETE** `/api/v1/campaigns/:id`

**Auth**: Required (Admin)

---

### 12.10 Tạo Voucher cho Campaign (Admin)

**POST** `/api/v1/campaigns/:id/vouchers`

**Auth**: Required (Admin)

---

## 13. Analytics Service

### 13.1 Dashboard Overview

**GET** `/api/v1/analytics/dashboard`

**Auth**: Required (Admin/Seller)

**Tác dụng**: Tổng hợp metrics cho dashboard

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

**Tác dụng**: Track user behavior events

**Bối cảnh sử dụng**: Frontend tracking (page views, clicks, etc.)

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

### 14.1 Lấy danh sách notifications

**GET** `/api/v1/notifications`

**Auth**: Required

**Tác dụng**: Lấy tất cả thông báo của user

**Query Params (optional)**:

- `page`: int
- `limit`: int

---

### 14.2 Lấy số unread

**GET** `/api/v1/notifications/unread-count`

**Auth**: Required

**Response (200)**:

```json
{
  "count": 5
}
```

---

### 14.3 Đánh dấu đã đọc

**PUT** `/api/v1/notifications/:id/read`

**Auth**: Required

---

### 14.4 Đánh dấu tất cả đã đọc

**PUT** `/api/v1/notifications/read-all`

**Auth**: Required

---

### 14.5 Lấy notification preferences

**GET** `/api/v1/notifications/preferences`

**Auth**: Required

---

### 14.6 Cập nhật notification preferences

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

Tất cả errors đều trả về format sau:

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

- `page` hoặc `offset`: Vị trí bắt đầu
- `limit`: Số items (max 100)

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
