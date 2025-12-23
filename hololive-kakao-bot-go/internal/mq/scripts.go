package mq

// Lua scripts for atomic MQ operations

const (
	// processWithIdempotency: 멱등성 확인 + 처리 마킹 + ACK (올인원)
	// KEYS[1]: idempotency key
	// KEYS[2]: stream key
	// ARGV[1]: consumer group
	// ARGV[2]: message ID
	// ARGV[3]: TTL in seconds
	// Returns: 1 if should process, 0 if already processed (and ACK'd)
	luaProcessWithIdempotency = `
		local idempotency_key = KEYS[1]
		local stream_key = KEYS[2]
		local group = ARGV[1]
		local msg_id = ARGV[2]
		local ttl = tonumber(ARGV[3])

		-- 이미 처리된 메시지인지 확인
		if redis.call('EXISTS', idempotency_key) == 1 then
			-- 이미 처리됨, ACK만 하고 0 반환
			redis.call('XACK', stream_key, group, msg_id)
			return 0
		end

		-- 처리 시작 마킹 (처리 전에 설정하여 동시 처리 방지)
		redis.call('SETEX', idempotency_key, ttl, 'processing')
		return 1  -- 처리 진행
	`

	// completeProcessing: 처리 완료 및 ACK
	// KEYS[1]: idempotency key
	// KEYS[2]: stream key
	// ARGV[1]: consumer group
	// ARGV[2]: message ID
	// ARGV[3]: TTL in seconds
	luaCompleteProcessing = `
		local idempotency_key = KEYS[1]
		local stream_key = KEYS[2]
		local group = ARGV[1]
		local msg_id = ARGV[2]
		local ttl = tonumber(ARGV[3])

		-- 처리 완료로 상태 변경
		redis.call('SETEX', idempotency_key, ttl, 'completed')

		-- ACK
		redis.call('XACK', stream_key, group, msg_id)
		return 1
	`
)
