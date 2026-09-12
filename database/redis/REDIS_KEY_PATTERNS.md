# ============================================================================
# TAFU Redis Data Structures & Lua Scripts
# Version: 1.0.0
# Date: 2026-01-07
# Purpose: Document all Redis keys, data structures, and Lua scripts
# ============================================================================

# =======================
# KEY NAMING CONVENTIONS
# =======================
# Format: {service}:{entity}:{id}:{attribute}
# Examples:
#   - identity:user:uuid123:permissions
#   - cart:user123
#   - inventory:SKU001

# =======================
# SERVICE: tafu-auth (Identity)
# =======================

# --- User Permissions Cache ---
# Key: identity:user:{userId}:permissions
# Type: SET
# TTL: 3600 (1 hour)
# Purpose: Cache user permissions to avoid DB lookup on every request
# Example:
#   SADD identity:user:550e8400-e29b-41d4-a716-446655440000:permissions "product:create" "order:read_own"
#   SMEMBERS identity:user:550e8400-e29b-41d4-a716-446655440000:permissions

# --- Token Blacklist ---
# Key: identity:blacklist:{jti}
# Type: STRING
# TTL: Remaining token expiry time
# Purpose: Invalidate revoked JWT tokens
# Example:
#   SETEX identity:blacklist:abc123jti 900 "1"
#   EXISTS identity:blacklist:abc123jti

# --- Rate Limit ---
# Key: ratelimit:{ip}:{endpoint}
# Type: STRING (counter)
# TTL: Window duration (e.g., 60 seconds)
# Example:
#   INCR ratelimit:192.168.1.1:/api/v1/auth/login
#   EXPIRE ratelimit:192.168.1.1:/api/v1/auth/login 60

# =======================
# SERVICE: tafu-cart
# =======================

# --- Shopping Cart ---
# Key: cart:{userId}
# Type: HASH
# TTL: 2592000 (30 days), refreshed on interaction
# Fields: {skuId} -> JSON string
# Example:
#   HSET cart:user123 "SKU001" '{"skuId":"SKU001","name":"Cotton T-Shirt","price":150000,"quantity":2,"thumbnail":"url","selected":true,"addedAt":1709223300}'
#   HGETALL cart:user123
#   EXPIRE cart:user123 2592000

# =======================
# SERVICE: tafu-inventory
# =======================

# --- Stock Cache ---
# Key: inventory:{skuId}
# Type: HASH
# TTL: None (synced with DB)
# Fields: total, reserved
# Example:
#   HSET inventory:SKU001 total 100 reserved 10
#   HGET inventory:SKU001 total
#   HINCRBY inventory:SKU001 reserved 5

# --- Lua Script: Reserve Stock (Atomic) ---
# Purpose: Atomically check and reserve stock
# Returns: 1 (success) or 0 (out of stock)
# File: scripts/reserve_stock.lua

# =======================
# SERVICE: tafu-campaign
# =======================

# --- Voucher Stock ---
# Key: voucher:{code}:stock
# Type: STRING (integer)
# TTL: Campaign end time
# Example:
#   SET voucher:SALE2024:stock 1000
#   DECR voucher:SALE2024:stock

# --- Voucher Claimed Users ---
# Key: voucher:{code}:users
# Type: SET
# TTL: Campaign end time
# Purpose: Track which users have claimed (prevent double-claim)
# Example:
#   SADD voucher:SALE2024:users user123
#   SISMEMBER voucher:SALE2024:users user123

# --- Lua Script: Atomic Claim ---
# Purpose: Check user hasn't claimed + stock > 0, then claim
# Returns: 1 (success), -1 (already claimed), 0 (out of stock)
# File: scripts/claim_voucher.lua

# =======================
# SERVICE: tafu-search
# =======================

# --- Search Results Cache ---
# Key: search:{hash}
# Type: STRING (JSON)
# TTL: 120 (2 minutes)
# Purpose: Cache search results to reduce Elasticsearch load
# Hash: MD5/SHA1 of search params JSON
# Example:
#   SET search:abc123hash '{"results":[...],"total":100}' EX 120

# =======================
# SERVICE: tafu-logistic
# =======================

# --- Shipping Fee Cache ---
# Key: fee:{provider}:{from}:{to}:{weight}
# Type: STRING (decimal)
# TTL: 3600 (1 hour)
# Purpose: Cache calculated shipping fees
# Example:
#   SET fee:GHN:1:2:500 "25000" EX 3600

# =======================
# SERVICE: tafu-review
# =======================

# --- Product Rating Summary Cache ---
# Key: rating:{productId}
# Type: STRING (JSON)
# TTL: 1800 (30 minutes)
# Purpose: Cache product rating aggregation
# Example:
#   SET rating:prod123 '{"average":4.5,"total":150,"breakdown":{"5":100,"4":30,"3":15,"2":3,"1":2}}' EX 1800

# =======================
# SERVICE: tafu-api-gateway
# =======================

# --- Rate Limiter ---
# Key: rate:{userId}:{windowId}
# Type: STRING (counter)
# TTL: Rate limit window
# Example using sliding window:
#   INCR rate:user123:1709223300
#   EXPIRE rate:user123:1709223300 60

# =======================
# SERVICE: tafu-analytic  
# =======================

# --- Real-time Counters ---
# Key: stats:live:{metric}:{timeWindow}
# Type: SORTED SET
# Purpose: Real-time stats with time-based members
# Example:
#   ZADD stats:live:pageviews:hourly 1709223300 "product123:1709223300"
#   ZRANGEBYSCORE stats:live:pageviews:hourly 1709220000 1709223600

# =======================
# SERVICE: tafu-notification
# =======================

# --- Processing Lock ---
# Key: notification:lock:{jobId}
# Type: STRING
# TTL: Job timeout
# Purpose: Prevent duplicate processing
# Example:
#   SETNX notification:lock:email123 "worker1"
#   EXPIRE notification:lock:email123 300
