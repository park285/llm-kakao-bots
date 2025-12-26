package admin

import (
	"fmt"
	"net/netip"
	"os"
	"strings"
	"time"
)

// SkipAuthMode 는 skip_auth 쿼리 파라미터의 허용 범위를 정의한다.
type SkipAuthMode string

const (
	// SkipAuthDisabled skip_auth 쿼리 파라미터를 완전히 비활성화한다.
	SkipAuthDisabled SkipAuthMode = "disabled"
	// SkipAuthTokenOnly InternalServiceToken 헤더가 있는 경우에만 CF Access를 우회한다.
	SkipAuthTokenOnly SkipAuthMode = "token_only"
	// SkipAuthDockerNetwork Docker 네트워크 (172.x.x.x, 10.x.x.x) 에서만 skip_auth를 허용한다.
	SkipAuthDockerNetwork SkipAuthMode = "docker_network"
	// SkipAuthLocalOnly localhost (127.0.0.1, ::1) 에서만 skip_auth를 허용한다.
	SkipAuthLocalOnly SkipAuthMode = "local_only"
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

	// SkipAuthMode 는 skip_auth 쿼리 파라미터의 허용 범위를 정의한다.
	// disabled: skip_auth 완전 비활성화 (프로덕션 권장)
	// token_only: X-Internal-Service-Token 헤더 필수
	// docker_network: Docker 네트워크 IP만 허용
	// local_only: localhost만 허용
	SkipAuthMode SkipAuthMode

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

func parseSkipAuthMode(raw string) SkipAuthMode {
	value := strings.TrimSpace(strings.ToLower(raw))
	switch value {
	case "disabled", "off", "false", "0":
		return SkipAuthDisabled
	case "token_only", "token":
		return SkipAuthTokenOnly
	case "docker_network", "docker":
		return SkipAuthDockerNetwork
	case "local_only", "local", "localhost":
		return SkipAuthLocalOnly
	default:
		// 기본값: token_only (보안과 호환성 균형)
		return SkipAuthTokenOnly
	}
}

// IsDockerNetworkIP 는 주어진 IP가 Docker 네트워크 대역인지 확인한다.
// Docker 기본 네트워크: 172.16.0.0/12, 10.0.0.0/8, 192.168.0.0/16
func IsDockerNetworkIP(addr netip.Addr) bool {
	if !addr.IsValid() {
		return false
	}
	// Docker 기본 브릿지 네트워크는 보통 172.17.0.0/16 이상을 사용
	// 사설 IP 대역 전체를 Docker 네트워크로 간주
	dockerRanges := []netip.Prefix{
		netip.MustParsePrefix("172.16.0.0/12"),  // Docker 기본
		netip.MustParsePrefix("10.0.0.0/8"),     // Docker Swarm/Compose
		netip.MustParsePrefix("192.168.0.0/16"), // 일부 Docker 설정
	}
	for _, prefix := range dockerRanges {
		if prefix.Contains(addr) {
			return true
		}
	}
	return false
}

// IsLocalhostIP 는 주어진 IP가 localhost인지 확인한다.
func IsLocalhostIP(addr netip.Addr) bool {
	if !addr.IsValid() {
		return false
	}
	return addr.IsLoopback()
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
		SkipAuthMode:         parseSkipAuthMode(os.Getenv("WATCHDOG_SKIP_AUTH_MODE")),

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
		value += ".cloudflareaccess.com"
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
