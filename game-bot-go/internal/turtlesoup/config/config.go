package config

import (
	"fmt"
	"time"

	commonconfig "github.com/park285/llm-kakao-bots/game-bot-go/internal/common/config"
)

// ServerConfig: HTTP/gRPC 서버 설정입니다.
type ServerConfig = commonconfig.ServerConfig

// ServerTuningConfig: 서버 성능 튜닝 옵션입니다.
type ServerTuningConfig = commonconfig.ServerTuningConfig

// CommandsConfig: 봇 명령어 접두사 설정입니다.
type CommandsConfig = commonconfig.CommandsConfig

// LlmConfig: LLM 서버 통신 설정 alias
type LlmConfig = commonconfig.LlmConfig

// PuzzleConfig: 퍼즐 생성 관련 설정입니다.
type PuzzleConfig struct {
	RewriteEnabled bool // Preset 퍼즐 사용 시 시나리오를 재작성할지 여부
}

// RedisConfig: Redis/Valkey 캐시 연결 설정입니다.
type RedisConfig = commonconfig.RedisConfig

// ValkeyMQConfig: Valkey 기반 메시지 큐 연결 설정입니다.
type ValkeyMQConfig = commonconfig.ValkeyMQConfig

// AccessConfig: 채팅방/사용자 접근 제어 설정입니다.
type AccessConfig = commonconfig.AccessConfig

// LogConfig: 로그 출력 설정입니다.
type LogConfig = commonconfig.LogConfig

// InjectionGuardConfig: Injection 검사 캐시 설정입니다.
type InjectionGuardConfig struct {
	CacheTTL        time.Duration // 캐시 TTL (Time To Live)
	CacheMaxEntries int           // LRU 캐시 최대 엔트리 수
}

// PostgresConfig: PostgreSQL 데이터베이스 설정
type PostgresConfig struct {
	Host       string
	Port       int
	SocketPath string
	Name       string
	User       string
	Password   string
	SSLMode    string
}

// Config: TurtleSoup 서비스 전체 설정을 통합하는 구조체입니다.
type Config struct {
	Server         ServerConfig
	ServerTuning   ServerTuningConfig
	Commands       CommandsConfig
	Llm            LlmConfig
	Puzzle         PuzzleConfig
	Redis          RedisConfig
	Valkey         ValkeyMQConfig
	Postgres       PostgresConfig
	Access         AccessConfig
	InjectionGuard InjectionGuardConfig
	Log            LogConfig
	Telemetry      commonconfig.TelemetryConfig
}

// LoadFromEnv: 환경 변수에서 전체 설정을 읽어옵니다.
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
	llmCfg, err := readLlmConfig()
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
	postgres, err := readPostgresConfig()
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
	telemetry, err := commonconfig.ReadTelemetryConfigFromEnv("turtle-soup-bot")
	if err != nil {
		return nil, fmt.Errorf("read telemetry config: %w", err)
	}

	llmCfg.EnableOTel = telemetry.Enabled

	return &Config{
		Server:         server,
		ServerTuning:   serverTuning,
		Commands:       commands,
		Llm:            llmCfg,
		Puzzle:         puzzle,
		Redis:          redis,
		Valkey:         valkey,
		Postgres:       postgres,
		Access:         access,
		InjectionGuard: injectionGuard,
		Log:            log,
		Telemetry:      telemetry,
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

func readLlmConfig() (LlmConfig, error) {
	cfg, err := commonconfig.ReadLlmConfigFromEnv()
	if err != nil {
		return LlmConfig{}, fmt.Errorf("read llm config failed: %w", err)
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
		[]string{"CACHE_SOCKET_PATH", "REDIS_SOCKET_PATH"},
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

func readPostgresConfig() (PostgresConfig, error) {
	port, err := commonconfig.IntFromEnv("DB_PORT", 5432)
	if err != nil {
		return PostgresConfig{}, fmt.Errorf("read DB_PORT failed: %w", err)
	}

	return PostgresConfig{
		Host:       commonconfig.StringFromEnv("DB_HOST", "localhost"),
		Port:       port,
		SocketPath: commonconfig.StringFromEnv("DB_SOCKET_PATH", ""),
		Name:       commonconfig.StringFromEnv("TURTLE_DB_NAME", "turtlesoup"),
		User:       commonconfig.StringFromEnv("TURTLE_DB_USER", "turtlesoup_app"),
		Password:   commonconfig.StringFromEnv("DB_PASSWORD", ""),
		SSLMode:    commonconfig.StringFromEnv("DB_SSLMODE", "disable"),
	}, nil
}
