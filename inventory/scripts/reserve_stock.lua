-- ============================================================================
-- Reserve Stock Lua Script
-- Service: tafu-inventory
-- Purpose: Atomically check and reserve stock
-- Key: inventory:{skuId} (HASH with fields: total, reserved)
-- Returns: 1 = success, 0 = out of stock
-- ============================================================================

local key = KEYS[1]
local qty = tonumber(ARGV[1])

-- Get current values
local total = tonumber(redis.call("HGET", key, "total") or 0)
local reserved = tonumber(redis.call("HGET", key, "reserved") or 0)

-- Calculate available stock
local available = total - reserved

-- Check if enough stock
if available >= qty then
    -- Reserve the stock
    redis.call("HINCRBY", key, "reserved", qty)
    return 1  -- Success
else
    return 0  -- Out of stock
end
