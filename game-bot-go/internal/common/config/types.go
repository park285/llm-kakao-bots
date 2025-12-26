package config

import "time"

// ServerConfig 는 타입이다.
type ServerConfig struct {
	Host string
	Port int
}

// CommandsConfig 는 타입이다.
type CommandsConfig struct {
	Prefix string
}

// LlmRestConfig 는 타입이다.
type LlmRestConfig struct {
	BaseURL          string
	APIKey           string
	Timeout          time.Duration
	ConnectTimeout   time.Duration
	HTTP2Enabled     bool
	RetryMaxAttempts int
	RetryDelay       time.Duration
}

// RedisConfig 는 타입이다.
type RedisConfig struct {
	Host     string
	Port     int
	Password string
	DB       int

	DialTimeout  time.Duration
	ReadTimeout  time.Duration
	WriteTimeout time.Duration

	PoolSize     int
	MinIdleConns int
}

// ValkeyMQConfig 는 타입이다.
type ValkeyMQConfig struct {
	Host     string
	Port     int
	Password string
	DB       int

	Timeout                     time.Duration
	DialTimeout                 time.Duration
	PoolSize                    int
	MinIdleConns                int
	ConsumerGroup               string
	ConsumerName                string
	ResetConsumerGroupOnStartup bool
	StreamKey                   string
	ReplyStreamKey              string

	BatchSize    int64
	BlockTimeout time.Duration
	Concurrency  int
	StreamMaxLen int64
}

// AccessConfig 는 타입이다.
type AccessConfig struct {
	Enabled        bool
	AllowedChatIDs []string
	BlockedChatIDs []string
	BlockedUserIDs []string
	Passthrough    bool
}

// LogConfig 는 타입이다.
type LogConfig struct {
	Dir string

	MaxSizeMB  int
	MaxBackups int
	MaxAgeDays int
	Compress   bool
}

// ServerTuningConfig 는 타입이다.
type ServerTuningConfig struct {
	ReadHeaderTimeout time.Duration
	IdleTimeout       time.Duration
	MaxHeaderBytes    int
}
