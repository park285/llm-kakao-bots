package config

import (
	commonconfig "github.com/park285/llm-kakao-bots/game-bot-go/internal/common/config"
)

// 공통 상수 re-export
const (
	KakaoMessageMaxLength     = commonconfig.KakaoMessageMaxLength
	AITimeoutSeconds          = commonconfig.AITimeoutSeconds
	MQBatchSize               = commonconfig.MQBatchSize
	MQReadTimeoutMS           = commonconfig.MQReadTimeoutMS
	MQStreamMaxLen            = commonconfig.MQStreamMaxLen
	QueueMaxDequeueIterations = commonconfig.QueueMaxDequeueIterations
	RedisVoteTTLSeconds       = commonconfig.RedisVoteTTLSeconds
	RedisQueueTTLSeconds      = commonconfig.RedisQueueTTLSeconds
	RedisMaxQueueSize         = commonconfig.RedisMaxQueueSize
	RedisStaleThresholdMS     = commonconfig.RedisStaleThresholdMS
	DefaultOutboundStreamKey  = commonconfig.DefaultOutboundStreamKey
)

// LLM 네임스페이스 상수.
const (
	// LlmNamespace: LLM 네임스페이스 상수
	LlmNamespace = "turtle-soup"
)

// 힌트 제한 상수.
const (
	// GameMaxHints: 게임당 최대 힌트 횟수
	GameMaxHints = 3
)

// 검증 관련 상수.
const (
	// ValidationMinQuestionLength: 질문 최소 길이
	ValidationMinQuestionLength = 2
	ValidationMaxQuestionLength = 200
	ValidationMinAnswerLength   = 1
	ValidationMaxAnswerLength   = 500
)

// Redis 키 상수.
const (
	// RedisKeyPrefix: Redis 키 접두사
	RedisKeyPrefix        = "turtle"
	RedisKeySessionPrefix = RedisKeyPrefix + ":session"
	RedisKeyLockPrefix    = RedisKeyPrefix + ":lock"
	RedisKeyVotePrefix    = RedisKeyPrefix + ":vote"
	RedisKeyPendingPrefix = RedisKeyPrefix + ":pending"
	RedisKeyProcessing    = RedisKeyPrefix + ":processing"
	RedisKeyPuzzleGlobal  = RedisKeyPrefix + ":puzzle:global"
	RedisKeyPuzzleChat    = RedisKeyPrefix + ":puzzle:chat"
)

// Redis TTL 상수 (도메인 전용).
const (
	// RedisSessionTTLSeconds: 세션 TTL (24시간)
	RedisSessionTTLSeconds    = 86400
	RedisLockTTLSeconds       = 120
	RedisLockTimeoutSeconds   = 60
	RedisProcessingTTLSeconds = 120
)

// 퍼즐 난이도 상수.
const (
	// PuzzleMinDifficulty: 퍼즐 최소 난이도
	PuzzleMinDifficulty     = 1
	PuzzleMaxDifficulty     = 5
	PuzzleDefaultDifficulty = 3
)

// 퍼즐 중복 방지 상수.
const (
	// PuzzleDedupMaxGenerationRetries: 퍼즐 중복 생성 시 최대 재시도 횟수
	PuzzleDedupMaxGenerationRetries = 3
	PuzzleDedupGlobalTTLSeconds     = 7 * 24 * 3600
	PuzzleDedupChatTTLSeconds       = 3 * 24 * 3600
)

// 인젝션 가드 캐시 상수.
const (
	// InjectionGuardCacheTTLSeconds: 인젝션 가드 캐시 TTL(초)
	InjectionGuardCacheTTLSeconds = 600
	InjectionGuardCacheMaxEntries = 10000
)

// MQ 상수 (도메인 전용).
const (
	// MQMaxQueueIterations: 큐 처리 최대 반복 횟수
	MQMaxQueueIterations = 10
)

// 기본 스트림 키 상수.
const (
	// DefaultInboundStreamKey: 기본 인바운드 스트림 키
	DefaultInboundStreamKey = "kakao:turtle-soup"
)
