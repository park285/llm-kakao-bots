package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/joho/godotenv"

	"github.com/kapu/hololive-kakao-bot-go/internal/constants"
	"github.com/kapu/hololive-kakao-bot-go/internal/util"
)

const maxHolodexAPIKeySlots = 5

// Config: 홀로라이브 봇의 전체 동작에 필요한 설정을 담는 구조체
type Config struct {
	Iris         IrisConfig
	ValkeyMQ     ValkeyMQConfig
	Server       ServerConfig
	Kakao        KakaoConfig
	Holodex      HolodexConfig
	YouTube      YouTubeConfig
	Valkey       ValkeyConfig
	Postgres     PostgresConfig
	Notification NotificationConfig
	Logging      LoggingConfig
	Bot          BotConfig
	Version      string
}

// IrisConfig: Iris 웹훅 서버 연결 및 메시지 전송 관련 설정
type IrisConfig struct {
	BaseURL string
}

// ValkeyMQConfig: Redis(Valkey) 기반 메시지 큐 통신 설정
type ValkeyMQConfig struct {
	Host                string
	Port                int
	Password            string
	StreamKey           string
	ConsumerGroup       string
	ConsumerName        string
	ReadCount           int
	BlockTimeoutSeconds int
	WorkerCount         int
}

// ServerConfig: 관리자용 웹 대시보드 및 API 서버 설정
type ServerConfig struct {
	Port            int
	AdminUser       string
	AdminPassHash   string // bcrypt 해시
	SessionSecret   string // 세션 HMAC 서명용
	ForceHTTPS      bool   // HTTPS 강제 여부
	AdminAllowedIPs []string
}

// KakaoConfig: 카카오톡 채팅방 허용 목록 및 접근 제어(ACL) 설정
type KakaoConfig struct {
	Rooms      []string
	ACLEnabled bool

	mu sync.RWMutex
}

// SnapshotACL: 현재 ACL 설정 상태(활성화 여부 및 허용된 방 목록)의 스냅샷을 반환한다.
// Thread-safe하게 읽기 락을 사용한다.
func (c *KakaoConfig) SnapshotACL() (enabled bool, rooms []string) {
	if c == nil {
		return false, nil
	}

	c.mu.RLock()
	defer c.mu.RUnlock()

	rooms = append([]string(nil), c.Rooms...)
	return c.ACLEnabled, rooms
}

// SetACLEnabled: ACL(접근 제어) 기능의 활성화 여부를 '동적으로' 설정한다.
func (c *KakaoConfig) SetACLEnabled(enabled bool) {
	if c == nil {
		return
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	c.ACLEnabled = enabled
}

// AddRoom: 허용 목록에 새로운 채팅방을 추가한다. 이미 존재하면 false를 반환한다.
func (c *KakaoConfig) AddRoom(room string) bool {
	if c == nil {
		return false
	}

	room = util.TrimSpace(room)
	if room == "" {
		return false
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	for _, existing := range c.Rooms {
		if existing == room {
			return false
		}
	}

	c.Rooms = append(c.Rooms, room)
	return true
}

// RemoveRoom: 허용 목록에서 특정 채팅방을 제거한다.
func (c *KakaoConfig) RemoveRoom(room string) bool {
	if c == nil {
		return false
	}

	room = util.TrimSpace(room)
	if room == "" {
		return false
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	removed := false
	rooms := make([]string, 0, len(c.Rooms))
	for _, existing := range c.Rooms {
		if existing == room {
			removed = true
			continue
		}
		rooms = append(rooms, existing)
	}

	c.Rooms = rooms
	return removed
}

// IsRoomAllowed: 해당 채팅방(chatID)이 봇 사용이 허용된 곳인지 확인한다.
// ACL이 비활성화되어 있으면 모든 방을 허용한다.
func (c *KakaoConfig) IsRoomAllowed(roomName, chatID string) bool {
	if c == nil {
		return true
	}

	chatID = util.TrimSpace(chatID)

	c.mu.RLock()
	defer c.mu.RUnlock()

	if !c.ACLEnabled {
		return true
	}

	// chatID 기반으로만 검증 (roomName은 참고용으로만 유지)
	if chatID == "" {
		return false // chatID가 없으면 거부
	}

	for _, allowed := range c.Rooms {
		if allowed == chatID {
			return true
		}
	}

	return false
}

// HolodexConfig: Holodex API 키 및 호출 관련 설정
type HolodexConfig struct {
	APIKeys []string
}

// YouTubeConfig: YouTube Data API 키 및 Quota 관리 설정
type YouTubeConfig struct {
	APIKey              string
	EnableQuotaBuilding bool
}

// ValkeyConfig: 데이터 캐싱 용도의 Redis(Valkey) 연결 설정
type ValkeyConfig struct {
	Host     string
	Port     int
	Password string
	DB       int
}

// PostgresConfig: 메인 데이터베이스(PostgreSQL) 연결 설정
type PostgresConfig struct {
	Host     string
	Port     int
	User     string
	Password string
	Database string
}

// NotificationConfig: 방송 알림 스케줄링(미리 알림 시간, 체크 주기) 설정
type NotificationConfig struct {
	AdvanceMinutes []int
	CheckInterval  time.Duration
}

// LoggingConfig: 애플리케이션 로그 설정 (레벨, 디렉토리, 로테이션 정책)
type LoggingConfig struct {
	Level      string
	Dir        string
	MaxSizeMB  int
	MaxBackups int
	MaxAgeDays int
	Compress   bool
}

// BotConfig: 봇의 기본 동작(명령어 접두사, 자기 자신 식별자) 설정
type BotConfig struct {
	Prefix   string
	SelfUser string
}

// Load: .env 파일 및 환경 변수로부터 설정을 로드하고, 기본값을 적용하여 Config 객체를 생성한다.
func Load() (*Config, error) {
	_ = godotenv.Load()

	cfg := &Config{
		Iris: IrisConfig{
			BaseURL: getEnv("IRIS_BASE_URL", "http://localhost:3000"),
		},
		ValkeyMQ: ValkeyMQConfig{
			Host:          getEnv("MQ_HOST", "localhost"),
			Port:          getEnvInt("MQ_PORT", 1833),
			Password:      getEnv("MQ_PASSWORD", ""),
			StreamKey:     getEnv("MQ_STREAM_KEY", "kakao:hololive"),
			ConsumerGroup: getEnv("MQ_CONSUMER_GROUP", "hololive-bot-group"),
			ConsumerName:  getEnv("MQ_CONSUMER_NAME", "consumer-1"),
			ReadCount:     getEnvInt("MQ_READ_COUNT", int(constants.MQConfig.ReadCount)),
			BlockTimeoutSeconds: getEnvInt(
				"MQ_BLOCK_TIMEOUT_SECONDS",
				int(constants.MQConfig.BlockTimeout.Seconds()),
			),
			WorkerCount: getEnvInt("MQ_WORKER_COUNT", constants.MQConfig.WorkerCount),
		},
		Server: ServerConfig{
			Port:            getEnvInt("SERVER_PORT", 30001),
			AdminUser:       getEnv("ADMIN_USER", "admin"),
			AdminPassHash:   getEnv("ADMIN_PASS_HASH", ""),
			SessionSecret:   getEnv("SESSION_SECRET", ""),
			ForceHTTPS:      getEnvBool("FORCE_HTTPS", false),
			AdminAllowedIPs: parseCommaSeparated(getEnv("ADMIN_ALLOWED_IPS", "")),
		},
		Kakao: KakaoConfig{
			Rooms:      parseCommaSeparated(getEnv("KAKAO_ROOMS", "홀로라이브 알림방")),
			ACLEnabled: getEnvBool("KAKAO_ACL_ENABLED", true),
		},
		Holodex: HolodexConfig{
			APIKeys: collectAPIKeys("HOLODEX_API_KEY_"),
		},
		YouTube: YouTubeConfig{
			APIKey:              getEnv("YOUTUBE_API_KEY", ""),
			EnableQuotaBuilding: getEnvBool("YOUTUBE_ENABLE_QUOTA_BUILDING", false),
		},
		Valkey: ValkeyConfig{
			Host:     getEnv("CACHE_HOST", "localhost"),
			Port:     getEnvInt("CACHE_PORT", 6379),
			Password: getEnv("CACHE_PASSWORD", ""),
			DB:       getEnvInt("CACHE_DB", 0),
		},
		Postgres: PostgresConfig{
			Host:     getEnv("POSTGRES_HOST", constants.DatabaseDefaults.Host),
			Port:     getEnvInt("POSTGRES_PORT", constants.DatabaseDefaults.Port),
			User:     getEnv("POSTGRES_USER", constants.DatabaseDefaults.User),
			Password: getEnv("POSTGRES_PASSWORD", constants.DatabaseDefaults.Password),
			Database: getEnv("POSTGRES_DB", constants.DatabaseDefaults.Database),
		},
		Notification: NotificationConfig{
			AdvanceMinutes: parseIntList(getEnv("NOTIFICATION_ADVANCE_MINUTES", "5,15,30")),
			CheckInterval:  time.Duration(getEnvInt("CHECK_INTERVAL_SECONDS", 60)) * time.Second,
		},
		Logging: LoggingConfig{
			Level:      getEnv("LOG_LEVEL", "info"),
			Dir:        getEnv("LOG_DIR", "logs"),
			MaxSizeMB:  getEnvInt("LOG_MAX_SIZE_MB", 100),
			MaxBackups: getEnvInt("LOG_MAX_BACKUPS", 5),
			MaxAgeDays: getEnvInt("LOG_MAX_AGE_DAYS", 30),
			Compress:   getEnvBool("LOG_COMPRESS", true),
		},
		Bot: BotConfig{
			Prefix:   getEnv("BOT_PREFIX", "!"),
			SelfUser: util.TrimSpace(getEnv("BOT_SELF_USER", "iris")),
		},
		Version: util.TrimSpace(getEnv("APP_VERSION", "1.1.0-go")),
	}

	if err := cfg.Validate(); err != nil {
		return nil, fmt.Errorf("config validation failed: %w", err)
	}

	return cfg, nil
}

// Validate: 필수 설정값이 누락되지 않았는지 검증한다.
func (c *Config) Validate() error {
	if c.Server.Port == 0 {
		return fmt.Errorf("SERVER_PORT is required")
	}
	if len(c.Kakao.Rooms) == 0 {
		return fmt.Errorf("KAKAO_ROOMS is required")
	}
	if len(c.Holodex.APIKeys) == 0 {
		return fmt.Errorf("at least one HOLODEX_API_KEY is required")
	}
	if c.Server.AdminPassHash == "" {
		return fmt.Errorf("ADMIN_PASS_HASH is required for admin panel")
	}
	if c.Server.SessionSecret == "" {
		return fmt.Errorf("SESSION_SECRET is required for session security")
	}
	return nil
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func getEnvInt(key string, defaultValue int) int {
	if value := os.Getenv(key); value != "" {
		if intVal, err := strconv.Atoi(value); err == nil {
			return intVal
		}
	}
	return defaultValue
}

func getEnvBool(key string, defaultValue bool) bool {
	if value := os.Getenv(key); value != "" {
		if boolVal, err := strconv.ParseBool(value); err == nil {
			return boolVal
		}
	}
	return defaultValue
}

func parseCommaSeparated(value string) []string {
	if value == "" {
		return []string{}
	}
	parts := strings.Split(value, ",")
	result := make([]string, 0, len(parts))
	for _, part := range parts {
		if trimmed := util.TrimSpace(part); trimmed != "" {
			result = append(result, trimmed)
		}
	}
	return result
}

func parseIntList(value string) []int {
	if value == "" {
		return []int{}
	}
	parts := strings.Split(value, ",")
	result := make([]int, 0, len(parts))
	for _, part := range parts {
		if trimmed := util.TrimSpace(part); trimmed != "" {
			if intVal, err := strconv.Atoi(trimmed); err == nil {
				result = append(result, intVal)
			}
		}
	}
	return result
}

func collectAPIKeys(prefix string) []string {
	keys := make([]string, 0)
	seen := make(map[string]struct{})

	addKey := func(raw string) {
		trimmed := util.TrimSpace(raw)
		if trimmed == "" {
			return
		}
		if _, exists := seen[trimmed]; exists {
			return
		}
		seen[trimmed] = struct{}{}
		keys = append(keys, trimmed)
	}

	for i := 1; i <= maxHolodexAPIKeySlots; i++ {
		envKey := fmt.Sprintf("%s%d", prefix, i)
		addKey(os.Getenv(envKey))
	}

	if base := strings.TrimSuffix(prefix, "_"); base != "" {
		if bulk := os.Getenv(base + "S"); bulk != "" {
			parts := strings.Split(bulk, ",")
			for _, part := range parts {
				addKey(part)
			}
		}
	}

	return keys
}
