package config

// LLM 네임스페이스 상수.
const (
	// LlmNamespace 는 상수다.
	LlmNamespace = "turtle-soup"
)

// 힌트 제한 상수.
const (
	// GameMaxHints 는 상수다.
	GameMaxHints = 3
)

// 검증 관련 상수.
const (
	// ValidationMinQuestionLength 는 상수다.
	ValidationMinQuestionLength = 2
	ValidationMaxQuestionLength = 200
	ValidationMinAnswerLength   = 1
	ValidationMaxAnswerLength   = 500
	KakaoMessageMaxLength       = 500
)

// Redis 키 상수.
const (
	// RedisKeyPrefix 는 상수다.
	RedisKeyPrefix        = "turtle"
	RedisKeySessionPrefix = RedisKeyPrefix + ":session"
	RedisKeyLockPrefix    = RedisKeyPrefix + ":lock"
	RedisKeyVotePrefix    = RedisKeyPrefix + ":vote"
	RedisKeyPendingPrefix = RedisKeyPrefix + ":pending"
	RedisKeyProcessing    = RedisKeyPrefix + ":processing"
	RedisKeyPuzzleGlobal  = RedisKeyPrefix + ":puzzle:global"
	RedisKeyPuzzleChat    = RedisKeyPrefix + ":puzzle:chat"
)

// Redis TTL 상수.
const (
	// RedisSessionTTLSeconds 는 상수다.
	RedisSessionTTLSeconds    = 86400
	RedisLockTTLSeconds       = 120
	RedisLockTimeoutSeconds   = 60
	RedisVoteTTLSeconds       = 120
	RedisProcessingTTLSeconds = 120
	RedisQueueTTLSeconds      = 300
	RedisMaxQueueSize         = 5
	RedisStaleThresholdMS     = 3600_000
)

// 퍼즐 난이도 상수.
const (
	// PuzzleMinDifficulty 는 상수다.
	PuzzleMinDifficulty     = 1
	PuzzleMaxDifficulty     = 5
	PuzzleDefaultDifficulty = 3
)

// 퍼즐 중복 방지 상수.
const (
	// PuzzleDedupMaxGenerationRetries 는 상수다.
	PuzzleDedupMaxGenerationRetries = 3
	PuzzleDedupGlobalTTLSeconds     = 7 * 24 * 3600
	PuzzleDedupChatTTLSeconds       = 3 * 24 * 3600
)

// AI 타임아웃 상수.
const (
	// AITimeoutSeconds 는 상수다.
	AITimeoutSeconds = 60
)

// 인젝션 가드 캐시 상수.
const (
	// InjectionGuardCacheTTLSeconds 는 상수다.
	InjectionGuardCacheTTLSeconds = 600
	InjectionGuardCacheMaxEntries = 10000
)

// MQ 상수.
const (
	// MQBatchSize 는 상수다.
	MQBatchSize               = 5
	MQReadTimeoutMS           = 5000
	MQMaxQueueIterations      = 10
	MQStreamMaxLen            = 1000
	QueueMaxDequeueIterations = 50
)

// 기본 스트림 키 상수.
const (
	// DefaultInboundStreamKey 는 상수다.
	DefaultInboundStreamKey  = "kakao:turtle-soup"
	DefaultOutboundStreamKey = "kakao:bot:reply"
)
