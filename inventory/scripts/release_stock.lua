-- ============================================================================
-- Release Stock Lua Script
-- Service: tafu-inventory
-- Purpose: Release previously reserved stock
-- Key: inventory:{skuId} (HASH with fields: total, reserved)
-- Returns: 1 = success
-- ============================================================================

local key = KEYS[1]
local qty = tonumber(ARGV[1])

-- Get current reserved
local reserved = tonumber(redis.call("HGET", key, "reserved") or 0)

-- Calculate new reserved (don't go below 0)
local new_reserved = math.max(0, reserved - qty)

-- Update reserved stock
redis.call("HSET", key, "reserved", new_reserved)

return 1  -- Always success
