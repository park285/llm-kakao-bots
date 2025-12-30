-- ============================================================
-- Script: mq_complete_processing
-- Purpose: completed 마킹 + ACK
-- KEYS[1]: idempotency_key
-- KEYS[2]: stream_key
-- ARGV[1]: consumer_group
-- ARGV[2]: message_id
-- ARGV[3]: retention_ttl_seconds
-- Returns: 1
-- ============================================================

local idempotency_key = KEYS[1]
local stream_key = KEYS[2]
local group = ARGV[1]
local msg_id = ARGV[2]
local retention_ttl = tonumber(ARGV[3])

redis.call('SETEX', idempotency_key, retention_ttl, 'completed')
redis.call('XACK', stream_key, group, msg_id)
return 1
