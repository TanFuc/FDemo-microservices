local key = KEYS[1]
local qty = tonumber(ARGV[1])

local total = tonumber(redis.call("HGET", key, "total") or 0)
local reserved = tonumber(redis.call("HGET", key, "reserved") or 0)

if (total - reserved) >= qty then
    redis.call("HINCRBY", key, "reserved", qty)
    return 1 -- Success
else
    return 0 -- Failed (Out of Stock)
end
