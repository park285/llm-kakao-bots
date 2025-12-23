package config

import (
	"fmt"

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

// RedisConfig 는 타입이다.
type RedisConfig = commonconfig.RedisConfig

// ValkeyMQConfig 는 타입이다.
type ValkeyMQConfig = commonconfig.ValkeyMQConfig

// PostgresConfig 는 타입이다.
type PostgresConfig struct {
	Host     string
	Port     int
	Name     string
	User     string
	Password string
	SSLMode  string
}

// AccessConfig 는 타입이다.
type AccessConfig = commonconfig.AccessConfig

// LogConfig 는 타입이다.
type LogConfig = commonconfig.LogConfig

// AdminConfig 는 타입이다.
type AdminConfig struct {
	UserIDs []string
}

// Config 는 타입이다.
type Config struct {
	Server       ServerConfig
	ServerTuning ServerTuningConfig
	Commands     CommandsConfig
	LlmRest      LlmRestConfig
	Redis        RedisConfig
	Valkey       ValkeyMQConfig
	Postgres     PostgresConfig
	Access       AccessConfig
	Admin        AdminConfig
	Log          LogConfig
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
	commands, err := readCommandsConfig()
	if err != nil {
		return nil, err
	}
	llmRest, err := readLlmRestConfig()
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

	return &Config{
		Server:       server,
		ServerTuning: serverTuning,
		Commands:     commands,
		LlmRest:      llmRest,
		Redis:        redisCfg,
		Valkey:       valkey,
		Postgres:     postgres,
		Access:       access,
		Admin:        admin,
		Log:          log,
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

func readLlmRestConfig() (LlmRestConfig, error) {
	cfg, err := commonconfig.ReadLlmRestConfigFromEnv()
	if err != nil {
		return LlmRestConfig{}, fmt.Errorf("read llm rest config failed: %w", err)
	}
	return cfg, nil
}

func readRedisConfig() (RedisConfig, error) {
	cfg, err := commonconfig.ReadRedisConfigFromEnv(
		[]string{"CACHE_HOST", "REDIS_HOST"},
		[]string{"CACHE_PORT", "REDIS_PORT"},
		[]string{"CACHE_PASSWORD", "REDIS_PASSWORD"},
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
		Host:     commonconfig.StringFromEnv("DB_HOST", "localhost"),
		Port:     port,
		Name:     commonconfig.StringFromEnv("DB_NAME", "twentyq"),
		User:     commonconfig.StringFromEnv("DB_USER", "twentyq_app"),
		Password: commonconfig.StringFromEnv("DB_PASSWORD", ""),
		SSLMode:  commonconfig.StringFromEnv("DB_SSLMODE", "disable"),
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
