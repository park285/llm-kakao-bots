-- Hybrid Dequeue: ZPOPMIN (or ZRANGE+ZREM) + HGET + HDEL
-- KEYS[1]: dataKey (HASH)
-- KEYS[2]: orderKey (ZSET)
-- ARGV[1]: staleThresholdMS (Deprecated/Unused in ZSET logic but kept for interface compat)

local dataKey = KEYS[1]
local orderKey = KEYS[2]

-- 1. 가장 오래된 항목(Score가 가장 작은) 조회
-- ZPOPMIN은 Redis 5.0+ 지원. 안전하게 ZRANGE + ZREM 사용
local members = redis.call("ZRANGE", orderKey, 0, 0)
if #members == 0 then
    return nil -- Empty
end

local userId = members[1]

-- 2. 메시지 데이터 조회
local val = redis.call("HGET", dataKey, userId)
local score = redis.call("ZSCORE", orderKey, userId)

-- 3. 데이터가 HASH에 없으면 ZSET에서도 정리 (Inconsistency Fix)
if not val then
    redis.call("ZREM", orderKey, userId)
    -- 재귀적으로 다음 항목 시도할 수도 있으나, 일단 빈 결과 반환하여 다음 폴링 유도
    return nil 
end

-- 4. 큐에서 제거 (Commit Dequeue)
redis.call("ZREM", orderKey, userId)
redis.call("HDEL", dataKey, userId)

return {userId, score, val}
