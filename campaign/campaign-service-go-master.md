# MISSION: BUILD DYNAMIC CAMPAIGN & PROMOTION SERVICE (GO + POSTGRES + REDIS)

**Role:** Principal Backend Architect.
**Goal:** Build a `campaign-service` that manages Vouchers/Coupons and calculates discounts using a flexible **Rule Engine**. It must handle high-concurrency voucher claiming (Flash Coupons).

**Tech Stack:**
- **Language:** Go 1.22+.
- **Database:** PostgreSQL (Transactions & JSONB for Rules).
- **Cache:** Redis (`go-redis/v9`) + Lua Scripts (For atomic claiming).
- **Architecture:** Clean Architecture.

---

## PHASE 1: DATABASE SCHEMA (THE RULE ENGINE)

We need a flexible schema to store complex conditions without hardcoding.

### 1. `Campaign`
- `ID`, `Name`, `StartTime`, `EndTime`, `Status`.

### 2. `Voucher` (The Coupon)
- **Table:** `vouchers`
- **Fields:**
    - `Code` (String, Unique) - e.g., "SALE1111".
    - `CampaignID` (UUID).
    - `TotalCount` (Int) - Max issuance.
    - `UsedCount` (Int).
    - `Type` (Enum: PERCENTAGE, FIXED_AMOUNT).
    - `Value` (Decimal) - e.g., 10 (percent) or 50000 (VND).
    - **`Conditions`** (JSONB) - The Rule Set.
      ```json
      {
        "min_order_value": 500000,
        "max_discount": 50000,
        "allowed_categories": ["electronics"],
        "excluded_products": ["sku_iphone_15"]
      }
      ```
    - `Status` (ACTIVE, INACTIVE).

### 3. `UserVoucher` (The Wallet)
- **Table:** `user_vouchers`
- **Fields:** `UserID`, `VoucherID`, `IsUsed` (Bool), `UsedAt` (Time).
- **Constraint:** Unique(UserID, VoucherID) -> User can claim only once (unless configured otherwise).

## PHASE 2: REDIS LUA SCRIPT (ATOMIC CLAIM)

Just like Inventory, Voucher Claiming needs to be atomic to prevent overselling.

- **Key:** `voucher:{code}:stock` (Int).
- **Key:** `voucher:{code}:users` (Set of UserIDs - to check double claim quickly).
- **Logic (`claim.lua`):**
    1. Check if UserID is in Set -> Return Error "Already Claimed".
    2. Check if Stock > 0.
    3. Decrement Stock.
    4. Add UserID to Set.
    5. Return Success.

## PHASE 3: USECASE LOGIC

### 1. `ClaimVoucher(ctx, userId, code)`
- **Step 1 (Redis):** Run Lua Script.
    - If Success: Proceed.
    - If Fail: Return Error.
- **Step 2 (Postgres Async):**
    - Insert into `UserVoucher`.
    - Update `Voucher` (Increment UsedCount - purely for stats).
    - *Note:* Redis is the Source of Truth for stock here.

### 2. `CalculateCart(ctx, cartItems, voucherCode)` (The Brain)
- **Input:** List of Items (Price, Category, SKU) + Applied Voucher Code.
- **Logic:**
    - Fetch Voucher Rules from DB/Redis.
    - **Rule Engine Validation:**
        - Is `Sum(Item.Price)` >= `Conditions.min_order_value`?
        - Do items belong to `Conditions.allowed_categories`?
    - **Calculation:**
        - If Percentage: `Discount = Total * Value / 100`.
        - Apply `Conditions.max_discount` cap.
    - **Return:** `FinalPrice`, `DiscountAmount`.

## PHASE 4: AUTO-VERIFICATION LOOP

**Instructions for AI:**
1.  **Init:** Setup Go project.
2.  **Generate:** Clean Architecture.
3.  **TEST (Logic & Concurrency):**
    - Create `internal/usecase/campaign_test.go`.
    - **Test Rule Engine:**
        - Create Voucher "10% OFF, Min Spend 100k".
        - Input Cart: 90k -> Expect Error "Not eligible".
        - Input Cart: 200k -> Expect Discount 20k.
    - **Test Concurrency:**
        - Voucher Stock = 50.
        - Spawn 100 Goroutines to Claim.
        - Expect exactly 50 Success.
4.  **Build:** Run `go build`.

---

## IMPORTANT RULES
- **Floating Point:** Use `shopspring/decimal` strictly.
- **Validation:** The `CalculateCart` method simulates the discount but DOES NOT save it. The actual usage happens in Order Service (when placing order).