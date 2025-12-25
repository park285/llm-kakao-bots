package admin

import (
	"fmt"
	"os"
	"strings"
	"time"
)

// Config 는 관리자 API 서버 설정이다.
type Config struct {
	Enabled bool
	Addr    string
	UseH2C  bool

	CFAccessTeamDomain string
	CFAccessAUD        string

	// InternalServiceToken 은 내부 서비스 간 인증에 사용되는 토큰이다.
	// CF Access 인증을 우회하여 Docker 네트워크 내 서비스 간 호출을 허용한다.
	InternalServiceToken string

	AllowedEmails []string
	AllowedIPs    []string

	ReadHeaderTimeout time.Duration
	ReadTimeout       time.Duration
	WriteTimeout      time.Duration
	IdleTimeout       time.Duration
	ShutdownTimeout   time.Duration
}

func envBool(key string, defaultValue bool) bool {
	raw, ok := os.LookupEnv(key)
	if !ok {
		return defaultValue
	}
	value := strings.TrimSpace(strings.ToLower(raw))
	switch value {
	case "1", "true", "yes", "y", "on":
		return true
	case "0", "false", "no", "n", "off":
		return false
	default:
		return defaultValue
	}
}

func envInt(key string, defaultValue int, minValue int) int {
	raw := strings.TrimSpace(os.Getenv(key))
	if raw == "" {
		if defaultValue < minValue {
			return minValue
		}
		return defaultValue
	}
	var parsed int
	_, err := fmt.Sscanf(raw, "%d", &parsed)
	if err != nil {
		if defaultValue < minValue {
			return minValue
		}
		return defaultValue
	}
	if parsed < minValue {
		return minValue
	}
	return parsed
}

func splitList(raw string) []string {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return nil
	}
	trimmed = strings.ReplaceAll(trimmed, ",", " ")
	return strings.Fields(trimmed)
}

func envString(key string, defaultValue string) string {
	raw := strings.TrimSpace(os.Getenv(key))
	if raw == "" {
		return defaultValue
	}
	return raw
}

// LoadAdminConfig 는 동작을 수행한다.
func LoadAdminConfig() Config {
	return Config{
		Enabled: envBool("WATCHDOG_ADMIN_ENABLED", false),
		Addr:    envString("WATCHDOG_ADMIN_ADDR", "127.0.0.1:30002"),
		UseH2C:  envBool("WATCHDOG_ADMIN_H2C", true),

		CFAccessTeamDomain: strings.TrimSpace(os.Getenv("WATCHDOG_ADMIN_CF_ACCESS_TEAM_DOMAIN")),
		CFAccessAUD:        strings.TrimSpace(os.Getenv("WATCHDOG_ADMIN_CF_ACCESS_AUD")),

		InternalServiceToken: strings.TrimSpace(os.Getenv("WATCHDOG_INTERNAL_SERVICE_TOKEN")),

		AllowedEmails: splitList(os.Getenv("WATCHDOG_ADMIN_ALLOWED_EMAILS")),
		AllowedIPs:    splitList(os.Getenv("WATCHDOG_ADMIN_ALLOWED_IPS")),

		ReadHeaderTimeout: time.Duration(envInt("WATCHDOG_ADMIN_READ_HEADER_TIMEOUT_SECONDS", 5, 0)) * time.Second,
		ReadTimeout:       time.Duration(envInt("WATCHDOG_ADMIN_READ_TIMEOUT_SECONDS", 30, 0)) * time.Second,
		WriteTimeout:      time.Duration(envInt("WATCHDOG_ADMIN_WRITE_TIMEOUT_SECONDS", 60, 0)) * time.Second,
		IdleTimeout:       time.Duration(envInt("WATCHDOG_ADMIN_IDLE_TIMEOUT_SECONDS", 120, 0)) * time.Second,
		ShutdownTimeout:   time.Duration(envInt("WATCHDOG_ADMIN_SHUTDOWN_TIMEOUT_SECONDS", 10, 0)) * time.Second,
	}
}

func normalizeCFAccessTeamDomain(raw string) string {
	value := strings.TrimSpace(raw)
	if value == "" {
		return ""
	}
	value = strings.TrimPrefix(value, "https://")
	value = strings.TrimPrefix(value, "http://")
	value = strings.TrimSuffix(value, "/")
	if !strings.Contains(value, ".") {
		value = value + ".cloudflareaccess.com"
	}
	return value
}

// ValidateForEnable 는 동작을 수행한다.
func (c Config) ValidateForEnable() error {
	if !c.Enabled {
		return nil
	}
	if strings.TrimSpace(c.Addr) == "" {
		return fmt.Errorf("WATCHDOG_ADMIN_ADDR is required")
	}
	allowlist, err := newIPAllowlist(c.AllowedIPs)
	if err != nil {
		return fmt.Errorf("WATCHDOG_ADMIN_ALLOWED_IPS is invalid: %w", err)
	}
	if allowlist == nil {
		return fmt.Errorf("WATCHDOG_ADMIN_ALLOWED_IPS is required")
	}
	return nil
}
