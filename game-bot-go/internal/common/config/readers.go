package config

import (
	"fmt"
	"strings"
	"time"
)

// ReadLlmConfigFromEnv: LLM 서버 gRPC 통신 설정을 환경 변수에서 읽어옵니다.
func ReadLlmConfigFromEnv() (LlmConfig, error) {
	llmTimeoutSeconds, err := Int64FromEnv("LLM_TIMEOUT_SECONDS", 30)
	if err != nil {
		return LlmConfig{}, fmt.Errorf("read LLM_TIMEOUT_SECONDS failed: %w", err)
	}

	llmConnectTimeoutSeconds, err := Int64FromEnv("LLM_CONNECT_TIMEOUT_SECONDS", 10)
	if err != nil {
		return LlmConfig{}, fmt.Errorf(
			"read LLM_CONNECT_TIMEOUT_SECONDS failed: %w",
			err,
		)
	}

	return LlmConfig{
		BaseURL:        StringFromEnv("LLM_BASE_URL", "grpc://localhost:40528"),
		APIKey:         StringFromEnvFirstNonEmpty([]string{"LLM_API_KEY", "HTTP_API_KEY"}, ""),
		Timeout:        time.Duration(llmTimeoutSeconds) * time.Second,
		ConnectTimeout: time.Duration(llmConnectTimeoutSeconds) * time.Second,
	}, nil
}

// ReadServerConfigFromEnv: HTTP 서버 호스트와 포트 설정을 환경 변수에서 읽어옵니다.
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

// ReadServerTuningConfigFromEnv: HTTP 서버 튜닝 설정(Timeouts, Limits)을 환경 변수에서 읽어옵니다.
func ReadServerTuningConfigFromEnv() (ServerTuningConfig, error) {
	readHeaderTimeout, err := DurationSecondsFromEnv("SERVER_READ_HEADER_TIMEOUT_SECONDS", 5)
	if err != nil {
		return ServerTuningConfig{}, fmt.Errorf(
			"read SERVER_READ_HEADER_TIMEOUT_SECONDS failed: %w",
			err,
		)
	}

	// 보안/안정성 기본값 적용함 (명시적으로 0을 주면 비활성화 가능)
	idleTimeout, err := DurationSecondsFromEnv("SERVER_IDLE_TIMEOUT_SECONDS", 90)
	if err != nil {
		return ServerTuningConfig{}, fmt.Errorf("read SERVER_IDLE_TIMEOUT_SECONDS failed: %w", err)
	}

	maxHeaderBytes, err := IntFromEnv("SERVER_MAX_HEADER_BYTES", 1<<20) // 1MiB
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

// ReadRedisConfigFromEnv: Redis(Valkey) 연결 설정을 환경 변수에서 읽어옵니다.
// 여러 환경 변수 키 중 첫 번째로 값이 존재하는 것을 사용합니다.
// socketPathKeys가 설정되면 UDS 모드로 동작하며, TCP 설정보다 우선합니다.
func ReadRedisConfigFromEnv(
	hostKeys []string,
	portKeys []string,
	passwordKeys []string,
	socketPathKeys []string,
	defaultHost string,
	defaultPort int,
	defaultPassword string,
) (RedisConfig, error) {
	port, err := IntFromEnvFirstNonEmpty(portKeys, defaultPort)
	if err != nil {
		return RedisConfig{}, fmt.Errorf("read redis port failed: %w", err)
	}

	// UDS 경로가 있으면 UDS 모드, 없으면 TCP 모드
	socketPath := StringFromEnvFirstNonEmpty(socketPathKeys, "")

	return RedisConfig{
		Host:       StringFromEnvFirstNonEmpty(hostKeys, defaultHost),
		Port:       port,
		Password:   StringFromEnvFirstNonEmpty(passwordKeys, defaultPassword),
		DB:         0,
		SocketPath: socketPath,

		DialTimeout:  10 * time.Second,
		ReadTimeout:  3 * time.Second,
		WriteTimeout: 3 * time.Second,

		PoolSize:     64,
		MinIdleConns: 10,
	}, nil
}

// ReadLogConfigFromEnv: 로그 파일 출력 설정(디렉터리, 크기, 백업 수)을 환경 변수에서 읽어옵니다.
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

// ValkeyMQConfigEnvOptions: ValkeyMQ 설정 읽기에 사용할 환경 변수 키 및 기본값 옵션입니다.
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
	BatchSizeKeys                   []string
	BlockTimeoutMillisKeys          []string
	ConcurrencyKeys                 []string
	StreamMaxLenKeys                []string

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
	DefaultBatchSize                   int64
	DefaultBlockTimeoutMillis          int64
	DefaultConcurrency                 int
	DefaultStreamMaxLen                int64
}

type valkeyMQTuning struct {
	batchSize          int64
	blockTimeoutMillis int64
	concurrency        int
	streamMaxLen       int64
}

func readValkeyMQTuning(opts ValkeyMQConfigEnvOptions) (valkeyMQTuning, error) {
	batchSize, err := Int64FromEnvFirstNonEmpty(opts.BatchSizeKeys, opts.DefaultBatchSize)
	if err != nil {
		return valkeyMQTuning{}, fmt.Errorf("read valkey mq batch size failed: %w", err)
	}

	blockTimeoutMillis, err := Int64FromEnvFirstNonEmpty(
		opts.BlockTimeoutMillisKeys,
		opts.DefaultBlockTimeoutMillis,
	)
	if err != nil {
		return valkeyMQTuning{}, fmt.Errorf("read valkey mq read timeout failed: %w", err)
	}

	concurrency, err := IntFromEnvFirstNonEmpty(opts.ConcurrencyKeys, opts.DefaultConcurrency)
	if err != nil {
		return valkeyMQTuning{}, fmt.Errorf("read valkey mq concurrency failed: %w", err)
	}

	streamMaxLen, err := Int64FromEnvFirstNonEmpty(opts.StreamMaxLenKeys, opts.DefaultStreamMaxLen)
	if err != nil {
		return valkeyMQTuning{}, fmt.Errorf("read valkey mq stream max len failed: %w", err)
	}

	if batchSize <= 0 {
		batchSize = opts.DefaultBatchSize
	}
	if blockTimeoutMillis <= 0 {
		blockTimeoutMillis = opts.DefaultBlockTimeoutMillis
	}
	if concurrency <= 0 {
		concurrency = opts.DefaultConcurrency
	}
	if streamMaxLen <= 0 {
		streamMaxLen = opts.DefaultStreamMaxLen
	}

	return valkeyMQTuning{
		batchSize:          batchSize,
		blockTimeoutMillis: blockTimeoutMillis,
		concurrency:        concurrency,
		streamMaxLen:       streamMaxLen,
	}, nil
}

// ReadValkeyMQConfigFromEnv: Valkey 기반 메시지 큐 설정을 환경 변수에서 읽어옵니다.
// 연결 정보, Consumer Group, Stream 키, 튜닝 파라미터를 포함합니다.
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

	tuning, err := readValkeyMQTuning(opts)
	if err != nil {
		return ValkeyMQConfig{}, err
	}

	resetGroupOnStartup, err := BoolFromEnvFirstNonEmpty(
		opts.ResetConsumerGroupOnStartupKeys,
		opts.DefaultResetConsumerGroupOnStartup,
	)
	if err != nil {
		return ValkeyMQConfig{}, fmt.Errorf("read valkey mq reset group on startup failed: %w", err)
	}

	timeout := time.Duration(timeoutMillis) * time.Millisecond
	blockTimeout := time.Duration(tuning.blockTimeoutMillis) * time.Millisecond

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
		BatchSize:    tuning.batchSize,
		BlockTimeout: blockTimeout,
		Concurrency:  tuning.concurrency,
		StreamMaxLen: tuning.streamMaxLen,
	}, nil
}

// AccessConfigEnvOptions: 접근 제어 설정 읽기에 사용할 환경 변수 접두사 및 기본값입니다.
type AccessConfigEnvOptions struct {
	EnvPrefix string

	DefaultEnabled     bool
	DefaultPassthrough bool

	DefaultAllowedChatIDs []string
}

// ReadAccessConfigFromEnv: 채팅방/사용자 접근 제어 설정을 환경 변수에서 읽어옵니다.
// 허용/차단 목록과 Passthrough 모드를 설정합니다.
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
