-- ============================================================
-- Script: lock_renew_read
-- Purpose: READ 락 TTL 갱신
-- KEYS[1]: read lock hash key
-- ARGV[1]: token
-- ARGV[2]: ttlMillis
-- Returns: 1 if renewed, 0 otherwise
-- ============================================================
local tokenField = "token:" .. ARGV[1]
if redis.call("HGET", KEYS[1], tokenField) == ARGV[1] then
    redis.call("PEXPIRE", KEYS[1], ARGV[2])
    return 1
else
    return 0
end
