package config

import (
	"fmt"
	"strings"
	"time"
)

// ReadLlmRestConfigFromEnv 는 동작을 수행한다.
func ReadLlmRestConfigFromEnv() (LlmRestConfig, error) {
	llmTimeoutSeconds, err := Int64FromEnv("LLM_REST_TIMEOUT_SECONDS", 30)
	if err != nil {
		return LlmRestConfig{}, fmt.Errorf("read LLM_REST_TIMEOUT_SECONDS failed: %w", err)
	}

	llmConnectTimeoutSeconds, err := Int64FromEnv("LLM_REST_CONNECT_TIMEOUT_SECONDS", 10)
	if err != nil {
		return LlmRestConfig{}, fmt.Errorf(
			"read LLM_REST_CONNECT_TIMEOUT_SECONDS failed: %w",
			err,
		)
	}

	llmHTTP2Enabled, err := BoolFromEnv("LLM_REST_HTTP2_ENABLED", true)
	if err != nil {
		return LlmRestConfig{}, fmt.Errorf("read LLM_REST_HTTP2_ENABLED failed: %w", err)
	}

	llmRetryMaxAttempts, err := IntFromEnv("LLM_REST_RETRY_MAX_ATTEMPTS", 2)
	if err != nil {
		return LlmRestConfig{}, fmt.Errorf("read LLM_REST_RETRY_MAX_ATTEMPTS failed: %w", err)
	}

	llmRetryDelay, err := DurationMillisFromEnv("LLM_REST_RETRY_DELAY_MS", 200)
	if err != nil {
		return LlmRestConfig{}, fmt.Errorf("read LLM_REST_RETRY_DELAY_MS failed: %w", err)
	}

	return LlmRestConfig{
		BaseURL:          StringFromEnv("LLM_REST_BASE_URL", "http://localhost:40527"),
		APIKey:           StringFromEnvFirstNonEmpty([]string{"LLM_REST_API_KEY", "HTTP_API_KEY"}, ""),
		Timeout:          time.Duration(llmTimeoutSeconds) * time.Second,
		ConnectTimeout:   time.Duration(llmConnectTimeoutSeconds) * time.Second,
		HTTP2Enabled:     llmHTTP2Enabled,
		RetryMaxAttempts: llmRetryMaxAttempts,
		RetryDelay:       llmRetryDelay,
	}, nil
}

// ReadServerConfigFromEnv 는 동작을 수행한다.
func ReadServerConfigFromEnv(defaultPort int) (ServerConfig, error) {
	serverPort, err := IntFromEnv("SERVER_PORT", defaultPort)
	if err != nil {
		return ServerConfig{}, fmt.Errorf("read SERVER_PORT failed: %w", err)
	}

	return ServerConfig{
		Host: StringFromEnv("SERVER_HOST", "0.0.0.0"),
		Port: serverPort,
	}, nil
}

// ReadServerTuningConfigFromEnv 는 동작을 수행한다.
func ReadServerTuningConfigFromEnv() (ServerTuningConfig, error) {
	readHeaderTimeout, err := DurationSecondsFromEnv("SERVER_READ_HEADER_TIMEOUT_SECONDS", 5)
	if err != nil {
		return ServerTuningConfig{}, fmt.Errorf(
			"read SERVER_READ_HEADER_TIMEOUT_SECONDS failed: %w",
			err,
		)
	}

	idleTimeout, err := DurationSecondsFromEnv("SERVER_IDLE_TIMEOUT_SECONDS", 0)
	if err != nil {
		return ServerTuningConfig{}, fmt.Errorf("read SERVER_IDLE_TIMEOUT_SECONDS failed: %w", err)
	}

	maxHeaderBytes, err := IntFromEnv("SERVER_MAX_HEADER_BYTES", 0)
	if err != nil {
		return ServerTuningConfig{}, fmt.Errorf("read SERVER_MAX_HEADER_BYTES failed: %w", err)
	}
	if maxHeaderBytes < 0 {
		return ServerTuningConfig{}, fmt.Errorf("invalid SERVER_MAX_HEADER_BYTES: %d", maxHeaderBytes)
	}

	return ServerTuningConfig{
		ReadHeaderTimeout: readHeaderTimeout,
		IdleTimeout:       idleTimeout,
		MaxHeaderBytes:    maxHeaderBytes,
	}, nil
}

// ReadRedisConfigFromEnv 는 동작을 수행한다.
func ReadRedisConfigFromEnv(
	hostKeys []string,
	portKeys []string,
	passwordKeys []string,
	defaultHost string,
	defaultPort int,
	defaultPassword string,
) (RedisConfig, error) {
	port, err := IntFromEnvFirstNonEmpty(portKeys, defaultPort)
	if err != nil {
		return RedisConfig{}, fmt.Errorf("read redis port failed: %w", err)
	}

	return RedisConfig{
		Host:     StringFromEnvFirstNonEmpty(hostKeys, defaultHost),
		Port:     port,
		Password: StringFromEnvFirstNonEmpty(passwordKeys, defaultPassword),
		DB:       0,

		DialTimeout:  10 * time.Second,
		ReadTimeout:  3 * time.Second,
		WriteTimeout: 3 * time.Second,

		PoolSize:     64,
		MinIdleConns: 10,
	}, nil
}

// ReadLogConfigFromEnv 는 동작을 수행한다.
func ReadLogConfigFromEnv() (LogConfig, error) {
	dir := StringFromEnv("LOG_DIR", "")
	if strings.TrimSpace(dir) == "" {
		return LogConfig{Dir: ""}, nil
	}

	maxSizeMB, err := IntFromEnv("LOG_FILE_MAX_SIZE_MB", 1)
	if err != nil {
		return LogConfig{}, fmt.Errorf("read LOG_FILE_MAX_SIZE_MB failed: %w", err)
	}
	if maxSizeMB <= 0 {
		return LogConfig{}, fmt.Errorf("invalid LOG_FILE_MAX_SIZE_MB: %d", maxSizeMB)
	}

	maxBackups, err := IntFromEnv("LOG_FILE_MAX_BACKUPS", 30)
	if err != nil {
		return LogConfig{}, fmt.Errorf("read LOG_FILE_MAX_BACKUPS failed: %w", err)
	}
	if maxBackups <= 0 {
		return LogConfig{}, fmt.Errorf("invalid LOG_FILE_MAX_BACKUPS: %d", maxBackups)
	}

	maxAgeDays, err := IntFromEnv("LOG_FILE_MAX_AGE_DAYS", 7)
	if err != nil {
		return LogConfig{}, fmt.Errorf("read LOG_FILE_MAX_AGE_DAYS failed: %w", err)
	}
	if maxAgeDays <= 0 {
		return LogConfig{}, fmt.Errorf("invalid LOG_FILE_MAX_AGE_DAYS: %d", maxAgeDays)
	}

	compress, err := BoolFromEnv("LOG_FILE_COMPRESS", true)
	if err != nil {
		return LogConfig{}, fmt.Errorf("read LOG_FILE_COMPRESS failed: %w", err)
	}

	return LogConfig{
		Dir:        dir,
		MaxSizeMB:  maxSizeMB,
		MaxBackups: maxBackups,
		MaxAgeDays: maxAgeDays,
		Compress:   compress,
	}, nil
}

// ValkeyMQConfigEnvOptions 는 타입이다.
type ValkeyMQConfigEnvOptions struct {
	HostKeys     []string
	PortKeys     []string
	PasswordKeys []string

	TimeoutMillisKeys []string
	PoolSizeKeys      []string
	MinIdleKeys       []string

	ConsumerGroupKeys               []string
	ConsumerNameKeys                []string
	ResetConsumerGroupOnStartupKeys []string
	StreamKeyKeys                   []string
	ReplyStreamKeyKeys              []string

	DefaultHost     string
	DefaultPort     int
	DefaultPassword string

	DefaultTimeoutMillis int64
	DefaultPoolSize      int
	DefaultMinIdle       int

	DefaultConsumerGroup               string
	DefaultConsumerName                string
	DefaultResetConsumerGroupOnStartup bool
	DefaultStreamKey                   string
	DefaultReplyStreamKey              string
}

// ReadValkeyMQConfigFromEnv 는 동작을 수행한다.
func ReadValkeyMQConfigFromEnv(opts ValkeyMQConfigEnvOptions) (ValkeyMQConfig, error) {
	port, err := IntFromEnvFirstNonEmpty(opts.PortKeys, opts.DefaultPort)
	if err != nil {
		return ValkeyMQConfig{}, fmt.Errorf("read valkey mq port failed: %w", err)
	}

	timeoutMillis, err := Int64FromEnvFirstNonEmpty(
		opts.TimeoutMillisKeys,
		opts.DefaultTimeoutMillis,
	)
	if err != nil {
		return ValkeyMQConfig{}, fmt.Errorf("read valkey mq timeout failed: %w", err)
	}

	poolSize, err := IntFromEnvFirstNonEmpty(opts.PoolSizeKeys, opts.DefaultPoolSize)
	if err != nil {
		return ValkeyMQConfig{}, fmt.Errorf("read valkey mq pool size failed: %w", err)
	}

	minIdle, err := IntFromEnvFirstNonEmpty(opts.MinIdleKeys, opts.DefaultMinIdle)
	if err != nil {
		return ValkeyMQConfig{}, fmt.Errorf(
			"read valkey mq min idle conns failed: %w",
			err,
		)
	}

	resetGroupOnStartup, err := BoolFromEnvFirstNonEmpty(
		opts.ResetConsumerGroupOnStartupKeys,
		opts.DefaultResetConsumerGroupOnStartup,
	)
	if err != nil {
		return ValkeyMQConfig{}, fmt.Errorf("read valkey mq reset group on startup failed: %w", err)
	}

	timeout := time.Duration(timeoutMillis) * time.Millisecond

	return ValkeyMQConfig{
		Host:     StringFromEnvFirstNonEmpty(opts.HostKeys, opts.DefaultHost),
		Port:     port,
		Password: StringFromEnvFirstNonEmpty(opts.PasswordKeys, opts.DefaultPassword),
		DB:       0,

		Timeout:      timeout,
		DialTimeout:  timeout,
		PoolSize:     poolSize,
		MinIdleConns: minIdle,
		ConsumerGroup: StringFromEnvFirstNonEmpty(
			opts.ConsumerGroupKeys,
			opts.DefaultConsumerGroup,
		),
		ConsumerName: StringFromEnvFirstNonEmpty(
			opts.ConsumerNameKeys,
			opts.DefaultConsumerName,
		),
		ResetConsumerGroupOnStartup: resetGroupOnStartup,
		StreamKey: StringFromEnvFirstNonEmpty(
			opts.StreamKeyKeys,
			opts.DefaultStreamKey,
		),
		ReplyStreamKey: StringFromEnvFirstNonEmpty(
			opts.ReplyStreamKeyKeys,
			opts.DefaultReplyStreamKey,
		),
	}, nil
}

// AccessConfigEnvOptions 는 타입이다.
type AccessConfigEnvOptions struct {
	EnvPrefix string

	DefaultEnabled     bool
	DefaultPassthrough bool

	DefaultAllowedChatIDs []string
}

// ReadAccessConfigFromEnv 는 동작을 수행한다.
func ReadAccessConfigFromEnv(opts AccessConfigEnvOptions) (AccessConfig, error) {
	prefix := opts.EnvPrefix

	enabled, err := BoolFromEnvFirstNonEmpty([]string{
		prefix + "ACCESS_ENABLED",
		"ACCESS_ENABLED",
	}, opts.DefaultEnabled)
	if err != nil {
		return AccessConfig{}, fmt.Errorf("read ACCESS_ENABLED failed: %w", err)
	}

	passthrough, err := BoolFromEnvFirstNonEmpty([]string{
		prefix + "ACCESS_PASSTHROUGH",
		"ACCESS_PASSTHROUGH",
	}, opts.DefaultPassthrough)
	if err != nil {
		return AccessConfig{}, fmt.Errorf("read ACCESS_PASSTHROUGH failed: %w", err)
	}

	allowedChatIDs := StringListFromEnvFirstNonEmpty([]string{
		prefix + "ALLOWED_CHAT_IDS",
		prefix + "ACCESS_ALLOWED_CHAT_IDS",
		"ALLOWED_CHAT_IDS",
		"ACCESS_ALLOWED_CHAT_IDS",
	}, opts.DefaultAllowedChatIDs)

	blockedChatIDs := StringListFromEnvFirstNonEmpty([]string{
		prefix + "BLOCKED_CHAT_IDS",
		prefix + "ACCESS_BLOCKED_CHAT_IDS",
		"BLOCKED_CHAT_IDS",
		"ACCESS_BLOCKED_CHAT_IDS",
	}, nil)

	blockedUserIDs := StringListFromEnvFirstNonEmpty([]string{
		prefix + "BLOCKED_USER_IDS",
		prefix + "ACCESS_BLOCKED_USER_IDS",
		"BLOCKED_USER_IDS",
		"ACCESS_BLOCKED_USER_IDS",
	}, nil)

	return AccessConfig{
		Enabled:        enabled,
		AllowedChatIDs: allowedChatIDs,
		BlockedChatIDs: blockedChatIDs,
		BlockedUserIDs: blockedUserIDs,
		Passthrough:    passthrough,
	}, nil
}
