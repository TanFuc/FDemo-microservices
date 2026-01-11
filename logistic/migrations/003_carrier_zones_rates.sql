-- Carrier Configuration, Shipping Zones, Rates, and Delivery Slots Schema
-- Migration: 003_carrier_zones_rates.sql
-- Enterprise-grade shipping configuration for multi-carrier logistics

-- =====================================================
-- Enums
-- =====================================================

CREATE TYPE carrier_status AS ENUM ('ACTIVE', 'INACTIVE', 'TESTING');
CREATE TYPE zone_type AS ENUM ('COUNTRY', 'PROVINCE', 'DISTRICT', 'POSTAL', 'CUSTOM');
CREATE TYPE rate_type AS ENUM ('FLAT', 'WEIGHT', 'DISTANCE', 'TIERED', 'PROVIDER');
CREATE TYPE delivery_slot_status AS ENUM ('AVAILABLE', 'FULL', 'CLOSED');

-- =====================================================
-- Carrier Configurations Table
-- =====================================================
CREATE TABLE carrier_configs (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    provider VARCHAR(50) NOT NULL, -- GHN, GHTK, VIETTEL_POST, etc.
    name VARCHAR(100) NOT NULL UNIQUE,
    display_name VARCHAR(255) NOT NULL,
    description TEXT,
    logo_url VARCHAR(500),

    -- API Configuration (encrypted in production)
    api_endpoint VARCHAR(500),
    api_key VARCHAR(500),
    api_secret VARCHAR(500),
    api_version VARCHAR(20),
    webhook_secret VARCHAR(500),

    -- Service Configuration
    supports_cod BOOLEAN NOT NULL DEFAULT FALSE,
    supports_return BOOLEAN NOT NULL DEFAULT FALSE,
    supports_express BOOLEAN NOT NULL DEFAULT FALSE,
    supports_intl BOOLEAN NOT NULL DEFAULT FALSE,

    -- Limits and constraints
    max_weight DECIMAL(10, 2) DEFAULT 0, -- in kg, 0 = no limit
    max_dimensions JSONB DEFAULT '{}', -- {length, width, height} in cm
    max_cod_amount DECIMAL(19, 4) DEFAULT 0, -- 0 = no limit

    -- Pricing defaults
    base_fee DECIMAL(19, 4) DEFAULT 0,
    cod_fee_percent DECIMAL(5, 2) DEFAULT 0,
    insurance_fee DECIMAL(19, 4) DEFAULT 0,

    -- Service level configuration
    service_types JSONB DEFAULT '[]', -- Array of available service types
    cutoff_time TIME, -- Daily cutoff for same-day pickup

    -- Status
    status carrier_status NOT NULL DEFAULT 'TESTING',
    priority INTEGER NOT NULL DEFAULT 0,

    -- Rate limiting
    rate_limit_rps INTEGER DEFAULT 10,

    -- Metadata
    metadata JSONB DEFAULT '{}',
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);

-- Indexes for carrier_configs
CREATE INDEX idx_carrier_configs_provider ON carrier_configs(provider);
CREATE INDEX idx_carrier_configs_status ON carrier_configs(status);
CREATE INDEX idx_carrier_configs_priority ON carrier_configs(priority DESC) WHERE status = 'ACTIVE';

-- =====================================================
-- Shipping Zones Table
-- =====================================================
CREATE TABLE shipping_zones (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name VARCHAR(100) NOT NULL,
    display_name VARCHAR(255) NOT NULL,
    description TEXT,
    type zone_type NOT NULL,

    -- Geographic identification
    country_code CHAR(2) NOT NULL DEFAULT 'VN',
    province_code VARCHAR(20),
    district_code VARCHAR(20),
    postal_codes JSONB DEFAULT '[]', -- Array of postal codes

    -- Hierarchy
    parent_zone_id UUID REFERENCES shipping_zones(id),

    -- Configuration
    is_active BOOLEAN NOT NULL DEFAULT TRUE,
    priority INTEGER NOT NULL DEFAULT 0,

    -- Supported carriers in this zone
    carrier_ids JSONB DEFAULT '[]', -- Array of carrier UUIDs

    -- Zone-specific settings
    settings JSONB DEFAULT '{}',

    -- Metadata
    metadata JSONB DEFAULT '{}',
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    deleted_at TIMESTAMP WITH TIME ZONE,

    -- Unique constraint
    UNIQUE(name, country_code)
);

-- Indexes for shipping_zones
CREATE INDEX idx_shipping_zones_country ON shipping_zones(country_code) WHERE deleted_at IS NULL;
CREATE INDEX idx_shipping_zones_province ON shipping_zones(country_code, province_code) WHERE deleted_at IS NULL;
CREATE INDEX idx_shipping_zones_district ON shipping_zones(country_code, province_code, district_code) WHERE deleted_at IS NULL;
CREATE INDEX idx_shipping_zones_parent ON shipping_zones(parent_zone_id) WHERE deleted_at IS NULL;
CREATE INDEX idx_shipping_zones_active ON shipping_zones(is_active, priority DESC) WHERE deleted_at IS NULL;
CREATE INDEX idx_shipping_zones_type ON shipping_zones(type) WHERE deleted_at IS NULL;
CREATE INDEX idx_shipping_zones_postal_codes ON shipping_zones USING GIN (postal_codes) WHERE deleted_at IS NULL;

-- =====================================================
-- Shipping Rates Table
-- =====================================================
CREATE TABLE shipping_rates (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name VARCHAR(100) NOT NULL,
    display_name VARCHAR(255),
    description TEXT,

    -- Zone and carrier association
    origin_zone_id UUID NOT NULL REFERENCES shipping_zones(id),
    dest_zone_id UUID NOT NULL REFERENCES shipping_zones(id),
    carrier_id UUID REFERENCES carrier_configs(id),

    -- Service type
    service_type VARCHAR(50) NOT NULL DEFAULT 'STANDARD',

    -- Rate calculation
    rate_type rate_type NOT NULL DEFAULT 'FLAT',
    base_fee DECIMAL(19, 4) NOT NULL DEFAULT 0,
    per_kg_fee DECIMAL(19, 4) DEFAULT 0,
    per_km_fee DECIMAL(19, 4) DEFAULT 0,

    -- Weight-based tiers (for TIERED rate type)
    weight_tiers JSONB DEFAULT '[]',

    -- Minimum and maximum
    min_fee DECIMAL(19, 4) DEFAULT 0,
    max_fee DECIMAL(19, 4) DEFAULT 0, -- 0 = no max
    min_weight DECIMAL(10, 2) DEFAULT 0,
    max_weight DECIMAL(10, 2) DEFAULT 0, -- 0 = no max

    -- Estimated delivery
    est_delivery_min SMALLINT DEFAULT 1,
    est_delivery_max SMALLINT DEFAULT 3,

    -- Surcharges
    surcharges JSONB DEFAULT '[]',

    -- Free shipping threshold
    free_shipping_min DECIMAL(19, 4) DEFAULT 0, -- 0 = no free shipping

    -- Validity
    is_active BOOLEAN NOT NULL DEFAULT TRUE,
    valid_from TIMESTAMP WITH TIME ZONE,
    valid_until TIMESTAMP WITH TIME ZONE,

    -- Priority (higher = preferred)
    priority INTEGER NOT NULL DEFAULT 0,

    -- Currency
    currency CHAR(3) NOT NULL DEFAULT 'VND',

    -- Metadata
    metadata JSONB DEFAULT '{}',
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    deleted_at TIMESTAMP WITH TIME ZONE,

    -- Constraints
    CONSTRAINT check_delivery_range CHECK (est_delivery_max >= est_delivery_min),
    CONSTRAINT check_validity CHECK (valid_until IS NULL OR valid_from IS NULL OR valid_until > valid_from)
);

-- Indexes for shipping_rates
CREATE INDEX idx_shipping_rates_origin_dest ON shipping_rates(origin_zone_id, dest_zone_id) WHERE deleted_at IS NULL;
CREATE INDEX idx_shipping_rates_carrier ON shipping_rates(carrier_id) WHERE deleted_at IS NULL;
CREATE INDEX idx_shipping_rates_service ON shipping_rates(service_type) WHERE deleted_at IS NULL;
CREATE INDEX idx_shipping_rates_active ON shipping_rates(is_active, priority DESC) WHERE deleted_at IS NULL;
CREATE INDEX idx_shipping_rates_validity ON shipping_rates(valid_from, valid_until) WHERE deleted_at IS NULL AND is_active = TRUE;

-- Composite index for rate lookup
CREATE INDEX idx_shipping_rates_lookup
    ON shipping_rates(origin_zone_id, dest_zone_id, service_type, is_active, priority DESC)
    WHERE deleted_at IS NULL;

-- =====================================================
-- Delivery Slot Templates Table
-- =====================================================
CREATE TABLE delivery_slot_templates (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name VARCHAR(100) NOT NULL,
    display_name VARCHAR(255) NOT NULL,

    -- Zone and carrier association
    zone_id UUID REFERENCES shipping_zones(id),
    carrier_id UUID REFERENCES carrier_configs(id),

    -- Time configuration
    start_time TIME NOT NULL,
    end_time TIME NOT NULL,
    duration INTEGER DEFAULT 120, -- Duration in minutes

    -- Capacity
    max_orders INTEGER NOT NULL DEFAULT 50,

    -- Days of week (bitmask)
    days_of_week SMALLINT NOT NULL DEFAULT 127, -- All days

    -- Lead time requirements
    min_lead_hours INTEGER DEFAULT 2,
    max_lead_days INTEGER DEFAULT 7,

    -- Cutoff time for same-day slots
    cutoff_time TIME,

    -- Pricing
    slot_fee DECIMAL(19, 4) DEFAULT 0,
    is_premium BOOLEAN NOT NULL DEFAULT FALSE,

    -- Status
    is_active BOOLEAN NOT NULL DEFAULT TRUE,

    -- Metadata
    metadata JSONB DEFAULT '{}',
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),

    -- Constraints
    CONSTRAINT check_time_range CHECK (end_time > start_time),
    CONSTRAINT check_days_of_week CHECK (days_of_week >= 0 AND days_of_week <= 127)
);

-- Indexes for delivery_slot_templates
CREATE INDEX idx_delivery_slot_templates_zone ON delivery_slot_templates(zone_id) WHERE is_active = TRUE;
CREATE INDEX idx_delivery_slot_templates_carrier ON delivery_slot_templates(carrier_id) WHERE is_active = TRUE;
CREATE INDEX idx_delivery_slot_templates_active ON delivery_slot_templates(is_active);

-- =====================================================
-- Delivery Slots Table (Generated from templates)
-- =====================================================
CREATE TABLE delivery_slots (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    template_id UUID NOT NULL REFERENCES delivery_slot_templates(id),

    -- Date and time
    date DATE NOT NULL,
    start_time TIMESTAMP WITH TIME ZONE NOT NULL,
    end_time TIMESTAMP WITH TIME ZONE NOT NULL,

    -- Zone and carrier
    zone_id UUID REFERENCES shipping_zones(id),
    carrier_id UUID REFERENCES carrier_configs(id),

    -- Capacity tracking
    max_orders INTEGER NOT NULL,
    booked_orders INTEGER NOT NULL DEFAULT 0,
    remaining_slots INTEGER NOT NULL,

    -- Status
    status delivery_slot_status NOT NULL DEFAULT 'AVAILABLE',

    -- Pricing
    slot_fee DECIMAL(19, 4) DEFAULT 0,
    is_premium BOOLEAN NOT NULL DEFAULT FALSE,

    -- Timestamps
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),

    -- Unique constraint per template per date
    UNIQUE(template_id, date),

    -- Constraints
    CONSTRAINT check_slot_capacity CHECK (booked_orders <= max_orders),
    CONSTRAINT check_remaining_slots CHECK (remaining_slots >= 0)
);

-- Indexes for delivery_slots
CREATE INDEX idx_delivery_slots_date ON delivery_slots(date);
CREATE INDEX idx_delivery_slots_zone_date ON delivery_slots(zone_id, date) WHERE status = 'AVAILABLE';
CREATE INDEX idx_delivery_slots_carrier_date ON delivery_slots(carrier_id, date) WHERE status = 'AVAILABLE';
CREATE INDEX idx_delivery_slots_available ON delivery_slots(date, status) WHERE status = 'AVAILABLE';

-- =====================================================
-- Delivery Slot Bookings Table
-- =====================================================
CREATE TABLE delivery_slot_bookings (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    slot_id UUID NOT NULL REFERENCES delivery_slots(id),
    order_id UUID NOT NULL,
    shipping_order_id UUID REFERENCES shipping_orders(id),

    -- Booking status
    status VARCHAR(20) NOT NULL DEFAULT 'BOOKED',

    -- Customer preferences
    contact_phone VARCHAR(20) NOT NULL,
    instructions TEXT,

    -- Timestamps
    booked_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    confirmed_at TIMESTAMP WITH TIME ZONE,
    cancelled_at TIMESTAMP WITH TIME ZONE,
    completed_at TIMESTAMP WITH TIME ZONE,

    -- Cancellation
    cancelled_by UUID,
    cancel_reason TEXT,

    -- Timestamps
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),

    -- Unique constraint
    UNIQUE(order_id)
);

-- Indexes for delivery_slot_bookings
CREATE INDEX idx_delivery_slot_bookings_slot ON delivery_slot_bookings(slot_id);
CREATE INDEX idx_delivery_slot_bookings_order ON delivery_slot_bookings(order_id);
CREATE INDEX idx_delivery_slot_bookings_status ON delivery_slot_bookings(status);
CREATE INDEX idx_delivery_slot_bookings_shipping ON delivery_slot_bookings(shipping_order_id) WHERE shipping_order_id IS NOT NULL;

-- =====================================================
-- Triggers
-- =====================================================

-- Auto-update updated_at
CREATE OR REPLACE FUNCTION update_updated_at_logistics()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER update_carrier_configs_updated_at
    BEFORE UPDATE ON carrier_configs
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_logistics();

CREATE TRIGGER update_shipping_zones_updated_at
    BEFORE UPDATE ON shipping_zones
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_logistics();

CREATE TRIGGER update_shipping_rates_updated_at
    BEFORE UPDATE ON shipping_rates
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_logistics();

CREATE TRIGGER update_delivery_slot_templates_updated_at
    BEFORE UPDATE ON delivery_slot_templates
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_logistics();

CREATE TRIGGER update_delivery_slots_updated_at
    BEFORE UPDATE ON delivery_slots
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_logistics();

CREATE TRIGGER update_delivery_slot_bookings_updated_at
    BEFORE UPDATE ON delivery_slot_bookings
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_logistics();

-- =====================================================
-- Comments
-- =====================================================
COMMENT ON TABLE carrier_configs IS 'Configuration for shipping carriers (GHN, GHTK, ViettelPost, etc.)';
COMMENT ON TABLE shipping_zones IS 'Geographic shipping zones for rate calculation';
COMMENT ON TABLE shipping_rates IS 'Shipping rates per zone pair and carrier';
COMMENT ON TABLE delivery_slot_templates IS 'Templates for recurring delivery time slots';
COMMENT ON TABLE delivery_slots IS 'Actual delivery slots generated from templates';
COMMENT ON TABLE delivery_slot_bookings IS 'Customer bookings for specific delivery slots';
