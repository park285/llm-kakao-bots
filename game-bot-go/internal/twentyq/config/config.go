package config

import (
	"fmt"

	commonconfig "github.com/park285/llm-kakao-bots/game-bot-go/internal/common/config"
)

// ServerConfig: HTTP 서버 설정 (포트 등) alias
type ServerConfig = commonconfig.ServerConfig

// ServerTuningConfig: 서버 튜닝 설정 (Timeouts, Limits 등) alias
type ServerTuningConfig = commonconfig.ServerTuningConfig

// CommandsConfig: 명령어 접두사 등 명령어 처리 관련 설정 alias
type CommandsConfig = commonconfig.CommandsConfig

// LlmConfig: LLM 서버 통신 설정 alias
type LlmConfig = commonconfig.LlmConfig

// RedisConfig: Redis 연결 설정 (캐시용) alias
type RedisConfig = commonconfig.RedisConfig

// ValkeyMQConfig: Valkey 기반 메시지 큐 설정 alias
type ValkeyMQConfig = commonconfig.ValkeyMQConfig

// PostgresConfig: PostgreSQL 데이터베이스 설정
type PostgresConfig struct {
	Host       string
	Port       int
	SocketPath string // UDS 경로 (비어있으면 TCP 사용)
	Name       string
	User       string
	Password   string
	SSLMode    string
}

// AccessConfig: 접근 제어 설정 (화이트리스트/블랙리스트) alias
type AccessConfig = commonconfig.AccessConfig

// LogConfig: 로깅 설정 (레벨, 포맷 등) alias
type LogConfig = commonconfig.LogConfig

// AdminConfig: 관리자 권한 설정
type AdminConfig struct {
	UserIDs []string
}

// StatsConfig: 통계 기록 관련 설정
type StatsConfig struct {
	WorkerCount        int
	QueueSize          int
	DropLogOnQueueFull bool
}

// UsageConfig: 사용량/비용 표시를 위한 설정입니다.
type UsageConfig struct {
	ExchangeRateAPIURL string
}

// Config: 전체 애플리케이션 설정 구조체
type Config struct {
	Server       ServerConfig
	ServerTuning ServerTuningConfig
	Commands     CommandsConfig
	Llm          LlmConfig
	Redis        RedisConfig
	Valkey       ValkeyMQConfig
	Postgres     PostgresConfig
	Access       AccessConfig
	Admin        AdminConfig
	Log          LogConfig
	Stats        StatsConfig
	Usage        UsageConfig
	Telemetry    commonconfig.TelemetryConfig // OpenTelemetry 분산 추적
}

// LoadFromEnv: 환경 변수로부터 전체 애플리케이션 설정을 로드합니다.
func LoadFromEnv() (*Config, error) {
	server, err := readServerConfig()
	if err != nil {
		return nil, err
	}
	serverTuning, err := readServerTuningConfig()
	if err != nil {
		return nil, err
	}
	commands, err := readCommandsConfig()
	if err != nil {
		return nil, err
	}
	llmCfg, err := readLlmConfig()
	if err != nil {
		return nil, err
	}
	redisCfg, err := readRedisConfig()
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
	admin := readAdminConfig()
	log, err := readLogConfig()
	if err != nil {
		return nil, err
	}
	stats, err := readStatsConfig()
	if err != nil {
		return nil, err
	}
	usage := readUsageConfig()
	telemetry, err := commonconfig.ReadTelemetryConfigFromEnv("twentyq-bot")
	if err != nil {
		return nil, fmt.Errorf("read telemetry config: %w", err)
	}

	// Telemetry 연동: gRPC 클라이언트도 OTel trace context 전파 활성화
	llmCfg.EnableOTel = telemetry.Enabled

	return &Config{
		Server:       server,
		ServerTuning: serverTuning,
		Commands:     commands,
		Llm:          llmCfg,
		Redis:        redisCfg,
		Valkey:       valkey,
		Postgres:     postgres,
		Access:       access,
		Admin:        admin,
		Log:          log,
		Stats:        stats,
		Usage:        usage,
		Telemetry:    telemetry,
	}, nil
}

func readUsageConfig() UsageConfig {
	apiURL := commonconfig.StringFromEnvFirstNonEmpty(
		[]string{"TWENTYQ_EXCHANGE_RATE_API_URL", "EXCHANGE_RATE_API_URL"},
		DefaultExchangeRateAPIURL,
	)
	return UsageConfig{ExchangeRateAPIURL: apiURL}
}

func readStatsConfig() (StatsConfig, error) {
	workerCount, err := commonconfig.IntFromEnv("STATS_WORKER_COUNT", 2)
	if err != nil {
		return StatsConfig{}, fmt.Errorf("read STATS_WORKER_COUNT failed: %w", err)
	}
	queueSize, err := commonconfig.IntFromEnv("STATS_QUEUE_SIZE", 100)
	if err != nil {
		return StatsConfig{}, fmt.Errorf("read STATS_QUEUE_SIZE failed: %w", err)
	}
	dropLog, err := commonconfig.BoolFromEnv("STATS_DROP_LOG_ON_QUEUE_FULL", false)
	if err != nil {
		return StatsConfig{}, fmt.Errorf("read STATS_DROP_LOG_ON_QUEUE_FULL failed: %w", err)
	}

	return StatsConfig{
		WorkerCount:        workerCount,
		QueueSize:          queueSize,
		DropLogOnQueueFull: dropLog,
	}, nil
}

func readServerConfig() (ServerConfig, error) {
	cfg, err := commonconfig.ReadServerConfigFromEnv(40258)
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

func readCommandsConfig() (CommandsConfig, error) {
	prefix := commonconfig.StringFromEnvFirstNonEmpty([]string{"TWENTYQ_COMMAND_PREFIX", "COMMAND_PREFIX"}, "/20q")
	return CommandsConfig{Prefix: prefix}, nil
}

func readLlmConfig() (LlmConfig, error) {
	cfg, err := commonconfig.ReadLlmConfigFromEnv()
	if err != nil {
		return LlmConfig{}, fmt.Errorf("read llm config failed: %w", err)
	}
	return cfg, nil
}

func readRedisConfig() (RedisConfig, error) {
	cfg, err := commonconfig.ReadRedisConfigFromEnv(
		[]string{"CACHE_HOST", "REDIS_HOST"},
		[]string{"CACHE_PORT", "REDIS_PORT"},
		[]string{"CACHE_PASSWORD", "REDIS_PASSWORD"},
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
		HostKeys:     []string{"MQ_HOST", "VALKEY_MQ_HOST"},
		PortKeys:     []string{"MQ_PORT", "VALKEY_MQ_PORT"},
		PasswordKeys: []string{"MQ_PASSWORD", "VALKEY_MQ_PASSWORD"},

		TimeoutMillisKeys: []string{"MQ_TIMEOUT", "VALKEY_MQ_TIMEOUT"},
		PoolSizeKeys:      []string{"MQ_CONNECTION_POOL_SIZE", "VALKEY_MQ_CONNECTION_POOL_SIZE"},
		MinIdleKeys:       []string{"MQ_CONNECTION_MIN_IDLE_SIZE", "VALKEY_MQ_CONNECTION_MIN_IDLE_SIZE"},

		ConsumerGroupKeys: []string{"MQ_CONSUMER_GROUP", "VALKEY_MQ_CONSUMER_GROUP"},
		ConsumerNameKeys:  []string{"MQ_CONSUMER_NAME", "VALKEY_MQ_CONSUMER_NAME"},
		ResetConsumerGroupOnStartupKeys: []string{
			"MQ_RESET_CONSUMER_GROUP_ON_STARTUP",
			"VALKEY_MQ_RESET_CONSUMER_GROUP_ON_STARTUP",
		},
		StreamKeyKeys:      []string{"MQ_STREAM_KEY", "VALKEY_MQ_STREAM_KEY"},
		ReplyStreamKeyKeys: []string{"MQ_REPLY_STREAM_KEY", "VALKEY_MQ_REPLY_STREAM_KEY"},
		BatchSizeKeys:      []string{"MQ_BATCH_SIZE", "VALKEY_MQ_BATCH_SIZE"},
		BlockTimeoutMillisKeys: []string{
			"MQ_READ_TIMEOUT_MS",
			"VALKEY_MQ_READ_TIMEOUT_MS",
		},
		ConcurrencyKeys:  []string{"MQ_CONCURRENCY", "VALKEY_MQ_CONCURRENCY"},
		StreamMaxLenKeys: []string{"MQ_STREAM_MAX_LEN", "VALKEY_MQ_STREAM_MAX_LEN"},

		DefaultHost:          "localhost",
		DefaultPort:          1833,
		DefaultPassword:      "",
		DefaultTimeoutMillis: 5000,
		DefaultPoolSize:      64,
		DefaultMinIdle:       10,

		DefaultConsumerGroup:               "20q-bot-group",
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

func readPostgresConfig() (PostgresConfig, error) {
	port, err := commonconfig.IntFromEnv("DB_PORT", 5432)
	if err != nil {
		return PostgresConfig{}, fmt.Errorf("read DB_PORT failed: %w", err)
	}

	return PostgresConfig{
		Host:       commonconfig.StringFromEnv("DB_HOST", "localhost"),
		Port:       port,
		SocketPath: commonconfig.StringFromEnv("DB_SOCKET_PATH", ""),
		Name:       commonconfig.StringFromEnv("DB_NAME", "twentyq"),
		User:       commonconfig.StringFromEnv("DB_USER", "twentyq_app"),
		Password:   commonconfig.StringFromEnv("DB_PASSWORD", ""),
		SSLMode:    commonconfig.StringFromEnv("DB_SSLMODE", "disable"),
	}, nil
}

func readAccessConfig() (AccessConfig, error) {
	cfg, err := commonconfig.ReadAccessConfigFromEnv(commonconfig.AccessConfigEnvOptions{
		EnvPrefix:             "TWENTYQ_",
		DefaultEnabled:        false,
		DefaultPassthrough:    false,
		DefaultAllowedChatIDs: nil,
	})
	if err != nil {
		return AccessConfig{}, fmt.Errorf("read access config failed: %w", err)
	}
	return cfg, nil
}

func readAdminConfig() AdminConfig {
	return AdminConfig{
		UserIDs: commonconfig.StringListFromEnv("ADMIN_USER_IDS", nil),
	}
}

func readLogConfig() (LogConfig, error) {
	cfg, err := commonconfig.ReadLogConfigFromEnv()
	if err != nil {
		return LogConfig{}, fmt.Errorf("read log config failed: %w", err)
	}
	return cfg, nil
}
