-- ============================================================
-- Script: mq_process_with_idempotency
-- Purpose: 멱등성 상태 기반 처리 결정 + processing 마킹
-- KEYS[1]: idempotency_key
-- KEYS[2]: stream_key
-- ARGV[1]: consumer_group
-- ARGV[2]: message_id
-- ARGV[3]: processing_ttl_seconds
-- Returns:
--  1: 처리 시작(락 획득)
--  0: 이미 완료됨(ACK 후 스킵)
-- -1: 처리 중(ACK 금지, 재시도/클레임 로직에 위임)
-- ============================================================

local idempotency_key = KEYS[1]
local stream_key = KEYS[2]
local group = ARGV[1]
local msg_id = ARGV[2]
local processing_ttl = tonumber(ARGV[3])

local current_val = redis.call('GET', idempotency_key)

if current_val == 'completed' then
    -- 이미 완료된 작업: ACK 처리 후 스킵
    redis.call('XACK', stream_key, group, msg_id)
    return 0
elseif current_val then
    -- processing(또는 알 수 없는 값)인 경우 ACK를 보내면 데이터 유실 위험이 있음
    return -1
end

redis.call('SETEX', idempotency_key, processing_ttl, 'processing')
return 1
