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
-- Security Notes:
--   PTTL=-2: 키가 만료되어 사라짐 (Race Condition) → 키 재생성 후 허용
--   PTTL=-1: 키에 TTL 없음 (Zombie Key) → Self-Healing: TTL 강제 설정 후 차단
-- ============================================================

local key = KEYS[1]
local ttlSeconds = tonumber(ARGV[1])

-- [Security] 입력값 검증: 잘못된 파라미터로 인한 Redis Panic 방지
if not ttlSeconds or ttlSeconds <= 0 then
    return redis.error_reply("ERR invalid ttl")
end

local ttlMillis = ttlSeconds * 1000

-- 1. SET NX 시도 (신규 차단 윈도우 시작)
local ok = redis.call('SET', key, '1', 'NX', 'PX', ttlMillis)
if ok then
    return {1, 0} -- 허용 (새 Rate Limit 윈도우 시작)
end

-- 2. SET 실패: 이미 키가 존재함. 남은 시간 조회
local remaining = redis.call('PTTL', key)

-- [Stability] Race Condition & Zombie Key 방어
if remaining == -2 then
    -- 상황: SET NX 실패 직후 키가 만료되어 사라짐 (Race Condition)
    -- 조치: 즉시 키를 다시 생성하여 이번 요청을 카운트하고 허용
    redis.call('SET', key, '1', 'PX', ttlMillis)
    return {1, 0}
elseif remaining == -1 then
    -- 상황: 키는 있는데 만료 시간이 없음 (Zombie Key - 보안 위협)
    -- 조치: [Self-Healing] 강제로 만료 시간을 설정하고, 이번 요청은 차단
    redis.call('PEXPIRE', key, ttlMillis)
    return {0, ttlMillis}
end

return {0, remaining}

