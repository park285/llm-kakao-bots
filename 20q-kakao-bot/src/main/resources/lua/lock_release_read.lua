-- KEYS[1]: read hash key
-- ARGV[1]: token
-- ARGV[2]: ttlMillis
local tokenField = "token:" .. ARGV[1]
if redis.call("HGET", KEYS[1], tokenField) == ARGV[1] then
    redis.call("HDEL", KEYS[1], tokenField)
    local count = redis.call("HINCRBY", KEYS[1], "counter", -1)
    if count <= 0 then
        redis.call("DEL", KEYS[1])
    else
        redis.call("PEXPIRE", KEYS[1], ARGV[2])
    end
    return 1
else
    return 0
end
