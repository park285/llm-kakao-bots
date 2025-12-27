-- ============================================================
-- Script: mq_process_with_idempotency
-- Purpose: idempotency check + mark processing + ACK
-- KEYS[1]: idempotency_key
-- KEYS[2]: stream_key
-- ARGV[1]: consumer_group
-- ARGV[2]: message_id
-- ARGV[3]: ttl_seconds
-- Returns: 1 if should process, 0 if already processed
-- ============================================================

local idempotency_key = KEYS[1]
local stream_key = KEYS[2]
local group = ARGV[1]
local msg_id = ARGV[2]
local ttl = tonumber(ARGV[3])

if redis.call('EXISTS', idempotency_key) == 1 then
    redis.call('XACK', stream_key, group, msg_id)
    return 0
end

redis.call('SETEX', idempotency_key, ttl, 'processing')
return 1
