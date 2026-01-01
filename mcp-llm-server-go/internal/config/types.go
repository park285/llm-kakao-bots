package config

import (
	"net"
	"net/url"
	"strconv"
)

const gemini3MinTemperature = 1.0

// ThinkingConfig: Gemini thinking 레벨 설정입니다.
type ThinkingConfig struct {
	LevelDefault string
	LevelHints   string
	LevelAnswer  string
	LevelVerify  string
}

// Level: 작업 유형별 thinking 레벨을 반환합니다.
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

// GeminiConfig: Gemini 모델 설정입니다.
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
	FailoverAttempts int
}

// PrimaryKey: 기본 API 키를 반환합니다.
func (g GeminiConfig) PrimaryKey() string {
	if len(g.APIKeys) == 0 {
		return ""
	}
	return g.APIKeys[0]
}

// ModelForTask: 작업 유형별 모델을 반환합니다.
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

// TemperatureForModel: 모델별 temperature를 계산합니다.
func (g GeminiConfig) TemperatureForModel(model string) float64 {
	if isGemini3(model) {
		return max(gemini3MinTemperature, g.Temperature)
	}
	return g.Temperature
}

// SessionConfig: 세션 관련 설정입니다.
type SessionConfig struct {
	MaxSessions       int
	SessionTTLMinutes int
	HistoryMaxPairs   int
}

// SessionStoreConfig: 세션 저장소 연결 설정입니다.
type SessionStoreConfig struct {
	URL                 string
	Enabled             bool
	Required            bool
	DisableCache        bool
	ConnectMaxAttempts  int
	ConnectRetrySeconds int
}

// GuardConfig: 입력 검증 설정입니다.
type GuardConfig struct {
	Enabled         bool
	Threshold       float64
	RulepacksDir    string
	CacheMaxSize    int
	CacheTTLSeconds int
}

// LoggingConfig: 로깅 설정입니다.
type LoggingConfig struct {
	Level      string
	LogDir     string
	MaxSizeMB  int
	MaxBackups int
	MaxAgeDays int
	Compress   bool
}

// HTTPConfig: HTTP 서버 설정입니다.
type HTTPConfig struct {
	Host         string
	Port         int
	HTTP2Enabled bool
}

// GRPCConfig: gRPC 서버 설정입니다.
type GRPCConfig struct {
	Host       string
	Port       int
	Enabled    bool
	SocketPath string // UDS 경로 (비어있으면 TCP만 사용)
}

// HTTPAuthConfig: API 키 인증 설정입니다.
type HTTPAuthConfig struct {
	APIKey   string
	Required bool
}

// HTTPRateLimitConfig: 요청 제한 설정입니다.
type HTTPRateLimitConfig struct {
	RequestsPerMinute int
	CacheSize         int
	CacheTTLSeconds   int
}

// DatabaseConfig: DB 연결 및 저장 설정입니다.
type DatabaseConfig struct {
	Host                                 string
	Port                                 int
	Name                                 string
	User                                 string
	Password                             string
	MinPool                              int
	MaxPool                              int
	ConnMaxLifetimeMinutes               int
	ConnMaxIdleTimeMinutes               int
	UsageBatchEnabled                    bool
	UsageBatchFlushIntervalSeconds       int
	UsageBatchFlushTimeoutSeconds        int
	UsageBatchMaxPendingRequests         int
	UsageBatchMaxBackoffSeconds          int
	UsageBatchErrorLogMaxIntervalSeconds int
}

// DSN: DB 접속 문자열을 반환합니다.
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

// Config: 애플리케이션 전체 설정입니다.
type Config struct {
	Gemini        GeminiConfig
	Session       SessionConfig
	SessionStore  SessionStoreConfig
	Guard         GuardConfig
	Logging       LoggingConfig
	HTTP          HTTPConfig
	GRPC          GRPCConfig
	HTTPAuth      HTTPAuthConfig
	HTTPRateLimit HTTPRateLimitConfig
	Database      DatabaseConfig
}
