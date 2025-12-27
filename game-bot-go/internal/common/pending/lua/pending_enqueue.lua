-- ============================================================
-- Script: pending_enqueue
-- Purpose: 대기열 메시지 추가 (HASH + ZSET)
-- KEYS[1]: dataKey (HASH)
-- KEYS[2]: orderKey (ZSET)
-- ARGV[1]: userId
-- ARGV[2]: messageValue (JSON)
-- ARGV[3]: timestamp (score)
-- ARGV[4]: maxSize
-- ARGV[5]: ttlSeconds
-- ARGV[6]: replaceOnDuplicate ("1" or "0")
-- Returns: "SUCCESS" | "DUPLICATE" | "QUEUE_FULL"
-- ============================================================

local dataKey = KEYS[1]
local orderKey = KEYS[2]
local userId = ARGV[1]
local messageValue = ARGV[2]
local timestamp = tonumber(ARGV[3])
local maxSize = tonumber(ARGV[4])
local ttl = tonumber(ARGV[5])
local replaceOnDuplicate = ARGV[6] == "1"

local STATUS_SUCCESS = "SUCCESS"
local STATUS_DUPLICATE = "DUPLICATE"
local STATUS_QUEUE_FULL = "QUEUE_FULL"

-- 1. 중복 체크 (HASH에 UserID가 있는지 확인)
local exists = redis.call("HEXISTS", dataKey, userId)

if exists == 1 then
    if replaceOnDuplicate then
        -- 교체 모드: 데이터 업데이트 및 순서 갱신(Score 업데이트 -> 맨 뒤로 이동)
        redis.call("HSET", dataKey, userId, messageValue)
        redis.call("ZADD", orderKey, timestamp, userId)
        
        -- TTL 갱신
        redis.call("EXPIRE", dataKey, ttl)
        redis.call("EXPIRE", orderKey, ttl)
        
        return STATUS_SUCCESS
    else
        return STATUS_DUPLICATE
    end
end

-- 2. 큐 크기 체크 (ZCARD)
local size = redis.call("ZCARD", orderKey)
if size >= maxSize then
    return STATUS_QUEUE_FULL
end

-- 3. 신규 추가
redis.call("HSET", dataKey, userId, messageValue)
redis.call("ZADD", orderKey, timestamp, userId)

-- 4. TTL 설정
redis.call("EXPIRE", dataKey, ttl)
redis.call("EXPIRE", orderKey, ttl)

return STATUS_SUCCESS
