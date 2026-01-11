# MISSION: BUILD REVIEW & RATING SERVICE (GO + MONGODB + REDIS)

**Role:** Senior Backend Engineer (Community & Social Systems Expert).
**Goal:** Build a `tafu-review` service that allows verified buyers to rate products and post reviews. It must efficiently handle aggregation (calculating 4.5 stars from 10k reviews) without slowing down the read API.

**Tech Stack:**
- **Language:** Go 1.22+.
- **Framework:** `github.com/gofiber/fiber/v2`.
- **Database:** MongoDB (`go.mongodb.org/mongo-driver`) - Best for unstructured text & nested replies.
- **Cache:** Redis (`github.com/redis/go-redis/v9`).
- **Communication:** gRPC (to call Order Service for verification).

---

## PHASE 1: MONGODB SCHEMA DESIGN

We use two collections: one for raw data, one for pre-calculated stats (Materialized View Pattern).

### 1. `reviews` Collection (The Raw Data)
- **Document Structure:**
    ```json
    {
      "_id": "uuid",
      "userId": "uuid",
      "userName": "Phuc Nguyen",
      "userAvatar": "url_from_profile",
      "productId": "uuid",   // Indexed
      "orderId": "uuid",     // Unique Index (Composite with productId to prevent duplicate reviews)
      "rating": 5,           // Int: 1-5
      "content": "Hàng xịn, giao nhanh!",
      "images": ["url1", "url2"], // URLs from Media Service
      "isPurchased": true,
      "reply": {             // Shop's reply
         "content": "Cảm ơn bạn ạ!",
         "repliedAt": "timestamp"
      },
      "status": "VISIBLE",   // VISIBLE, HIDDEN (Admin moderation)
      "createdAt": "timestamp" // Indexed for sorting
    }
    ```

### 2. `product_ratings` Collection (The Cache/View)
- Updated asynchronously whenever a new review is added.
- **Document Structure:**
    ```json
    {
      "_id": "productId",
      "averageRating": 4.8,
      "totalReviews": 150,
      "starCounts": {
         "1": 2, "2": 0, "3": 5, "4": 20, "5": 123
      },
      "updatedAt": "timestamp"
    }
    ```

## PHASE 2: USECASE LOGIC

### 1. `CreateReview(ctx, dto)`
- **Step 1: Verification (Crucial)**
    - Call **Order Service** (gRPC): `GetOrderDetail(orderId, userId)`.
    - Logic: Ensure Order exists, Status is `COMPLETED` (or DELIVERED), and `productId` is in the item list.
    - *Fail fast if not eligible.*
- **Step 2: Persistence**
    - Insert into `reviews`.
- **Step 3: Aggregation (Async/Background)**
    - Trigger a recalculation for `product_ratings`.
    - Formula: `NewAvg = ((OldAvg * OldTotal) + NewRating) / (OldTotal + 1)`.
    - Update `product_ratings` collection.
    - **Redis Invalidation:** Delete key `rating:{productId}`.

### 2. `GetProductReviews(ctx, productId, page, limit)`
- **Query:** MongoDB `reviews`.
- **Filter:** `productId == ?` AND `status == 'VISIBLE'`.
- **Sort:** `createdAt` DESC (Newest first) or `rating` DESC.

### 3. `GetRatingSummary(ctx, productId)` (High Traffic API)
- **Step 1:** Check Redis `rating:{productId}`.
- **Step 2:** If Miss -> Get from `product_ratings` collection -> Set Redis (TTL 30 mins) -> Return.
- **Payload:** `{ average: 4.8, total: 150, breakdown: {5: 123, 4: 20...} }`.

### 4. `ReplyReview(ctx, reviewId, replyContent)` (For Seller)
- **Logic:** `db.reviews.updateOne({_id: reviewId}, {$set: {reply: ...}})`

## PHASE 3: AUTO-VERIFICATION LOOP

**Instructions for AI:**
1.  **Init:** `go mod init tafu-review`. Setup Fiber, Mongo, Redis.
2.  **Generate:**
    - `internal/core`: Domain models.
    - `internal/adapter`: Mongo Repositories.
    - `internal/handler`: HTTP Handlers.
3.  **TEST (Logic Simulation):**
    - Create `tests/rating_logic_test.go`.
    - **Scenario:**
        - Insert 3 reviews: 5-star, 5-star, 2-star.
        - Run Aggregation Logic.
        - **Expect:** Total=3, Average=4.0.
        - Check Redis key is set after `GetRatingSummary`.
4.  **Build:** Run `go build -o server cmd/server/main.go`.

---

## IMPORTANT RULES
- **Sanitization:** Strip HTML tags from review content to prevent XSS.
- **Media Validation:** Ensure images array contains valid URLs (starts with http/https).
- **Error Handling:** If Order Service is down during verification, fail the request (Safety first, don't allow fake reviews).