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

// LlmNamespace: LLM 네임스페이스 상수
const (
	LlmNamespace = "twentyq"
)

// MaxHintsTotal: 게임당 최대 힌트 횟수
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
	RedisProcessingTTLSeconds = 200
)

// MQMaxQueueIterations 는 twentyq 전용 상수이다.
const (
	MQMaxQueueIterations = 100
)

// DefaultInboundStreamKey 는 twentyq 인바운드 스트림 키이다.
const (
	DefaultInboundStreamKey = "kakao:20q"
)

// AllCategories 게임에서 사용하는 모든 카테고리 목록.
var AllCategories = []string{
	"organism",
	"food",
	"object",
	"place",
	"concept",
	"movie",
	"idiom_proverb",
}
