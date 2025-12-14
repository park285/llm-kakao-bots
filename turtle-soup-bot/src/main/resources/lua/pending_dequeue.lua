-- 원자적 dequeue 연산: LPOP + Stale 체크 + SREM
-- KEYS[1]: queueKey (큐)
-- KEYS[2]: userSetKey (사용자 중복 방지 Set)
-- ARGV[1]: currentTimestamp (현재 시각 ms)
-- ARGV[2]: staleThresholdMs (Stale 임계값 ms)
-- ARGV[3]: maxIterations (루프 제한, Redis 블로킹 방지)
--
-- 저장 포맷: timestamp|JSON
-- 반환값:
--   nil: 큐가 비어있음
--   "EXHAUSTED": 루프 제한 도달 (뒤에 데이터 있을 수 있음)
--   JSON 문자열: 유효한 메시지

local queueKey = KEYS[1]
local userSetKey = KEYS[2]
local currentTimestamp = tonumber(ARGV[1])
local staleThresholdMs = tonumber(ARGV[2])
local maxIterations = tonumber(ARGV[3] or "50")

-- timestamp|JSON 포맷에서 timestamp와 JSON 분리 
local function extractTimestamp(data)
    local delimPos = string.find(data, "|", 1, true)
    if not delimPos then
        return nil, nil
    end
    local ts = tonumber(string.sub(data, 1, delimPos - 1))
    local json = string.sub(data, delimPos + 1)
    return ts, json
end

-- JSON에서 userId 추출 (SREM용)
local function extractUserId(json)
    local ok, message = pcall(cjson.decode, json)
    if ok and message.userId then
        return message.userId
    end
    return nil
end

local iterations = 0

while iterations < maxIterations do
    iterations = iterations + 1

    local data = redis.call("LPOP", queueKey)
    if not data then
        return nil
    end

    local timestamp, json = extractTimestamp(data)

    -- 유효한 포맷일 때만 처리
    if timestamp and json then
        local userId = extractUserId(json)
        if userId then
            redis.call("SREM", userSetKey, userId)
        end

        -- Stale 체크: 정상 메시지면 반환, stale이면 다음 iteration
        local age = currentTimestamp - timestamp
        if age <= staleThresholdMs then
            return json
        end
    end
    -- 잘못된 포맷 또는 stale: 자동으로 다음 iteration
end

return "EXHAUSTED"
