-- Specialized Analytics Event Tables for ClickHouse
-- Migration: 002_specialized_events.sql
-- High-performance OLAP tables for e-commerce analytics

-- =====================================================
-- Product Events Table
-- =====================================================
CREATE TABLE IF NOT EXISTS product_events (
    event_id UUID,
    user_id String,
    session_id String,
    product_id String,
    sku_id Nullable(String),
    shop_id String,
    category_id String,
    brand_id Nullable(String),

    -- Event type
    event_type LowCardinality(String), -- view, add_to_cart, remove_from_cart, purchase, wishlist_add, wishlist_remove

    -- Product context
    price Decimal64(4),
    quantity UInt32,
    currency LowCardinality(String),

    -- Source/attribution
    source LowCardinality(String),
    ref_product_id Nullable(String),
    search_query Nullable(String),
    position Nullable(UInt16),

    -- User context
    device_type LowCardinality(String),
    platform LowCardinality(String),

    -- Request info
    ip_address IPv4,
    user_agent String,
    country LowCardinality(String),
    city Nullable(String),

    -- Timestamp
    created_at DateTime64(3) DEFAULT now64(3)
)
ENGINE = MergeTree()
PARTITION BY toYYYYMM(created_at)
ORDER BY (product_id, created_at, event_id)
TTL created_at + INTERVAL 2 YEAR;

-- Materialized view for product view counts (daily)
CREATE MATERIALIZED VIEW IF NOT EXISTS mv_product_views_daily
ENGINE = SummingMergeTree()
PARTITION BY toYYYYMM(date)
ORDER BY (product_id, date)
AS SELECT
    product_id,
    toDate(created_at) as date,
    countIf(event_type = 'view') as views,
    countIf(event_type = 'add_to_cart') as cart_adds,
    countIf(event_type = 'purchase') as purchases,
    uniqExact(user_id) as unique_users
FROM product_events
GROUP BY product_id, date;

-- =====================================================
-- Search Query Events Table
-- =====================================================
CREATE TABLE IF NOT EXISTS search_queries (
    event_id UUID,
    user_id String,
    session_id String,

    -- Query info
    query String,
    normalized_query String,
    query_length UInt16,
    query_type LowCardinality(String),

    -- Results
    total_results UInt32,
    results_page UInt8,
    results_limit UInt8,
    has_results UInt8, -- Boolean as UInt8

    -- Filters
    category_filter Nullable(String),
    brand_filter Nullable(String),
    price_min_filter Nullable(Decimal64(4)),
    price_max_filter Nullable(Decimal64(4)),
    sort_by Nullable(String),

    -- Interaction
    clicked_product_id Nullable(String),
    click_position Nullable(UInt16),
    time_to_click_ms Nullable(UInt32),
    clicked_count UInt8,

    -- Conversion
    added_to_cart UInt8,
    purchased UInt8,

    -- Performance
    response_time_ms UInt32,

    -- User context
    device_type LowCardinality(String),
    platform LowCardinality(String),
    ip_address IPv4,
    country LowCardinality(String),

    -- Timestamp
    created_at DateTime64(3) DEFAULT now64(3)
)
ENGINE = MergeTree()
PARTITION BY toYYYYMM(created_at)
ORDER BY (normalized_query, created_at, event_id)
TTL created_at + INTERVAL 1 YEAR;

-- Materialized view for popular search queries
CREATE MATERIALIZED VIEW IF NOT EXISTS mv_popular_searches
ENGINE = SummingMergeTree()
PARTITION BY toYYYYMM(date)
ORDER BY (normalized_query, date)
AS SELECT
    normalized_query,
    toDate(created_at) as date,
    count() as search_count,
    countIf(has_results = 1) as with_results,
    countIf(clicked_count > 0) as with_clicks,
    countIf(added_to_cart = 1) as with_cart_adds,
    countIf(purchased = 1) as with_purchases,
    avg(response_time_ms) as avg_response_time
FROM search_queries
GROUP BY normalized_query, date;

-- =====================================================
-- Order Events Table
-- =====================================================
CREATE TABLE IF NOT EXISTS order_events (
    event_id UUID,
    order_id UUID,
    user_id String,
    session_id String,

    -- Event type
    event_type LowCardinality(String), -- created, paid, shipped, delivered, cancelled, refunded

    -- Order details
    total_amount Decimal64(4),
    subtotal_amount Decimal64(4),
    shipping_amount Decimal64(4),
    discount_amount Decimal64(4),
    tax_amount Decimal64(4),
    currency LowCardinality(String),

    -- Item counts
    item_count UInt16,
    unique_items UInt16,

    -- Payment
    payment_method LowCardinality(String),
    payment_status LowCardinality(String),

    -- Shipping
    shipping_method Nullable(String),
    shipping_provider Nullable(String),

    -- Promotions
    voucher_code Nullable(String),
    voucher_discount Nullable(Decimal64(4)),
    campaign_id Nullable(String),

    -- Attribution
    source LowCardinality(String),
    utm_source Nullable(String),
    utm_medium Nullable(String),
    utm_campaign Nullable(String),
    referrer_url Nullable(String),

    -- User behavior
    cart_duration_min Nullable(UInt32),
    checkout_steps Nullable(UInt8),
    is_first_order UInt8,
    is_returning UInt8,

    -- User context
    device_type LowCardinality(String),
    platform LowCardinality(String),
    ip_address IPv4,
    country LowCardinality(String),
    city Nullable(String),

    -- Timestamp
    created_at DateTime64(3) DEFAULT now64(3)
)
ENGINE = MergeTree()
PARTITION BY toYYYYMM(created_at)
ORDER BY (order_id, created_at, event_id)
TTL created_at + INTERVAL 3 YEAR;

-- Materialized view for daily order stats
CREATE MATERIALIZED VIEW IF NOT EXISTS mv_order_stats_daily
ENGINE = SummingMergeTree()
PARTITION BY toYYYYMM(date)
ORDER BY (date)
AS SELECT
    toDate(created_at) as date,
    countIf(event_type = 'created') as orders_created,
    countIf(event_type = 'paid') as orders_paid,
    countIf(event_type = 'shipped') as orders_shipped,
    countIf(event_type = 'delivered') as orders_delivered,
    countIf(event_type = 'cancelled') as orders_cancelled,
    countIf(event_type = 'refunded') as orders_refunded,
    sumIf(total_amount, event_type = 'created') as gmv,
    avgIf(total_amount, event_type = 'created') as aov,
    uniqExactIf(user_id, event_type = 'created') as unique_buyers,
    countIf(is_first_order = 1 AND event_type = 'created') as new_customers
FROM order_events
GROUP BY date;

-- =====================================================
-- Revenue Events Table (Daily aggregates per shop)
-- =====================================================
CREATE TABLE IF NOT EXISTS revenue_events (
    event_id UUID,
    shop_id String,
    date Date,

    -- Revenue metrics
    gross_revenue Decimal64(4),
    net_revenue Decimal64(4),
    refund_amount Decimal64(4),
    discount_amount Decimal64(4),
    shipping_revenue Decimal64(4),
    commission_amount Decimal64(4),
    currency LowCardinality(String),

    -- Order counts
    total_orders UInt32,
    completed_orders UInt32,
    cancelled_orders UInt32,
    refunded_orders UInt32,

    -- Item metrics
    total_items_sold UInt32,
    unique_products UInt32,
    unique_customers UInt32,
    new_customers UInt32,
    returning_customers UInt32,

    -- Average metrics
    avg_order_value Decimal64(4),
    avg_items_per_order Decimal64(4),

    -- Timestamps
    created_at DateTime64(3) DEFAULT now64(3),
    updated_at DateTime64(3) DEFAULT now64(3)
)
ENGINE = ReplacingMergeTree(updated_at)
PARTITION BY toYYYYMM(date)
ORDER BY (shop_id, date, event_id);

-- =====================================================
-- Page View Events Table
-- =====================================================
CREATE TABLE IF NOT EXISTS page_views (
    event_id UUID,
    user_id String,
    session_id String,

    -- Page info
    page_type LowCardinality(String),
    page_url String,
    page_title Nullable(String),

    -- Entity context
    product_id Nullable(String),
    category_id Nullable(String),
    shop_id Nullable(String),
    search_query Nullable(String),

    -- Navigation
    referrer_url Nullable(String),
    entry_page UInt8,
    exit_page UInt8,
    page_depth UInt16,

    -- Engagement
    time_on_page_ms Nullable(UInt32),
    scroll_depth Nullable(UInt8),

    -- User context
    device_type LowCardinality(String),
    platform LowCardinality(String),
    browser Nullable(String),
    ip_address IPv4,
    country LowCardinality(String),
    city Nullable(String),

    -- Timestamp
    created_at DateTime64(3) DEFAULT now64(3)
)
ENGINE = MergeTree()
PARTITION BY toYYYYMM(created_at)
ORDER BY (page_type, created_at, event_id)
TTL created_at + INTERVAL 6 MONTH;

-- =====================================================
-- Session Events Table
-- =====================================================
CREATE TABLE IF NOT EXISTS sessions (
    session_id String,
    user_id String,

    -- Session info
    session_start DateTime64(3),
    session_end Nullable(DateTime64(3)),
    duration_seconds Nullable(UInt32),

    -- Page metrics
    page_views UInt16,
    unique_pages UInt16,
    entry_page String,
    exit_page Nullable(String),

    -- Product interactions
    products_viewed UInt16,
    products_carted UInt16,
    searches_performed UInt16,

    -- Conversion
    converted UInt8,
    conversion_value Nullable(Decimal64(4)),
    order_id Nullable(String),

    -- Attribution
    source LowCardinality(String),
    medium Nullable(String),
    campaign Nullable(String),
    landing_url String,

    -- User context
    is_new_user UInt8,
    user_type LowCardinality(String),
    device_type LowCardinality(String),
    platform LowCardinality(String),
    browser Nullable(String),
    country LowCardinality(String),
    city Nullable(String),

    -- Timestamps
    created_at DateTime64(3) DEFAULT now64(3),
    updated_at DateTime64(3) DEFAULT now64(3)
)
ENGINE = ReplacingMergeTree(updated_at)
PARTITION BY toYYYYMM(session_start)
ORDER BY (session_id);

-- =====================================================
-- Funnel Analysis View
-- =====================================================
CREATE VIEW IF NOT EXISTS v_conversion_funnel AS
SELECT
    toDate(created_at) as date,
    countDistinct(session_id) as sessions,
    countDistinctIf(session_id, products_viewed > 0) as viewed_product,
    countDistinctIf(session_id, products_carted > 0) as added_to_cart,
    countDistinctIf(session_id, converted = 1) as purchased
FROM sessions
WHERE session_start >= now() - INTERVAL 30 DAY
GROUP BY date
ORDER BY date;

-- =====================================================
-- Comments
-- =====================================================
-- product_events: Tracks all product interactions (views, cart actions, purchases)
-- search_queries: Tracks search queries and their outcomes
-- order_events: Tracks order lifecycle events
-- revenue_events: Daily revenue aggregates per shop
-- page_views: Tracks page views with engagement metrics
-- sessions: Tracks user sessions with attribution
