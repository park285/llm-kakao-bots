-- ============================================================
-- Script: lock_release
-- Purpose: WRITE 락 해제
-- KEYS[1]: write lock key
-- ARGV[1]: token
-- Returns: 1 if released, 0 otherwise
-- ============================================================
if redis.call("GET", KEYS[1]) == ARGV[1] then
    return redis.call("DEL", KEYS[1])
else
    return 0
end
