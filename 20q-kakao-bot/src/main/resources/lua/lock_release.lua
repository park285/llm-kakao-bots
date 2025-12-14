-- KEYS[1]: write lock key
-- ARGV[1]: token
if redis.call("GET", KEYS[1]) == ARGV[1] then
    return redis.call("DEL", KEYS[1])
else
    return 0
end
