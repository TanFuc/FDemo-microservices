-- Flash Sales, Bundle Deals, Gift with Purchase Schema
-- Migration: 002_flash_sales_bundles.sql
-- Enterprise-grade promotional campaigns

-- =====================================================
-- Enums
-- =====================================================

CREATE TYPE flash_sale_status AS ENUM ('DRAFT', 'SCHEDULED', 'ACTIVE', 'ENDED', 'CANCELLED');
CREATE TYPE bundle_deal_status AS ENUM ('DRAFT', 'ACTIVE', 'INACTIVE', 'EXPIRED');
CREATE TYPE bundle_discount_type AS ENUM ('PERCENTAGE', 'FIXED_AMOUNT', 'FIXED_PRICE');
CREATE TYPE gwp_status AS ENUM ('DRAFT', 'ACTIVE', 'INACTIVE', 'EXPIRED');
CREATE TYPE gwp_trigger_type AS ENUM ('MIN_SPEND', 'PRODUCT_BUY', 'CATEGORY_BUY', 'QUANTITY', 'TIERED');
CREATE TYPE gwp_selection_type AS ENUM ('AUTOMATIC', 'USER_CHOICE');

-- =====================================================
-- Flash Sales Table
-- =====================================================
CREATE TABLE flash_sales (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    campaign_id UUID REFERENCES campaigns(id),

    -- Basic info
    name VARCHAR(255) NOT NULL,
    slug VARCHAR(255) NOT NULL UNIQUE,
    description TEXT,
    banner_url VARCHAR(500),
    mobile_banner_url VARCHAR(500),

    -- Timing
    start_time TIMESTAMP WITH TIME ZONE NOT NULL,
    end_time TIMESTAMP WITH TIME ZONE NOT NULL,
    timezone VARCHAR(50) DEFAULT 'Asia/Ho_Chi_Minh',

    -- Status
    status flash_sale_status NOT NULL DEFAULT 'DRAFT',

    -- Limits
    max_products_per_user INTEGER DEFAULT 0,
    max_total_orders INTEGER DEFAULT 0,

    -- Stats
    total_products INTEGER DEFAULT 0,
    total_sold_qty INTEGER DEFAULT 0,
    total_revenue DECIMAL(19, 4) DEFAULT 0,

    -- Visibility
    is_visible BOOLEAN DEFAULT TRUE,
    is_featured BOOLEAN DEFAULT FALSE,
    sort_order INTEGER DEFAULT 0,

    -- Targeting (JSONB for flexibility)
    shop_ids JSONB DEFAULT '[]',
    category_ids JSONB DEFAULT '[]',
    user_segments JSONB DEFAULT '[]',

    -- Metadata
    metadata JSONB DEFAULT '{}',

    -- Audit
    created_by VARCHAR(100),
    updated_by VARCHAR(100),
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),

    CONSTRAINT check_flash_sale_time CHECK (end_time > start_time)
);

-- Flash Sale Products Table
CREATE TABLE flash_sale_products (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    flash_sale_id UUID NOT NULL REFERENCES flash_sales(id) ON DELETE CASCADE,

    -- Product info
    product_id VARCHAR(100) NOT NULL,
    sku_id VARCHAR(100),
    shop_id UUID NOT NULL,

    -- Pricing
    original_price DECIMAL(19, 4) NOT NULL,
    flash_price DECIMAL(19, 4) NOT NULL,
    discount_percent DECIMAL(5, 2),

    -- Inventory
    total_stock INTEGER NOT NULL DEFAULT 0,
    sold_qty INTEGER DEFAULT 0,
    available_qty INTEGER DEFAULT 0,
    max_qty_per_user INTEGER DEFAULT 0,

    -- Limits
    min_qty_per_order INTEGER DEFAULT 1,
    max_qty_per_order INTEGER DEFAULT 0,

    -- Display
    sort_order INTEGER DEFAULT 0,
    is_highlighted BOOLEAN DEFAULT FALSE,

    -- Status
    is_active BOOLEAN DEFAULT TRUE,

    -- Stats
    view_count BIGINT DEFAULT 0,
    cart_add_count BIGINT DEFAULT 0,

    -- Product snapshot
    product_name VARCHAR(500),
    product_image VARCHAR(500),
    product_slug VARCHAR(255),

    -- Timestamps
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),

    CONSTRAINT check_flash_price CHECK (flash_price <= original_price),
    CONSTRAINT check_available_qty CHECK (available_qty >= 0),
    UNIQUE(flash_sale_id, product_id, sku_id)
);

-- Indexes for flash_sales
CREATE INDEX idx_flash_sales_status ON flash_sales(status);
CREATE INDEX idx_flash_sales_time ON flash_sales(start_time, end_time);
CREATE INDEX idx_flash_sales_featured ON flash_sales(is_featured, sort_order) WHERE is_visible = TRUE;
CREATE INDEX idx_flash_sales_active ON flash_sales(status, start_time, end_time) WHERE status = 'ACTIVE';

-- Indexes for flash_sale_products
CREATE INDEX idx_flash_sale_products_sale ON flash_sale_products(flash_sale_id);
CREATE INDEX idx_flash_sale_products_product ON flash_sale_products(product_id);
CREATE INDEX idx_flash_sale_products_shop ON flash_sale_products(shop_id);
CREATE INDEX idx_flash_sale_products_active ON flash_sale_products(flash_sale_id, is_active) WHERE is_active = TRUE;

-- =====================================================
-- Bundle Deals Table
-- =====================================================
CREATE TABLE bundle_deals (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    campaign_id UUID REFERENCES campaigns(id),
    shop_id UUID NOT NULL,

    -- Basic info
    name VARCHAR(255) NOT NULL,
    slug VARCHAR(255) NOT NULL,
    description TEXT,
    image_url VARCHAR(500),

    -- Bundle items (JSONB array)
    items JSONB NOT NULL DEFAULT '[]',
    min_items INTEGER DEFAULT 2,
    max_items INTEGER DEFAULT 0,

    -- Pricing
    discount_type bundle_discount_type NOT NULL,
    discount_value DECIMAL(19, 4) NOT NULL,
    bundle_price DECIMAL(19, 4),
    max_discount DECIMAL(19, 4),

    -- Calculated prices
    original_total DECIMAL(19, 4),
    final_price DECIMAL(19, 4),
    savings_amount DECIMAL(19, 4),
    savings_percent DECIMAL(5, 2),

    -- Limits
    total_stock INTEGER DEFAULT 0,
    sold_count INTEGER DEFAULT 0,
    max_per_user INTEGER DEFAULT 0,

    -- Validity
    valid_from TIMESTAMP WITH TIME ZONE,
    valid_until TIMESTAMP WITH TIME ZONE,

    -- Status
    status bundle_deal_status NOT NULL DEFAULT 'DRAFT',
    is_stackable BOOLEAN DEFAULT FALSE,
    is_visible BOOLEAN DEFAULT TRUE,

    -- Conditions
    min_order_value DECIMAL(19, 4),
    user_segments JSONB DEFAULT '[]',

    -- Stats
    view_count BIGINT DEFAULT 0,
    total_revenue DECIMAL(19, 4) DEFAULT 0,

    -- Metadata
    metadata JSONB DEFAULT '{}',

    -- Audit
    created_by VARCHAR(100),
    updated_by VARCHAR(100),
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),

    UNIQUE(shop_id, slug)
);

-- Indexes for bundle_deals
CREATE INDEX idx_bundle_deals_shop ON bundle_deals(shop_id);
CREATE INDEX idx_bundle_deals_status ON bundle_deals(status);
CREATE INDEX idx_bundle_deals_validity ON bundle_deals(valid_from, valid_until) WHERE status = 'ACTIVE';
CREATE INDEX idx_bundle_deals_active ON bundle_deals(shop_id, status) WHERE status = 'ACTIVE';

-- =====================================================
-- Gift with Purchase Table
-- =====================================================
CREATE TABLE gift_with_purchases (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    campaign_id UUID REFERENCES campaigns(id),
    shop_id UUID, -- NULL = platform-wide

    -- Basic info
    name VARCHAR(255) NOT NULL,
    slug VARCHAR(255) NOT NULL,
    description TEXT,
    banner_url VARCHAR(500),
    promo_badge VARCHAR(50),

    -- Trigger configuration
    trigger_type gwp_trigger_type NOT NULL,
    min_spend_amount DECIMAL(19, 4),
    trigger_products JSONB DEFAULT '[]',
    trigger_categories JSONB DEFAULT '[]',
    trigger_quantity INTEGER,

    -- Tiered thresholds
    tiers JSONB DEFAULT '[]',

    -- Gift selection
    selection_type gwp_selection_type NOT NULL DEFAULT 'AUTOMATIC',
    gift_options JSONB NOT NULL DEFAULT '[]',
    max_gifts_per_order INTEGER DEFAULT 1,

    -- Stock management
    total_gift_stock INTEGER DEFAULT 0,
    claimed_count INTEGER DEFAULT 0,

    -- Limits
    max_claims_per_user INTEGER DEFAULT 0,
    max_total_claims INTEGER DEFAULT 0,

    -- Validity
    valid_from TIMESTAMP WITH TIME ZONE,
    valid_until TIMESTAMP WITH TIME ZONE,

    -- Status
    status gwp_status NOT NULL DEFAULT 'DRAFT',
    is_stackable BOOLEAN DEFAULT TRUE,
    is_visible BOOLEAN DEFAULT TRUE,

    -- Priority
    priority INTEGER DEFAULT 0,

    -- Conditions
    user_segments JSONB DEFAULT '[]',
    excluded_products JSONB DEFAULT '[]',
    new_users_only BOOLEAN DEFAULT FALSE,
    first_order_only BOOLEAN DEFAULT FALSE,

    -- Stats
    view_count BIGINT DEFAULT 0,
    conversion_count BIGINT DEFAULT 0,

    -- Metadata
    metadata JSONB DEFAULT '{}',

    -- Audit
    created_by VARCHAR(100),
    updated_by VARCHAR(100),
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- GWP Claims Table (tracks who claimed what)
CREATE TABLE gwp_claims (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    gwp_id UUID NOT NULL REFERENCES gift_with_purchases(id),
    user_id VARCHAR(100) NOT NULL,
    order_id UUID NOT NULL,
    gift_option_id UUID NOT NULL,
    product_id VARCHAR(100) NOT NULL,
    quantity INTEGER NOT NULL DEFAULT 1,
    tier_index INTEGER,
    trigger_amount DECIMAL(19, 4),
    claimed_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),

    -- Unique per order to prevent duplicates
    UNIQUE(gwp_id, order_id)
);

-- Indexes for gift_with_purchases
CREATE INDEX idx_gwp_shop ON gift_with_purchases(shop_id);
CREATE INDEX idx_gwp_status ON gift_with_purchases(status);
CREATE INDEX idx_gwp_validity ON gift_with_purchases(valid_from, valid_until) WHERE status = 'ACTIVE';
CREATE INDEX idx_gwp_trigger ON gift_with_purchases(trigger_type, status) WHERE status = 'ACTIVE';
CREATE INDEX idx_gwp_priority ON gift_with_purchases(priority DESC) WHERE status = 'ACTIVE';

-- Indexes for gwp_claims
CREATE INDEX idx_gwp_claims_gwp ON gwp_claims(gwp_id);
CREATE INDEX idx_gwp_claims_user ON gwp_claims(user_id);
CREATE INDEX idx_gwp_claims_order ON gwp_claims(order_id);

-- =====================================================
-- Triggers
-- =====================================================

CREATE OR REPLACE FUNCTION update_campaign_updated_at()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER update_flash_sales_updated_at
    BEFORE UPDATE ON flash_sales
    FOR EACH ROW EXECUTE FUNCTION update_campaign_updated_at();

CREATE TRIGGER update_flash_sale_products_updated_at
    BEFORE UPDATE ON flash_sale_products
    FOR EACH ROW EXECUTE FUNCTION update_campaign_updated_at();

CREATE TRIGGER update_bundle_deals_updated_at
    BEFORE UPDATE ON bundle_deals
    FOR EACH ROW EXECUTE FUNCTION update_campaign_updated_at();

CREATE TRIGGER update_gwp_updated_at
    BEFORE UPDATE ON gift_with_purchases
    FOR EACH ROW EXECUTE FUNCTION update_campaign_updated_at();

-- =====================================================
-- Comments
-- =====================================================
COMMENT ON TABLE flash_sales IS 'Flash sale promotional events with time-limited discounts';
COMMENT ON TABLE flash_sale_products IS 'Products participating in flash sales with discounted pricing';
COMMENT ON TABLE bundle_deals IS 'Bundle deals where multiple products are sold together at discount';
COMMENT ON TABLE gift_with_purchases IS 'Gift with purchase promotions triggered by order conditions';
COMMENT ON TABLE gwp_claims IS 'Tracks claims of gifts from GWP promotions';
