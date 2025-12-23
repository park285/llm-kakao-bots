-- WRITE 락 TTL 갱신
-- KEYS[1]: write lock key
-- ARGV[1]: token
-- ARGV[2]: ttlMillis
if redis.call("GET", KEYS[1]) == ARGV[1] then
    return redis.call("PEXPIRE", KEYS[1], ARGV[2])
else
    return 0
end
