-- ============================================================
-- Script: pending_dequeue
-- Purpose: 대기열 메시지 꺼내기 (HASH + ZSET)
-- KEYS[1]: dataKey (HASH)
-- KEYS[2]: orderKey (ZSET)
-- ARGV[1]: staleThresholdMS (호환용, ZSET 로직에서는 미사용)
-- ============================================================
-- Returns:
--   Empty: nil
--   Success: {userId, score, json}
--   Inconsistent: {"INCONSISTENT", userId} (ZSET에만 존재, HASH에 없음)
-- ============================================================

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

-- 3. [Stability] 데이터 불일치 방어: ZSET에만 존재, HASH에 없음
if not val then
    -- ZSET에서 정리 (Self-Healing)
    redis.call("ZREM", orderKey, userId)
    -- Go 클라이언트에 명시적 INCONSISTENT 반환 → 즉시 재시도 유도
    return {"INCONSISTENT", userId}
end

-- 4. 큐에서 제거 (Commit Dequeue)
redis.call("ZREM", orderKey, userId)
redis.call("HDEL", dataKey, userId)

return {userId, score, val}

