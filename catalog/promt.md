# MISSION: BUILD HIGH-PERFORMANCE CATALOG SERVICE (GO + MONGO + REDIS + NATS)

**Role:** Principal Backend Engineer (Golang Expert).
**Goal:** Build a production-ready `catalog-service` that manages Categories, Brands, Products (with Rich Media & Metadata), and publishes events to NATS JetStream.

**Tech Stack (Strict):**
- **Language:** Go 1.22+.
- **Framework:** `github.com/gofiber/fiber/v2` (Performance focus).
- **Database:** MongoDB (`go.mongodb.org/mongo-driver`).
- **Cache:** Redis (`github.com/redis/go-redis/v9`).
- **Broker:** NATS JetStream (`github.com/nats-io/nats.go`).
- **Architecture:** Clean Architecture (`cmd`, `internal/domain`, `internal/usecase`, `internal/repository`, `internal/delivery/http`).

---

## PHASE 1: DOMAIN MODELS (`internal/domain`)

Define strict Structs with BSON/JSON tags.

### 1. `Category`
```go
type AttributeDefinition struct {
    Key        string   `bson:"key" json:"key"`
    Type       string   `bson:"type" json:"type"` // text, number, select
    IsRequired bool     `bson:"isRequired" json:"isRequired"`
    Options    []string `bson:"options,omitempty" json:"options,omitempty"`
}

type Category struct {
    ID                   primitive.ObjectID    `bson:"_id,omitempty" json:"id"`
    Name                 string                `bson:"name" json:"name"`
    Slug                 string                `bson:"slug" json:"slug"`
    ImageURL             string                `bson:"imageUrl" json:"imageUrl"`
    AttributeDefinitions []AttributeDefinition `bson:"attributeDefinitions" json:"attributeDefinitions"`
    ParentID             *primitive.ObjectID   `bson:"parentId,omitempty" json:"parentId"`
}
2. Brand
Go

type Brand struct {
    ID          primitive.ObjectID `bson:"_id,omitempty" json:"id"`
    Name        string             `bson:"name" json:"name"`
    Slug        string             `bson:"slug" json:"slug"`
    LogoURL     string             `bson:"logoUrl" json:"logoUrl"`
    Status      string             `bson:"status" json:"status"` // ACTIVE, INACTIVE
}
3. Product (Updated with Metadata)
Go

type Variation struct {
    SKU         string                 `bson:"sku" json:"sku"`
    Price       float64                `bson:"price" json:"price"`
    Stock       int                    `bson:"stock" json:"stock"` // Virtual display only
    ImageURL    string                 `bson:"imageUrl" json:"imageUrl"`
    Attributes  map[string]interface{} `bson:"attributes" json:"attributes"`
    TierIndex   []int                  `bson:"tierIndex" json:"tierIndex"`
    
    // --- Flexible Config for SKU ---
    // Example: { "flashSaleLimit": 2, "preOrderEta": "2023-12-01" }
    Metadata    map[string]interface{} `bson:"metadata,omitempty" json:"metadata,omitempty"`
}

type Product struct {
    ID          primitive.ObjectID     `bson:"_id,omitempty" json:"id"`
    Name        string                 `bson:"name" json:"name"`
    Slug        string                 `bson:"slug" json:"slug"`
    CategoryID  primitive.ObjectID     `bson:"categoryId" json:"categoryId"`
    BrandID     primitive.ObjectID     `bson:"brandId" json:"brandId"`
    
    // --- Rich Media ---
    Thumbnail   string                 `bson:"thumbnail" json:"thumbnail"`
    Images      []string               `bson:"images" json:"images"`
    VideoURL    string                 `bson:"videoUrl,omitempty" json:"videoUrl,omitempty"`
    Description string                 `bson:"description" json:"description"`

    // --- Attributes ---
    Specs       map[string]interface{} `bson:"specs" json:"specs"`
    Variations  []Variation            `bson:"variations" json:"variations"`
    
    // --- Flexible Config for Product ---
    // Example: { "isFlashSale": true, "campaignId": "SALE1111", "tags": ["hot", "new"] }
    Metadata    map[string]interface{} `bson:"metadata,omitempty" json:"metadata,omitempty"`

    Status      string                 `bson:"status" json:"status"` // DRAFT, PUBLISHED
    CreatedAt   time.Time              `bson:"createdAt" json:"createdAt"`
    UpdatedAt   time.Time              `bson:"updatedAt" json:"updatedAt"`
}
PHASE 2: REPOSITORIES & INFRASTRUCTURE
1. MongoDB Repository
Indices:

Product: Unique(Slug), Text(Name), Index(CategoryID), Index(BrandID).

Category: Unique(Slug).

Metadata Indexing: Consider creating a Wildcard Index on metadata.$** if you plan to filter by metadata fields often (e.g. Find({ "metadata.isFlashSale": true })).

2. Redis Repository (Cache)
Cache Category Definitions.

Cache Product: Consider caching ProductDetail by Slug (TTL 5 mins) because Metadata flags might be read frequently by frontend to show badges (e.g. "Flash Sale" badge).

3. NATS Publisher
Subjects: catalog.product.created, catalog.product.updated.

Payload: MUST include metadata so downstream services (like Search Service or Campaign Service) can use this info.

PHASE 3: USECASE LOGIC
CreateProduct(ctx, dto)
Validation:

Validate Category specs.

Metadata Validation: (Optional) You can add logic to sanitize metadata keys if needed, but usually we allow dynamic content here.

Logic:

Auto-generate Slug.

Set default Status.

Persistence: Save to Mongo.

Publish: Send event to NATS including the new Metadata.

UpdateProductMetadata(ctx, id, metadata)
Specific method to PATCH only the metadata (useful for internal Admin tools / Campaign Service to tag products quickly without re-sending the whole payload).

Publish catalog.product.updated.

PHASE 4: AUTO-VERIFICATION LOOP
Instructions for AI:

Init: Setup Project.

Generate: Create Code.

TEST (Critical): Create internal/usecase/product_test.go.

Test Metadata: Create a product with Metadata: {"isFlashSale": true, "limit": 5}. Retrieve it and verify the map contains these keys.

Test Variation Metadata: Create a SKU with Metadata: {"preOrder": true}. Verify persistence.

Build: Run go build.

IMPORTANT RULES
Map Initialization: Ensure Metadata maps are initialized to {} (empty map) if nil, to avoid Null Pointer Exceptions when accessing keys.

BSON Handling: Use bson:"metadata,omitempty" to save space if metadata is empty.