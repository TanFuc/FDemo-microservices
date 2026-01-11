-- ============================================================================
-- Claim Voucher Lua Script
-- Service: tafu-campaign
-- Purpose: Atomically claim a voucher (check user + stock)
-- Keys: 
--   KEYS[1] = voucher:{code}:stock (STRING - integer)
--   KEYS[2] = voucher:{code}:users (SET - user IDs who claimed)
-- Args:
--   ARGV[1] = userId
-- Returns: 
--   1 = success
--   -1 = already claimed
--   0 = out of stock
-- ============================================================================

local stockKey = KEYS[1]
local usersKey = KEYS[2]
local userId = ARGV[1]

-- Step 1: Check if user already claimed
local alreadyClaimed = redis.call("SISMEMBER", usersKey, userId)
if alreadyClaimed == 1 then
    return -1  -- Already claimed
end

-- Step 2: Check stock
local stock = tonumber(redis.call("GET", stockKey) or 0)
if stock <= 0 then
    return 0  -- Out of stock
end

-- Step 3: Atomic claim
-- Decrement stock
redis.call("DECR", stockKey)
-- Add user to claimed set
redis.call("SADD", usersKey, userId)

return 1  -- Success
