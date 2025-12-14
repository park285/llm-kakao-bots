-- 원자적 dequeue 연산: LPOP + Stale 체크 + SREM
-- KEYS[1]: queueKey (큐)
-- KEYS[2]: userSetKey (사용자 중복 방지 Set)
-- ARGV[1]: currentTimestamp (현재 시각 ms)
-- ARGV[2]: staleThresholdMs (Stale 임계값 ms)
-- ARGV[3]: maxIterations (루프 제한, Redis 블로킹 방지)

local queueKey = KEYS[1]
local userSetKey = KEYS[2]
local currentTimestamp = tonumber(ARGV[1])
local staleThresholdMs = tonumber(ARGV[2])
local maxIterations = tonumber(ARGV[3] or "50")  -- 기본값 50

local iterations = 0

while iterations < maxIterations do
    iterations = iterations + 1
    -- 큐에서 메시지 꺼내기
    local json = redis.call("LPOP", queueKey)
    if not json then
        return nil  -- 큐가 비어있음
    end
    
    -- JSON 파싱 (안전한 pcall 사용)
    local ok, message = pcall(cjson.decode, json)
    if not ok then
        -- JSON 파싱 실패: 손상된 데이터, 다음 메시지로
        -- userSet 정리 불가 (userId 알 수 없음)
        -- 리스크: 해당 유저는 TTL 만료 전까지 차단 (발생 빈도 매우 낮음)
    else
        local userId = message.userId
        local timestamp = tonumber(message.timestamp) or 0
        
        -- Stale 체크
        local age = currentTimestamp - timestamp
        if age > staleThresholdMs then
            -- Stale 메시지: userSet에서 제거 후 다음 메시지 처리
            if userId then redis.call("SREM", userSetKey, userId) end
        else
            -- 정상 메시지: userSet에서 제거 후 반환
            if userId then redis.call("SREM", userSetKey, userId) end
            return json
        end
    end
end

return nil  -- 루프 제한 도달 (다음 폴링 때 처리)
