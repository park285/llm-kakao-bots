// Package config: 설정 관리
package config

import (
	"os"
	"strconv"
	"strings"
	"time"
)

// Config: 애플리케이션 설정
type Config struct {
	// 서버 설정
	Port         string
	Environment  string
	LogLevel     string
	ForceHTTPS   bool
	LogDirectory string

	// TLS 설정 (HTTP/2 지원)
	TLSEnabled  bool
	TLSCertPath string
	TLSKeyPath  string

	// 인증 설정
	AdminUser            string
	AdminPassHash        string
	AdminSecretKey       string
	SessionTokenRotation bool

	// Metrics 설정
	MetricsAPIKey string

	// 외부 서비스 URL
	ValkeyURL      string
	JaegerQueryURL string
	DockerHost     string

	// 각 봇 프록시 URL
	HoloBotURL    string
	TwentyQBotURL string
	TurtleBotURL  string
	LLMServerURL  string

	// OTEL 설정
	OTELEnabled     bool
	OTELEndpoint    string
	OTELServiceName string
	OTLPInsecure    bool
}

// SessionConfig: 세션 관련 상수
var SessionConfig = struct {
	ExpiryDuration   time.Duration
	AbsoluteTimeout  time.Duration
	IdleSessionTTL   time.Duration
	GracePeriod      time.Duration
	RotationInterval time.Duration
}{
	ExpiryDuration:   30 * time.Minute,
	AbsoluteTimeout:  8 * time.Hour,
	IdleSessionTTL:   10 * time.Second,
	GracePeriod:      30 * time.Second,
	RotationInterval: 15 * time.Minute,
}

// Load: 환경 변수에서 설정 로드
func Load() *Config {
	return &Config{
		Port:         getEnv("PORT", "30090"),
		Environment:  getEnv("ENV", "production"),
		LogLevel:     getEnv("LOG_LEVEL", "info"),
		ForceHTTPS:   getEnvBool("FORCE_HTTPS", true),
		LogDirectory: getEnv("LOG_DIR", "/app/logs"),

		TLSEnabled:  getEnvBool("TLS_ENABLED", false),
		TLSCertPath: getEnv("TLS_CERT_PATH", "/certs/localhost.crt"),
		TLSKeyPath:  getEnv("TLS_KEY_PATH", "/certs/localhost.key"),

		AdminUser:            getEnv("ADMIN_USER", "admin"),
		AdminPassHash:        getEnvAny("ADMIN_PASS_HASH", "ADMIN_PASS_BCRYPT"),
		AdminSecretKey:       getEnvAny("SESSION_SECRET", "ADMIN_SECRET_KEY"),
		SessionTokenRotation: getEnvBool("SESSION_TOKEN_ROTATION", true),

		MetricsAPIKey: getEnv("METRICS_API_KEY", ""),

		ValkeyURL:      getEnv("VALKEY_URL", "valkey-cache:6379"),
		JaegerQueryURL: getEnv("JAEGER_QUERY_URL", "http://jaeger:16686"),
		DockerHost:     getEnv("DOCKER_HOST", "tcp://docker-proxy:2375"),

		HoloBotURL:    getEnv("HOLO_BOT_URL", "http://hololive-bot:30001"),
		TwentyQBotURL: getEnv("TWENTYQ_BOT_URL", "http://twentyq-bot:30081"),
		TurtleBotURL:  getEnv("TURTLE_BOT_URL", "http://turtle-soup-bot:30082"),
		LLMServerURL:  getEnv("LLM_SERVER_URL", "http://mcp-llm-server:30010"),

		OTELEnabled:     getEnvBool("OTEL_ENABLED", false),
		OTELEndpoint:    getEnv("OTEL_EXPORTER_OTLP_ENDPOINT", "jaeger:4317"),
		OTELServiceName: getEnv("OTEL_SERVICE_NAME", "admin-dashboard"),
		OTLPInsecure:    getEnvBool("OTEL_EXPORTER_OTLP_INSECURE", true),
	}
}

func getEnv(key, fallback string) string {
	if val := os.Getenv(key); val != "" {
		return val
	}
	return fallback
}

func getEnvAny(keys ...string) string {
	for _, key := range keys {
		if val := strings.TrimSpace(os.Getenv(key)); val != "" {
			return val
		}
	}
	return ""
}

func getEnvBool(key string, fallback bool) bool {
	if val := os.Getenv(key); val != "" {
		b, err := strconv.ParseBool(val)
		if err == nil {
			return b
		}
	}
	return fallback
}
