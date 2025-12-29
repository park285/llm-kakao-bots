-- ============================================================
-- Script: pending_dequeue_batch
-- Purpose: 대기열 메시지 다건 꺼내기 (HASH + ZSET)
-- KEYS[1]: dataKey (HASH)
-- KEYS[2]: orderKey (ZSET)
-- ARGV[1]: batchSize
-- ============================================================
-- Returns:
--   Empty: nil
--   Success: {userId, score, json, ...}
--   Inconsistent: {"INCONSISTENT", userId, ...}
-- ============================================================

local dataKey = KEYS[1]
local orderKey = KEYS[2]
local batchSize = tonumber(ARGV[1])

if not batchSize or batchSize < 1 then
    return nil
end

local members = redis.call("ZRANGE", orderKey, 0, batchSize - 1)
if #members == 0 then
    return nil
end

local result = {}
for i = 1, #members do
    local userId = members[i]
    local val = redis.call("HGET", dataKey, userId)
    if not val then
        redis.call("ZREM", orderKey, userId)
        table.insert(result, "INCONSISTENT")
        table.insert(result, userId)
    else
        local score = redis.call("ZSCORE", orderKey, userId)
        redis.call("ZREM", orderKey, userId)
        redis.call("HDEL", dataKey, userId)
        table.insert(result, userId)
        table.insert(result, score)
        table.insert(result, val)
    end
end

return result
