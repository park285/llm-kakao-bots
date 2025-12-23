package config

// LlmNamespace 는 상수다.
const (
	LlmNamespace = "twentyq"
)

// MaxHintsTotal 는 상수다.
const (
	MaxHintsTotal = 1
)

// RedisKeyPrefix 는 Redis 키 상수 목록이다.
const (
	RedisKeyPrefix        = "20q"
	RedisKeySessionPrefix = RedisKeyPrefix + ":riddle:session"
	RedisKeyHistoryPrefix = RedisKeyPrefix + ":history"
	RedisKeyCategory      = RedisKeyPrefix + ":category"

	RedisKeyHints        = RedisKeyPrefix + ":hints"
	RedisKeyPlayers      = RedisKeyPrefix + ":players"
	RedisKeyWrongGuesses = RedisKeyPrefix + ":wrongGuesses"
	RedisKeyTopics       = RedisKeyPrefix + ":topics"

	RedisKeyVotePrefix    = RedisKeyPrefix + ":surrender:vote"
	RedisKeyPendingPrefix = RedisKeyPrefix + ":pending-messages"
	RedisKeyLockPrefix    = RedisKeyPrefix + ":lock"
)

// RedisSessionTTLSeconds 는 Redis TTL 상수 목록이다.
const (
	RedisSessionTTLSeconds    = 12 * 60 * 60
	RedisLockTTLSeconds       = 300
	RedisVoteTTLSeconds       = 120
	RedisProcessingTTLSeconds = 200
	RedisQueueTTLSeconds      = 300
	RedisMaxQueueSize         = 5
	RedisStaleThresholdMS     = 3600_000
)

// AITimeoutSeconds 는 상수다.
const (
	AITimeoutSeconds = 60
)

// KakaoMessageMaxLength 는 상수다.
const (
	KakaoMessageMaxLength = 500
)

// MQBatchSize 는 MQ 처리 상수 목록이다.
const (
	MQBatchSize               = 5
	MQReadTimeoutMS           = 5000
	MQMaxQueueIterations      = 100
	MQStreamMaxLen            = 1000
	QueueMaxDequeueIterations = 50
)

// DefaultInboundStreamKey 는 기본 스트림 키 상수 목록이다.
const (
	DefaultInboundStreamKey  = "kakao:20q"
	DefaultOutboundStreamKey = "kakao:bot:reply"
)

// AllCategories 게임에서 사용하는 모든 카테고리 목록.
// topic_selector.go와 session_store.go에서 공유.
var AllCategories = []string{
	"organism",
	"food",
	"object",
	"place",
	"concept",
	"movie",
	"idiom_proverb",
}
