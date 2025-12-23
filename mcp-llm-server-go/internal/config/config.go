package config

import (
	"cmp"
	"errors"
	"fmt"
	"log/slog"
	"math"
	"net"
	"net/url"
	"os"
	"strconv"
	"strings"
	"sync"

	"github.com/joho/godotenv"
)

const gemini3MinTemperature = 1.0

var (
	configOnce  sync.Once
	configValue *Config
)

// ThinkingConfig 는 Gemini thinking 레벨 설정이다.
type ThinkingConfig struct {
	LevelDefault string
	LevelHints   string
	LevelAnswer  string
	LevelVerify  string
}

// Level 는 작업 유형별 thinking 레벨을 반환한다.
func (t ThinkingConfig) Level(task string) string {
	switch task {
	case "hints":
		return t.LevelHints
	case "answer":
		return t.LevelAnswer
	case "verify":
		return t.LevelVerify
	default:
		return t.LevelDefault
	}
}

// GeminiConfig 는 Gemini 모델 설정이다.
type GeminiConfig struct {
	APIKeys          []string
	DefaultModel     string
	HintsModel       string
	AnswerModel      string
	VerifyModel      string
	Temperature      float64
	MaxOutputTokens  int
	Thinking         ThinkingConfig
	MaxRetries       int
	TimeoutSeconds   int
	ModelCacheSize   int
	FailoverAttempts int
}

// PrimaryKey 는 기본 API 키를 반환한다.
func (g GeminiConfig) PrimaryKey() string {
	if len(g.APIKeys) == 0 {
		return ""
	}
	return g.APIKeys[0]
}

// ModelForTask 는 작업 유형별 모델을 반환한다.
func (g GeminiConfig) ModelForTask(task string) string {
	switch task {
	case "hints":
		if g.HintsModel != "" {
			return g.HintsModel
		}
	case "answer":
		if g.AnswerModel != "" {
			return g.AnswerModel
		}
	case "verify":
		if g.VerifyModel != "" {
			return g.VerifyModel
		}
	}
	return g.DefaultModel
}

// TemperatureForModel 는 모델별 temperature 를 계산한다.
func (g GeminiConfig) TemperatureForModel(model string) float64 {
	if isGemini3(model) {
		if math.IsNaN(g.Temperature) || math.IsInf(g.Temperature, 0) {
			return gemini3MinTemperature
		}
		return math.Max(gemini3MinTemperature, g.Temperature)
	}
	return g.Temperature
}

// SessionConfig 는 세션 관련 설정이다.
type SessionConfig struct {
	MaxSessions       int
	SessionTTLMinutes int
	HistoryMaxPairs   int
}

// SessionStoreConfig 는 세션 저장소 연결 설정이다.
type SessionStoreConfig struct {
	URL                 string
	Enabled             bool
	Required            bool
	DisableCache        bool
	ConnectMaxAttempts  int
	ConnectRetrySeconds int
}

// GuardConfig 는 입력 검증 설정이다.
type GuardConfig struct {
	Enabled         bool
	Threshold       float64
	RulepacksDir    string
	CacheMaxSize    int
	CacheTTLSeconds int
}

// LoggingConfig 는 로깅 설정이다.
type LoggingConfig struct {
	Level      string
	LogDir     string
	MaxSizeMB  int
	MaxBackups int
	MaxAgeDays int
	Compress   bool
}

// HTTPConfig 는 HTTP 서버 설정이다.
type HTTPConfig struct {
	Host         string
	Port         int
	HTTP2Enabled bool
}

// HTTPAuthConfig 는 API 키 인증 설정이다.
type HTTPAuthConfig struct {
	APIKey string
}

// HTTPRateLimitConfig 는 요청 제한 설정이다.
type HTTPRateLimitConfig struct {
	RequestsPerMinute int
	CacheSize         int
	CacheTTLSeconds   int
}

// DatabaseConfig 는 DB 연결 및 저장 설정이다.
type DatabaseConfig struct {
	Host                                 string
	Port                                 int
	Name                                 string
	User                                 string
	Password                             string
	MinPool                              int
	MaxPool                              int
	UsageBatchEnabled                    bool
	UsageBatchFlushIntervalSeconds       int
	UsageBatchMaxPendingRequests         int
	UsageBatchMaxBackoffSeconds          int
	UsageBatchErrorLogMaxIntervalSeconds int
}

// DSN 은 DB 접속 문자열을 반환한다.
func (d DatabaseConfig) DSN() string {
	host := net.JoinHostPort(d.Host, strconv.Itoa(d.Port))
	u := &url.URL{
		Scheme: "postgresql",
		Host:   host,
		Path:   "/" + d.Name,
	}
	if d.Password == "" {
		u.User = url.User(d.User)
	} else {
		u.User = url.UserPassword(d.User, d.Password)
	}
	return u.String()
}

// Config 는 애플리케이션 전체 설정이다.
type Config struct {
	Gemini        GeminiConfig
	Session       SessionConfig
	SessionStore  SessionStoreConfig
	Guard         GuardConfig
	Logging       LoggingConfig
	HTTP          HTTPConfig
	HTTPAuth      HTTPAuthConfig
	HTTPRateLimit HTTPRateLimitConfig
	Database      DatabaseConfig
}

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
			UsageBatchEnabled:                    getEnvBool("DB_USAGE_BATCH_ENABLED", false),
			UsageBatchFlushIntervalSeconds:       max(1, getEnvNonNegativeInt("DB_USAGE_BATCH_FLUSH_INTERVAL_SECONDS", 1)),
			UsageBatchMaxPendingRequests:         max(1, getEnvNonNegativeInt("DB_USAGE_BATCH_MAX_PENDING_REQUESTS", 50)),
			UsageBatchMaxBackoffSeconds:          getEnvNonNegativeInt("DB_USAGE_BATCH_MAX_BACKOFF_SECONDS", 60),
			UsageBatchErrorLogMaxIntervalSeconds: getEnvNonNegativeInt("DB_USAGE_BATCH_ERROR_LOG_MAX_INTERVAL_SECONDS", 60),
		},
	}
}

func parseAPIKeys() []string {
	keysValue := strings.TrimSpace(os.Getenv("GOOGLE_API_KEYS"))
	if keysValue != "" {
		return splitKeys(keysValue)
	}
	key := strings.TrimSpace(os.Getenv("GOOGLE_API_KEY"))
	if key == "" {
		return nil
	}
	return []string{key}
}

func splitKeys(value string) []string {
	items := strings.FieldsFunc(value, func(r rune) bool {
		return r == ',' || r == ' ' || r == '\t' || r == '\n' || r == '\r'
	})
	result := make([]string, 0, len(items))
	for _, item := range items {
		trimmed := strings.TrimSpace(item)
		if trimmed != "" {
			result = append(result, trimmed)
		}
	}
	return result
}

func isGemini3(model string) bool {
	return strings.Contains(strings.ToLower(model), "gemini-3")
}

func getEnvString(key string, def string) string {
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		return def
	}
	return value
}

func getEnvInt(key string, def int) int {
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		return def
	}
	parsed, err := strconv.Atoi(value)
	if err != nil {
		return def
	}
	return parsed
}

func getEnvNonNegativeInt(key string, def int) int {
	value := getEnvInt(key, def)
	if value < 0 {
		return 0
	}
	return value
}

func getEnvFloat(key string, def float64) float64 {
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		return def
	}
	parsed, err := strconv.ParseFloat(value, 64)
	if err != nil {
		return def
	}
	return parsed
}

func getEnvBool(key string, def bool) bool {
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		return def
	}
	value = strings.ToLower(value)
	return value == "true" || value == "1" || value == "yes" || value == "y"
}

func maskSecret(value string) string {
	if value == "" {
		return "<missing>"
	}
	if len(value) <= 4 {
		return strings.Repeat("*", len(value))
	}
	return value[:2] + "***" + value[len(value)-2:]
}

func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

// cmp.Or 는 첫 번째 non-zero 값을 반환한다. (Go 1.22+)
func orValue[T comparable](values ...T) T {
	return cmp.Or(values...)
}
