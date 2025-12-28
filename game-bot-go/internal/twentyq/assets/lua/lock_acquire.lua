-- ============================================================
-- Script: lock_acquire
-- Purpose: 배타적 락 획득 (원자적)
-- KEYS[1]: lock key
-- ARGV[1]: token
-- ARGV[2]: ttlMillis
-- Returns: 1 if acquired, 0 otherwise
-- ============================================================
local ok = redis.call("SET", KEYS[1], ARGV[1], "NX", "PX", ARGV[2])
if ok then
    return 1
end
return 0
