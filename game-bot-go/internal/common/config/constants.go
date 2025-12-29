package config

// 카카오톡 메시지 관련 상수.
const (
	// KakaoMessageMaxLength: 카카오톡 메시지 최대 길이 제한
	KakaoMessageMaxLength = 500
)

// AI 관련 상수.
const (
	// AITimeoutSeconds: AI 응답 대기 타임아웃(초)
	AITimeoutSeconds = 60
)

// MQ 공통 상수.
const (
	// MQBatchSize: 메시지 큐 배치 크기
	MQBatchSize = 5
	// MQReadTimeoutMS: 메시지 큐 읽기 타임아웃(ms)
	MQReadTimeoutMS = 5000
	// MQConsumerConcurrency: 메시지 큐 소비 동시성
	MQConsumerConcurrency = 5
	// MQStreamMaxLen: 스트림 최대 길이
	MQStreamMaxLen = 1000
	// QueueMaxDequeueIterations: 큐에서 최대 디큐 반복 횟수
	QueueMaxDequeueIterations = 50
	// QueueDequeueBatchSize: 큐 디큐 배치 크기
	QueueDequeueBatchSize = MQBatchSize
)

// Redis 공통 TTL 상수.
const (
	// RedisVoteTTLSeconds: 투표 TTL (2분)
	RedisVoteTTLSeconds = 120
	// RedisQueueTTLSeconds: 큐 TTL (5분)
	RedisQueueTTLSeconds = 300
	// RedisMaxQueueSize: 큐 최대 크기
	RedisMaxQueueSize = 5
	// RedisStaleThresholdMS: 오래된 메시지 임계값 (1시간)
	RedisStaleThresholdMS = 3600_000
)

// 스트림 키 상수.
const (
	// DefaultOutboundStreamKey: 봇 응답 스트림 키
	DefaultOutboundStreamKey = "kakao:bot:reply"
)
