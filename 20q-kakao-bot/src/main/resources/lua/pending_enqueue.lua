local queueKey = KEYS[1]
local userSetKey = KEYS[2]
local userId = ARGV[1]
local messageJson = ARGV[2]
local maxSize = tonumber(ARGV[3])
local ttl = tonumber(ARGV[4])

if redis.call("SISMEMBER", userSetKey, userId) == 1 then
    return "DUPLICATE"
end

local size = redis.call("LLEN", queueKey)
if size >= maxSize then
    return "QUEUE_FULL"
end

redis.call("RPUSH", queueKey, messageJson)
redis.call("SADD", userSetKey, userId)
redis.call("EXPIRE", queueKey, ttl)
redis.call("EXPIRE", userSetKey, ttl)

return "SUCCESS"
