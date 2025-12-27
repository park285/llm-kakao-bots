-- ============================================================
-- Script: lock_acquire_read
-- Purpose: READ 락 획득 (원자적)
-- KEYS[1]: write lock key
-- KEYS[2]: read lock hash key
-- ARGV[1]: token
-- ARGV[2]: ttlMillis
-- Returns: 1 if acquired, 0 otherwise
-- ============================================================
if redis.call("EXISTS", KEYS[1]) == 1 then
    return 0
else
    redis.call("HINCRBY", KEYS[2], "counter", 1)
    redis.call("HSET", KEYS[2], "token:" .. ARGV[1], ARGV[1])
    redis.call("PEXPIRE", KEYS[2], ARGV[2])
    return 1
end
