-- KEYS[1]: write lock key
-- KEYS[2]: read hash key
-- ARGV[1]: token
-- ARGV[2]: ttlMillis
if redis.call("EXISTS", KEYS[1]) == 1 then
    return 0
else
    redis.call("HINCRBY", KEYS[2], "counter", 1)
    redis.call("HSET", KEYS[2], "token:" .. ARGV[1], ARGV[1])
    redis.call("PEXPIRE", KEYS[2], ARGV[2])
    return 1
end
