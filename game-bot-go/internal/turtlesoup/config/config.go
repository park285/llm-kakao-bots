package config

import (
	"fmt"
	"time"

	commonconfig "github.com/park285/llm-kakao-bots/game-bot-go/internal/common/config"
)

// ServerConfig 는 타입이다.
type ServerConfig = commonconfig.ServerConfig

// ServerTuningConfig 는 타입이다.
type ServerTuningConfig = commonconfig.ServerTuningConfig

// CommandsConfig 는 타입이다.
type CommandsConfig = commonconfig.CommandsConfig

// LlmRestConfig 는 타입이다.
type LlmRestConfig = commonconfig.LlmRestConfig

// PuzzleConfig 는 타입이다.
type PuzzleConfig struct {
	RewriteEnabled bool
}

// RedisConfig 는 타입이다.
type RedisConfig = commonconfig.RedisConfig

// ValkeyMQConfig 는 타입이다.
type ValkeyMQConfig = commonconfig.ValkeyMQConfig

// AccessConfig 는 타입이다.
type AccessConfig = commonconfig.AccessConfig

// LogConfig 는 타입이다.
type LogConfig = commonconfig.LogConfig

// InjectionGuardConfig 는 타입이다.
type InjectionGuardConfig struct {
	CacheTTL        time.Duration
	CacheMaxEntries int
}

// Config 는 타입이다.
type Config struct {
	Server         ServerConfig
	ServerTuning   ServerTuningConfig
	Commands       CommandsConfig
	LlmRest        LlmRestConfig
	Puzzle         PuzzleConfig
	Redis          RedisConfig
	Valkey         ValkeyMQConfig
	Access         AccessConfig
	InjectionGuard InjectionGuardConfig
	Log            LogConfig
}

// LoadFromEnv 는 동작을 수행한다.
func LoadFromEnv() (*Config, error) {
	server, err := readServerConfig()
	if err != nil {
		return nil, err
	}
	serverTuning, err := readServerTuningConfig()
	if err != nil {
		return nil, err
	}
	commands := readCommandsConfig()
	llmRest, err := readLlmRestConfig()
	if err != nil {
		return nil, err
	}
	puzzle, err := readPuzzleConfig()
	if err != nil {
		return nil, err
	}
	redis, err := readRedisConfig()
	if err != nil {
		return nil, err
	}
	valkey, err := readValkeyMQConfig()
	if err != nil {
		return nil, err
	}
	access, err := readAccessConfig()
	if err != nil {
		return nil, err
	}
	injectionGuard, err := readInjectionGuardConfig()
	if err != nil {
		return nil, err
	}
	log, err := readLogConfig()
	if err != nil {
		return nil, err
	}

	return &Config{
		Server:         server,
		ServerTuning:   serverTuning,
		Commands:       commands,
		LlmRest:        llmRest,
		Puzzle:         puzzle,
		Redis:          redis,
		Valkey:         valkey,
		Access:         access,
		InjectionGuard: injectionGuard,
		Log:            log,
	}, nil
}

func readServerConfig() (ServerConfig, error) {
	cfg, err := commonconfig.ReadServerConfigFromEnv(40257)
	if err != nil {
		return ServerConfig{}, fmt.Errorf("read server config failed: %w", err)
	}
	return cfg, nil
}

func readServerTuningConfig() (ServerTuningConfig, error) {
	cfg, err := commonconfig.ReadServerTuningConfigFromEnv()
	if err != nil {
		return ServerTuningConfig{}, fmt.Errorf("read server tuning config failed: %w", err)
	}
	return cfg, nil
}

func readCommandsConfig() CommandsConfig {
	return CommandsConfig{
		Prefix: commonconfig.StringFromEnv("TURTLESOUP_COMMAND_PREFIX", "/스프"),
	}
}

func readLlmRestConfig() (LlmRestConfig, error) {
	cfg, err := commonconfig.ReadLlmRestConfigFromEnv()
	if err != nil {
		return LlmRestConfig{}, fmt.Errorf("read llm rest config failed: %w", err)
	}
	return cfg, nil
}

func readPuzzleConfig() (PuzzleConfig, error) {
	puzzleRewriteEnabled, err := commonconfig.BoolFromEnv("PUZZLE_REWRITE_ENABLED", true)
	if err != nil {
		return PuzzleConfig{}, fmt.Errorf("read PUZZLE_REWRITE_ENABLED failed: %w", err)
	}
	return PuzzleConfig{RewriteEnabled: puzzleRewriteEnabled}, nil
}

func readRedisConfig() (RedisConfig, error) {
	cfg, err := commonconfig.ReadRedisConfigFromEnv(
		[]string{"REDIS_HOST", "CACHE_HOST"},
		[]string{"REDIS_PORT", "CACHE_PORT"},
		[]string{"REDIS_PASSWORD", "CACHE_PASSWORD"},
		"localhost",
		6379,
		"",
	)
	if err != nil {
		return RedisConfig{}, fmt.Errorf("read redis config failed: %w", err)
	}
	return cfg, nil
}

func readValkeyMQConfig() (ValkeyMQConfig, error) {
	cfg, err := commonconfig.ReadValkeyMQConfigFromEnv(commonconfig.ValkeyMQConfigEnvOptions{
		HostKeys:     []string{"VALKEY_MQ_HOST", "MQ_HOST"},
		PortKeys:     []string{"VALKEY_MQ_PORT", "MQ_PORT"},
		PasswordKeys: []string{"VALKEY_MQ_PASSWORD", "MQ_PASSWORD"},

		TimeoutMillisKeys: []string{"VALKEY_MQ_TIMEOUT", "MQ_TIMEOUT"},
		PoolSizeKeys:      []string{"VALKEY_MQ_CONNECTION_POOL_SIZE", "MQ_CONNECTION_POOL_SIZE"},
		MinIdleKeys:       []string{"VALKEY_MQ_CONNECTION_MIN_IDLE_SIZE", "MQ_CONNECTION_MIN_IDLE_SIZE"},

		ConsumerGroupKeys: []string{"VALKEY_MQ_CONSUMER_GROUP", "MQ_CONSUMER_GROUP"},
		ConsumerNameKeys:  []string{"VALKEY_MQ_CONSUMER_NAME", "MQ_CONSUMER_NAME"},
		ResetConsumerGroupOnStartupKeys: []string{
			"VALKEY_MQ_RESET_CONSUMER_GROUP_ON_STARTUP",
			"MQ_RESET_CONSUMER_GROUP_ON_STARTUP",
		},
		StreamKeyKeys:      []string{"VALKEY_MQ_STREAM_KEY", "MQ_STREAM_KEY"},
		ReplyStreamKeyKeys: []string{"VALKEY_MQ_REPLY_STREAM_KEY", "MQ_REPLY_STREAM_KEY"},
		BatchSizeKeys:      []string{"VALKEY_MQ_BATCH_SIZE", "MQ_BATCH_SIZE"},
		BlockTimeoutMillisKeys: []string{
			"VALKEY_MQ_READ_TIMEOUT_MS",
			"MQ_READ_TIMEOUT_MS",
		},
		ConcurrencyKeys:  []string{"VALKEY_MQ_CONCURRENCY", "MQ_CONCURRENCY"},
		StreamMaxLenKeys: []string{"VALKEY_MQ_STREAM_MAX_LEN", "MQ_STREAM_MAX_LEN"},

		DefaultHost:          "localhost",
		DefaultPort:          1833,
		DefaultPassword:      "",
		DefaultTimeoutMillis: 5000,
		DefaultPoolSize:      64,
		DefaultMinIdle:       10,

		DefaultConsumerGroup:               "turtle-soup-bot-group",
		DefaultConsumerName:                "consumer-1",
		DefaultResetConsumerGroupOnStartup: false,
		DefaultStreamKey:                   DefaultInboundStreamKey,
		DefaultReplyStreamKey:              DefaultOutboundStreamKey,
		DefaultBatchSize:                   commonconfig.MQBatchSize,
		DefaultBlockTimeoutMillis:          commonconfig.MQReadTimeoutMS,
		DefaultConcurrency:                 commonconfig.MQConsumerConcurrency,
		DefaultStreamMaxLen:                commonconfig.MQStreamMaxLen,
	})
	if err != nil {
		return ValkeyMQConfig{}, fmt.Errorf("read valkey mq config failed: %w", err)
	}
	return cfg, nil
}

func readAccessConfig() (AccessConfig, error) {
	cfg, err := commonconfig.ReadAccessConfigFromEnv(commonconfig.AccessConfigEnvOptions{
		EnvPrefix:             "TURTLESOUP_",
		DefaultEnabled:        true,
		DefaultPassthrough:    false,
		DefaultAllowedChatIDs: []string{"267947734"},
	})
	if err != nil {
		return AccessConfig{}, fmt.Errorf("read access config failed: %w", err)
	}
	return cfg, nil
}

func readInjectionGuardConfig() (InjectionGuardConfig, error) {
	ttlSeconds, err := commonconfig.Int64FromEnvFirstNonEmpty(
		[]string{
			"TURTLESOUP_INJECTION_GUARD_CACHE_TTL_SECONDS",
			"INJECTION_GUARD_CACHE_TTL_SECONDS",
		},
		int64(InjectionGuardCacheTTLSeconds),
	)
	if err != nil {
		return InjectionGuardConfig{}, fmt.Errorf("read injection guard cache ttl failed: %w", err)
	}
	if ttlSeconds < 0 {
		return InjectionGuardConfig{}, fmt.Errorf("invalid injection guard cache ttl seconds: %d", ttlSeconds)
	}

	maxEntries, err := commonconfig.IntFromEnvFirstNonEmpty(
		[]string{
			"TURTLESOUP_INJECTION_GUARD_CACHE_MAX_ENTRIES",
			"INJECTION_GUARD_CACHE_MAX_ENTRIES",
		},
		InjectionGuardCacheMaxEntries,
	)
	if err != nil {
		return InjectionGuardConfig{}, fmt.Errorf("read injection guard cache max entries failed: %w", err)
	}
	if maxEntries < 0 {
		return InjectionGuardConfig{}, fmt.Errorf("invalid injection guard cache max entries: %d", maxEntries)
	}

	return InjectionGuardConfig{
		CacheTTL:        time.Duration(ttlSeconds) * time.Second,
		CacheMaxEntries: maxEntries,
	}, nil
}

func readLogConfig() (LogConfig, error) {
	cfg, err := commonconfig.ReadLogConfigFromEnv()
	if err != nil {
		return LogConfig{}, fmt.Errorf("read log config failed: %w", err)
	}
	return cfg, nil
}
