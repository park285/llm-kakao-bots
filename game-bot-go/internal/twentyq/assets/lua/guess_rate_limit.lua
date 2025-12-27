-- ============================================================
-- Script: guess_rate_limit
-- Purpose: 정답 시도 개인별 Rate Limit (원자적 SET NX + PTTL 조회)
-- ============================================================
-- KEYS[1]: rate_limit_key (사용자별 Rate Limit 키)
-- ARGV[1]: ttl_seconds (제한 시간, 초 단위)
-- ============================================================
-- Returns:
--   허용: {1, 0}
--   제한: {0, remaining_ms} (남은 시간, 밀리초)
-- Error:
--   ERR invalid ttl (ttl_seconds가 0 이하인 경우)
-- ============================================================

local key = KEYS[1]
local ttlSeconds = tonumber(ARGV[1])
if not ttlSeconds or ttlSeconds <= 0 then
    return redis.error_reply("ERR invalid ttl")
end

local ttlMillis = ttlSeconds * 1000

-- 1. 먼저 SET NX를 시도합니다. (가장 흔한 '성공' 케이스를 최적화)
-- 성공 시 'OK'가 반환되고, 실패 시(이미 키가 있음) nil이 반환됩니다.
local ok = redis.call('SET', key, '1', 'NX', 'PX', ttlMillis)

if ok then
    return {1, 0} -- 성공적으로 Rate Limit 통과
end

-- 2. SET에 실패했다면 이미 제한이 걸려있는 상태입니다. 남은 시간을 반환합니다.
local remaining = redis.call('PTTL', key)
if remaining < 0 then
    return {1, 0}
end
return {0, remaining}
