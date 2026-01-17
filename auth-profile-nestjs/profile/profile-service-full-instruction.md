# Role
You are a Senior Backend Architect and Developer expert in NestJS, MongoDB (Mongoose), and Microservices patterns.

# Context & Goal
We are building the **Profile Service** for a high-scale E-commerce system.
- **Dependency**: This service relies on the **Identity Service** for authentication. It receives the `userId` (UUID string) from the JWT token (via request headers or guard).
- **Core Function**: Manage User Personal Info, Address Books (Shipping/Billing), and Merchant/Shop Profiles (if the user is a Seller).
- **Database**: MongoDB (using `@nestjs/mongoose`).
- **Safety Requirement**: Although using NoSQL, we must ensure data consistency, especially for the "Default Address" logic.

# Strict Technical Guidelines
1.  **Type Safety**: STRICTLY NO `any` types. Use strict Interfaces and DTOs.
2.  **Validation**: Use `class-validator` and `class-transformer` for all DTOs.
3.  **Database**: Use Mongoose Schemas with strict typing.
4.  **Error Handling**: Throw `NotFoundException`, `ConflictException` (for duplicate shop names), etc.
5.  **Response Format**: All endpoints must return a standard response structure.

---

# PART 1: Mongoose Schemas & Interfaces
**Action:** Create `src/modules/profile/schemas` and define these 2 collections:

## 1. Address Schema (`address.schema.ts`)
This collection stores shipping addresses. A user can have multiple addresses.
- **Fields**:
    - `userId`: String (Indexed). **Required**. (Mapped from Identity Service UUID).
    - `contactName`: String. Required.
    - `phone`: String. Required.
    - `provinceCode`: String. Required.
    - `districtCode`: String. Required.
    - `wardCode`: String. Required.
    - `streetLine`: String. Required (e.g., "123 Le Loi").
    - `fullAddress`: String. (Computed or stored for fast display).
    - `isDefault`: Boolean. Default `false`.
    - `type`: Enum ['HOME', 'OFFICE']. Default 'HOME'.
- **Indices**: Create a compound index on `{ userId: 1 }`.

## 2. Profile Schema (`profile.schema.ts`)
This collection stores the main user info and optional Shop info.
- **Fields**:
    - `userId`: String (Unique Index). **Required**.
    - `displayName`: String. Required.
    - `email`: String. (Read-only, synced from Identity).
    - `avatarUrl`: String. Optional.
    - `bio`: String. Optional.
    - `shopConfig`: Nested Object (Optional - only exists if user is a SELLER).
        - `shopName`: String (Unique Sparse Index).
        - `description`: String.
        - `logoUrl`: String.
        - `pickupAddressId`: ObjectId (Ref to Address).
- **Timestamps**: Enable `{ timestamps: true }`.

---

# PART 2: Data Transfer Objects (DTOs)
**Action:** Create `src/modules/profile/dto`.

1.  `create-profile.dto.ts`: `displayName`, `avatarUrl`, `bio`.
2.  `update-profile.dto.ts`: Partial of create dto.
3.  `address.dto.ts`:
    - All address fields with `@IsString`, `@IsNotEmpty`.
    - `phone`: Validate phone format.
    - `isDefault`: `@IsBoolean`, `@IsOptional`.
4.  `register-shop.dto.ts`: `shopName`, `description`. (Used when a user wants to become a seller).

---

# PART 3: Profile Service (Business Logic)
**Action:** Create `src/modules/profile/profile.service.ts`.

## 1. Method: `getOrCreateProfile(userId: string)`
- Try to find Profile by `userId`.
- **Lazy Creation Logic**: If not found, create a new Profile document with default `displayName` (e.g., "User-" + last 4 chars of ID) and return it. This ensures the profile always exists when requested.

## 2. Method: `addAddress(userId: string, dto: AddressDto)`
- **Atomic Logic for Default Address**:
    - If `dto.isDefault` is `true`:
        - FIRST, run `updateMany({ userId }, { $set: { isDefault: false } })` to unset previous default.
        - THEN, create the new address with `isDefault: true`.
    - If `dto.isDefault` is `false`:
        - Check if this is the **first** address for this user. If yes, force `isDefault: true`.
- Save and return the Address.

## 3. Method: `setDefaultAddress(userId: string, addressId: string)`
- Use a MongoDB Transaction (Session) or ordered operations:
    1. `updateMany({ userId }, { $set: { isDefault: false } })`.
    2. `updateOne({ _id: addressId, userId }, { $set: { isDefault: true } })`.
- Throw `NotFoundException` if the address doesn't belong to the user.

## 4. Method: `registerShop(userId: string, dto: RegisterShopDto)`
- Check if `shopName` is already taken (global check). If yes, throw `ConflictException`.
- Update the Profile document: `$set: { shopConfig: dto }`.

---

# PART 4: Controller
**Action:** Create `src/modules/profile/profile.controller.ts`.

- **Endpoints**:
    - `GET /profiles/me`: Get my profile (call `getOrCreateProfile`).
    - `PATCH /profiles/me`: Update profile info.
    - `POST /profiles/me/shop`: Register/Update Shop info.
    - `GET /profiles/me/addresses`: Get list of addresses.
    - `POST /profiles/me/addresses`: Add new address.
    - `PATCH /profiles/me/addresses/:id/set-default`: Set an address as default.
    - `DELETE /profiles/me/addresses/:id`: Delete address.

# Instructions for Execution
1.  **Analyze**: Read the requirements carefully.
2.  **Scaffold**: Generate the Mongoose Schemas first (`src/modules/profile/schemas`).
3.  **DTOs**: Generate DTOs with validation decorators.
4.  **Service**: Implement the Service with the **Atomic Default Address Logic** exactly as described.
5.  **Controller**: Implement the Controller.
6.  **Module**: Ensure `ProfileModule` imports `MongooseModule.forFeature([...])`.

Start step-by-step.