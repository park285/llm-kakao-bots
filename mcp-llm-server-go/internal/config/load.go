package config

import (
	"errors"
	"fmt"
	"log/slog"
	"sync"

	"github.com/joho/godotenv"
)

var (
	configOnce  sync.Once
	configValue *Config
)

// Load 는 환경 변수 기반 설정을 로드한다.
func Load() *Config {
	configOnce.Do(func() {
		_ = godotenv.Load()
		configValue = buildConfig()
	})
	return configValue
}

// ProvideConfig 는 설정을 로드하고 검증한다.
func ProvideConfig() (*Config, error) {
	cfg := Load()
	if cfg == nil {
		return nil, errors.New("config not initialized")
	}
	if err := cfg.Validate(); err != nil {
		return nil, err
	}
	return cfg, nil
}

// Validate 는 설정 유효성을 검사한다.
func (c *Config) Validate() error {
	if c == nil {
		return errors.New("config is nil")
	}
	models := []string{
		c.Gemini.DefaultModel,
		c.Gemini.HintsModel,
		c.Gemini.AnswerModel,
		c.Gemini.VerifyModel,
	}
	for _, model := range models {
		if model == "" {
			continue
		}
		if !isGemini3(model) {
			return fmt.Errorf("gemini 3 only: model=%s", model)
		}
	}
	return nil
}

// LogEnvStatus 는 환경 설정 상태를 로그로 남긴다.
func LogEnvStatus(cfg *Config, logger *slog.Logger) {
	if logger == nil || cfg == nil {
		return
	}

	envFilePresent := fileExists(".env")
	primaryKey := maskSecret(cfg.Gemini.PrimaryKey())
	logger.Debug(
		"env_status",
		"env_file", envFilePresent,
		"gemini_keys", len(cfg.Gemini.APIKeys),
		"primary_key", primaryKey,
		"model", cfg.Gemini.DefaultModel,
		"timeout", cfg.Gemini.TimeoutSeconds,
		"session_store_url", cfg.SessionStore.URL,
		"db_host", cfg.Database.Host,
		"db_name", cfg.Database.Name,
		"session_ttl", cfg.Session.SessionTTLMinutes,
		"history_pairs", cfg.Session.HistoryMaxPairs,
	)

	if len(cfg.Gemini.APIKeys) == 0 {
		logger.Error("env_missing_google_api_key")
	}
}

func buildConfig() *Config {
	return &Config{
		Gemini: GeminiConfig{
			APIKeys:         parseAPIKeys(),
			DefaultModel:    getEnvString("GEMINI_MODEL", "gemini-3-flash-preview"),
			HintsModel:      getEnvString("GEMINI_HINTS_MODEL", ""),
			AnswerModel:     getEnvString("GEMINI_ANSWER_MODEL", ""),
			VerifyModel:     getEnvString("GEMINI_VERIFY_MODEL", ""),
			Temperature:     getEnvFloat("GEMINI_TEMPERATURE", 0.7),
			MaxOutputTokens: getEnvInt("GEMINI_MAX_TOKENS", 8192),
			Thinking: ThinkingConfig{
				LevelDefault: getEnvString("GEMINI_THINKING_LEVEL", "low"),
				LevelHints:   getEnvString("GEMINI_THINKING_LEVEL_HINTS", "low"),
				LevelAnswer:  getEnvString("GEMINI_THINKING_LEVEL_ANSWER", "low"),
				LevelVerify:  getEnvString("GEMINI_THINKING_LEVEL_VERIFY", "low"),
			},
			MaxRetries:       max(1, getEnvInt("GEMINI_MAX_RETRIES", 6)),
			TimeoutSeconds:   getEnvInt("GEMINI_TIMEOUT", 60),
			ModelCacheSize:   getEnvInt("GEMINI_MODEL_CACHE_SIZE", 20),
			FailoverAttempts: max(1, getEnvInt("GEMINI_FAILOVER_ATTEMPTS", 2)),
		},
		Session: SessionConfig{
			MaxSessions:       getEnvInt("MAX_SESSIONS", 50),
			SessionTTLMinutes: getEnvInt("SESSION_TTL_MINUTES", 1440),
			HistoryMaxPairs:   getEnvNonNegativeInt("SESSION_HISTORY_MAX_PAIRS", 10),
		},
		SessionStore: SessionStoreConfig{
			URL:                 getEnvString("SESSION_STORE_URL", "redis://localhost:6379"),
			Enabled:             getEnvBool("SESSION_STORE_ENABLED", true),
			Required:            getEnvBool("SESSION_STORE_REQUIRED", false),
			DisableCache:        getEnvBool("SESSION_STORE_DISABLE_CACHE", false),
			ConnectMaxAttempts:  max(1, getEnvNonNegativeInt("SESSION_STORE_CONNECT_MAX_ATTEMPTS", 6)),
			ConnectRetrySeconds: getEnvNonNegativeInt("SESSION_STORE_CONNECT_RETRY_SECONDS", 5),
		},
		Guard: GuardConfig{
			Enabled:         getEnvBool("GUARD_ENABLED", true),
			Threshold:       getEnvFloat("GUARD_THRESHOLD", 0.85),
			RulepacksDir:    getEnvString("RULEPACKS_DIR", "rulepacks"),
			CacheMaxSize:    getEnvInt("GUARD_CACHE_SIZE", 10000),
			CacheTTLSeconds: getEnvInt("GUARD_CACHE_TTL", 3600),
		},
		Logging: LoggingConfig{
			Level:      getEnvString("LOG_LEVEL", "info"),
			LogDir:     getEnvString("LOG_DIR", ""),
			MaxSizeMB:  getEnvInt("LOG_FILE_MAX_SIZE_MB", 1),
			MaxBackups: getEnvInt("LOG_FILE_MAX_BACKUPS", 30),
			MaxAgeDays: getEnvInt("LOG_FILE_MAX_AGE_DAYS", 7),
			Compress:   getEnvBool("LOG_FILE_COMPRESS", true),
		},
		HTTP: HTTPConfig{
			Host:         getEnvString("HTTP_HOST", "127.0.0.1"),
			Port:         getEnvInt("HTTP_PORT", 40527),
			HTTP2Enabled: getEnvBool("HTTP2_ENABLED", true),
		},
		HTTPAuth: HTTPAuthConfig{
			APIKey: getEnvString("HTTP_API_KEY", ""),
		},
		HTTPRateLimit: HTTPRateLimitConfig{
			RequestsPerMinute: getEnvNonNegativeInt("HTTP_RATE_LIMIT_RPM", 0),
			CacheSize:         max(1, getEnvNonNegativeInt("HTTP_RATE_LIMIT_CACHE_SIZE", 10000)),
			CacheTTLSeconds:   max(1, getEnvNonNegativeInt("HTTP_RATE_LIMIT_CACHE_TTL_SECONDS", 120)),
		},
		Database: DatabaseConfig{
			Host:                                 getEnvString("DB_HOST", "localhost"),
			Port:                                 getEnvInt("DB_PORT", 5432),
			Name:                                 getEnvString("DB_NAME", "twentyq"),
			User:                                 getEnvString("DB_USER", "twentyq"),
			Password:                             getEnvString("DB_PASSWORD", ""),
			MinPool:                              getEnvInt("DB_MIN_POOL", 1),
			MaxPool:                              getEnvInt("DB_MAX_POOL", 5),
			ConnMaxLifetimeMinutes:               getEnvNonNegativeInt("DB_CONN_MAX_LIFETIME_MINUTES", 60),
			ConnMaxIdleTimeMinutes:               getEnvNonNegativeInt("DB_CONN_MAX_IDLE_TIME_MINUTES", 10),
			UsageBatchEnabled:                    getEnvBool("DB_USAGE_BATCH_ENABLED", false),
			UsageBatchFlushIntervalSeconds:       max(1, getEnvNonNegativeInt("DB_USAGE_BATCH_FLUSH_INTERVAL_SECONDS", 1)),
			UsageBatchFlushTimeoutSeconds:        max(1, getEnvNonNegativeInt("DB_USAGE_BATCH_FLUSH_TIMEOUT_SECONDS", 5)),
			UsageBatchMaxPendingRequests:         max(1, getEnvNonNegativeInt("DB_USAGE_BATCH_MAX_PENDING_REQUESTS", 50)),
			UsageBatchMaxBackoffSeconds:          getEnvNonNegativeInt("DB_USAGE_BATCH_MAX_BACKOFF_SECONDS", 60),
			UsageBatchErrorLogMaxIntervalSeconds: getEnvNonNegativeInt("DB_USAGE_BATCH_ERROR_LOG_MAX_INTERVAL_SECONDS", 60),
		},
	}
}
