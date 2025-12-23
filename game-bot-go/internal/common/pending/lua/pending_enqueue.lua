-- Hybrid Enqueue: HASH(Data) + ZSET(Order)
-- KEYS[1]: dataKey (Check & Data Storage - HASH)
-- KEYS[2]: orderKey (Order Queue - ZSET)
-- ARGV[1]: userId (Member Key)
-- ARGV[2]: messageValue (JSON Data)
-- ARGV[3]: timestamp (Score)
-- ARGV[4]: maxSize (Queue Limit)
-- ARGV[5]: ttl (Expiration)
-- ARGV[6]: replaceOnDuplicate ("1" or "0")

local dataKey = KEYS[1]
local orderKey = KEYS[2]
local userId = ARGV[1]
local messageValue = ARGV[2]
local timestamp = tonumber(ARGV[3])
local maxSize = tonumber(ARGV[4])
local ttl = tonumber(ARGV[5])
local replaceOnDuplicate = ARGV[6] == "1"

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
        
        return "SUCCESS"
    else
        return "DUPLICATE"
    end
end

-- 2. 큐 크기 체크 (ZCARD)
local size = redis.call("ZCARD", orderKey)
if size >= maxSize then
    return "QUEUE_FULL"
end

-- 3. 신규 추가
redis.call("HSET", dataKey, userId, messageValue)
redis.call("ZADD", orderKey, timestamp, userId)

-- 4. TTL 설정
redis.call("EXPIRE", dataKey, ttl)
redis.call("EXPIRE", orderKey, ttl)

return "SUCCESS"
