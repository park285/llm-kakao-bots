-- 원자적 enqueue 연산: 중복 체크 + 큐 크기 체크 + RPUSH + SADD
-- KEYS[1]: queueKey (큐)
-- KEYS[2]: userSetKey (사용자 중복 방지 Set)
-- ARGV[1]: userId (사용자 ID)
-- ARGV[2]: messageJson (메시지 JSON)
-- ARGV[3]: maxSize (큐 최대 크기)
-- ARGV[4]: ttl (TTL 초)

local queueKey = KEYS[1]
local userSetKey = KEYS[2]
local userId = ARGV[1]
local messageJson = ARGV[2]
local maxSize = tonumber(ARGV[3])
local ttl = tonumber(ARGV[4])

-- 중복 체크
if redis.call("SISMEMBER", userSetKey, userId) == 1 then
    return "DUPLICATE"
end

-- 큐 크기 체크
local size = redis.call("LLEN", queueKey)
if size >= maxSize then
    return "QUEUE_FULL"
end

-- 큐에 추가
redis.call("RPUSH", queueKey, messageJson)
redis.call("SADD", userSetKey, userId)

-- TTL 설정
redis.call("EXPIRE", queueKey, ttl)
redis.call("EXPIRE", userSetKey, ttl)

return "SUCCESS"
