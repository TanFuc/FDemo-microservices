-- ============================================================================
-- TAFU-ANALYTIC Analytics Service Database Migration
-- Version: 001
-- Date: 2026-01-07
-- Database: ClickHouse
-- Purpose: High-throughput event ingestion and real-time analytics
-- ============================================================================

-- ===================
-- Database
-- ===================
CREATE DATABASE IF NOT EXISTS analytics;

-- ===================
-- Table: user_events (Raw Events)
-- ===================
CREATE TABLE IF NOT EXISTS analytics.user_events (
    event_id UUID DEFAULT generateUUIDv4(),
    
    -- User identification
    user_id String,
    session_id String,
    guest_id String,  -- For non-logged-in users
    
    -- Event details
    event_type LowCardinality(String),  -- view_item, add_to_cart, checkout_start, purchase
    event_category LowCardinality(String),  -- product, cart, checkout, user
    
    -- Context
    metadata String,  -- JSON string for flexible data
    url String,
    referrer String,
    page_title String,
    
    -- Device info
    ip_address String,
    user_agent String,
    device_type LowCardinality(String),  -- mobile, desktop, tablet
    browser LowCardinality(String),
    os LowCardinality(String),
    
    -- Geo
    country_code LowCardinality(String),
    city String,
    
    -- Timestamps
    created_at DateTime DEFAULT now(),
    event_date Date DEFAULT toDate(created_at)
) ENGINE = MergeTree()
PARTITION BY toYYYYMM(created_at)
ORDER BY (event_type, created_at, user_id)
TTL created_at + INTERVAL 365 DAY
SETTINGS index_granularity = 8192;

-- ===================
-- Table: page_views (Specific for Page Analytics)
-- ===================
CREATE TABLE IF NOT EXISTS analytics.page_views (
    view_id UUID DEFAULT generateUUIDv4(),
    user_id String,
    session_id String,
    
    page_url String,
    page_path String,
    page_title String,
    referrer String,
    
    -- Engagement
    time_on_page UInt32,  -- seconds
    scroll_depth UInt8,   -- percentage 0-100
    
    device_type LowCardinality(String),
    country_code LowCardinality(String),
    
    created_at DateTime DEFAULT now()
) ENGINE = MergeTree()
PARTITION BY toYYYYMM(created_at)
ORDER BY (page_path, created_at)
TTL created_at + INTERVAL 180 DAY;

-- ===================
-- Table: product_events (Product-specific Analytics)
-- ===================
CREATE TABLE IF NOT EXISTS analytics.product_events (
    event_id UUID DEFAULT generateUUIDv4(),
    user_id String,
    session_id String,
    
    product_id String,
    sku_id String,
    category_id String,
    brand_id String,
    
    event_type LowCardinality(String),  -- view, add_to_cart, add_to_wishlist, purchase
    
    -- Product details at event time
    product_name String,
    price Decimal64(4),
    quantity UInt32 DEFAULT 1,
    
    -- Source
    source LowCardinality(String),  -- search, recommendation, category, direct
    search_query String,
    
    created_at DateTime DEFAULT now()
) ENGINE = MergeTree()
PARTITION BY toYYYYMM(created_at)
ORDER BY (product_id, event_type, created_at)
TTL created_at + INTERVAL 365 DAY;

-- ===================
-- Materialized View: Hourly Product Views
-- ===================
CREATE MATERIALIZED VIEW IF NOT EXISTS analytics.mv_product_views_hourly
ENGINE = SummingMergeTree()
PARTITION BY toYYYYMM(hour)
ORDER BY (product_id, hour)
AS SELECT
    product_id,
    toStartOfHour(created_at) AS hour,
    count() AS view_count,
    uniqExact(user_id) AS unique_viewers
FROM analytics.product_events
WHERE event_type = 'view'
GROUP BY product_id, hour;

-- ===================
-- Materialized View: Daily Product Stats
-- ===================
CREATE MATERIALIZED VIEW IF NOT EXISTS analytics.mv_product_stats_daily
ENGINE = SummingMergeTree()
PARTITION BY toYYYYMM(date)
ORDER BY (product_id, date)
AS SELECT
    product_id,
    toDate(created_at) AS date,
    countIf(event_type = 'view') AS views,
    countIf(event_type = 'add_to_cart') AS add_to_carts,
    countIf(event_type = 'purchase') AS purchases,
    uniqExactIf(user_id, event_type = 'view') AS unique_viewers,
    sumIf(price * quantity, event_type = 'purchase') AS revenue
FROM analytics.product_events
GROUP BY product_id, date;

-- ===================
-- Materialized View: Hourly Event Counts
-- ===================
CREATE MATERIALIZED VIEW IF NOT EXISTS analytics.mv_events_hourly
ENGINE = SummingMergeTree()
PARTITION BY toYYYYMM(hour)
ORDER BY (event_type, hour)
AS SELECT
    event_type,
    toStartOfHour(created_at) AS hour,
    count() AS event_count,
    uniqExact(user_id) AS unique_users,
    uniqExact(session_id) AS unique_sessions
FROM analytics.user_events
GROUP BY event_type, hour;

-- ===================
-- Materialized View: Funnel Analysis (Daily)
-- ===================
CREATE MATERIALIZED VIEW IF NOT EXISTS analytics.mv_funnel_daily
ENGINE = SummingMergeTree()
PARTITION BY toYYYYMM(date)
ORDER BY (date)
AS SELECT
    toDate(created_at) AS date,
    uniqExactIf(session_id, event_type = 'view_item') AS viewed_sessions,
    uniqExactIf(session_id, event_type = 'add_to_cart') AS cart_sessions,
    uniqExactIf(session_id, event_type = 'checkout_start') AS checkout_sessions,
    uniqExactIf(session_id, event_type = 'purchase') AS purchase_sessions
FROM analytics.user_events
GROUP BY date;

-- ===================
-- Table: search_queries (Search Analytics)
-- ===================
CREATE TABLE IF NOT EXISTS analytics.search_queries (
    query_id UUID DEFAULT generateUUIDv4(),
    user_id String,
    session_id String,
    
    query String,
    query_normalized String,  -- lowercase, trimmed
    
    results_count UInt32,
    clicked_position UInt8,  -- which result was clicked (0 = no click)
    clicked_product_id String,
    
    filters_used String,  -- JSON of applied filters
    
    created_at DateTime DEFAULT now()
) ENGINE = MergeTree()
PARTITION BY toYYYYMM(created_at)
ORDER BY (query_normalized, created_at)
TTL created_at + INTERVAL 90 DAY;

-- ===================
-- Table: revenue_events (Financial Analytics)
-- ===================
CREATE TABLE IF NOT EXISTS analytics.revenue_events (
    event_id UUID DEFAULT generateUUIDv4(),
    order_id String,
    user_id String,
    
    event_type LowCardinality(String),  -- order_placed, payment_success, refund
    
    amount Decimal64(4),
    currency LowCardinality(String),
    payment_method LowCardinality(String),
    
    items_count UInt32,
    discount_amount Decimal64(4),
    shipping_fee Decimal64(4),
    
    created_at DateTime DEFAULT now()
) ENGINE = MergeTree()
PARTITION BY toYYYYMM(created_at)
ORDER BY (event_type, created_at)
TTL created_at + INTERVAL 730 DAY;  -- 2 years for financial data

-- ===================
-- Materialized View: Daily Revenue
-- ===================
CREATE MATERIALIZED VIEW IF NOT EXISTS analytics.mv_revenue_daily
ENGINE = SummingMergeTree()
PARTITION BY toYYYYMM(date)
ORDER BY (date, payment_method)
AS SELECT
    toDate(created_at) AS date,
    payment_method,
    count() AS order_count,
    sum(amount) AS total_revenue,
    sum(discount_amount) AS total_discounts,
    sum(shipping_fee) AS total_shipping,
    avg(amount) AS avg_order_value
FROM analytics.revenue_events
WHERE event_type = 'payment_success'
GROUP BY date, payment_method;
