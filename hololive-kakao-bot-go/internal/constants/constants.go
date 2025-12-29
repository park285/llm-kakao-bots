package constants

import "time"

// CacheTTL 는 패키지 변수다.
var CacheTTL = struct {
	LiveStreams      time.Duration
	UpcomingStreams  time.Duration
	ChannelSchedule  time.Duration
	ChannelInfo      time.Duration
	ChannelSearch    time.Duration
	NextStreamInfo   time.Duration
	NotificationSent time.Duration
}{
	LiveStreams:      5 * time.Minute,  // 5분 - 라이브 스트림 목록
	UpcomingStreams:  5 * time.Minute,  // 5분 - 예정 스트림 목록
	ChannelSchedule:  5 * time.Minute,  // 5분 - 채널 스케줄
	ChannelInfo:      20 * time.Minute, // 20분 - 채널 정보
	ChannelSearch:    10 * time.Minute, // 10분 - 채널 검색 결과
	NextStreamInfo:   60 * time.Minute, // 1시간 - 다음 방송 정보
	NotificationSent: 24 * time.Hour,   // 24시간 - 알림 발송 기록
}

// MemberCacheDefaults 는 패키지 변수다.
var MemberCacheDefaults = struct {
	ValkeyTTL           time.Duration
	WarmUpChunkSize     int
	WarmUpMaxGoroutines int
}{
	ValkeyTTL:           30 * time.Minute,
	WarmUpChunkSize:     50,
	WarmUpMaxGoroutines: 10,
}

// WebSocketConfig 는 패키지 변수다.
var WebSocketConfig = struct {
	MaxReconnectAttempts int
	ReconnectDelay       time.Duration
}{
	MaxReconnectAttempts: 5,
	ReconnectDelay:       5 * time.Second,
}

// ValkeyConfig 는 패키지 변수다.
var ValkeyConfig = struct {
	ReadyTimeout      time.Duration
	BlockingPoolSize  int
	PipelineMultiplex int
}{
	ReadyTimeout:      5 * time.Second,
	BlockingPoolSize:  100,
	PipelineMultiplex: 4,
}

// AIInputLimits 는 패키지 변수다.
var AIInputLimits = struct {
	MaxQueryLength int
}{
	MaxQueryLength: 500,
}

// RetryConfig 는 패키지 변수다.
var RetryConfig = struct {
	MaxAttempts int
	BaseDelay   time.Duration
	Jitter      time.Duration
}{
	MaxAttempts: 3,
	BaseDelay:   500 * time.Millisecond,
	Jitter:      250 * time.Millisecond,
}

// CircuitBreakerConfig 는 패키지 변수다.
var CircuitBreakerConfig = struct {
	FailureThreshold    int
	ResetTimeout        time.Duration
	RateLimitTimeout    time.Duration
	HealthCheckInterval time.Duration
	HealthCheckTimeout  time.Duration
}{
	FailureThreshold:    3,                // 3회 연속 실패 시 Circuit OPEN
	ResetTimeout:        30 * time.Second, // 기본 재시도 대기 시간 (30초)
	RateLimitTimeout:    1 * time.Hour,    // 429 Rate Limit 전용 타임아웃 (1시간)
	HealthCheckInterval: 10 * time.Minute, // Health Check 주기 (10분)
	HealthCheckTimeout:  10 * time.Second, // Health Check 타임아웃 (10초)
}

// PaginationConfig 는 패키지 변수다.
var PaginationConfig = struct {
	ItemsPerPage   int
	Timeout        time.Duration
	MaxEmbedFields int
}{
	ItemsPerPage:   10,              // 페이지당 항목 수
	Timeout:        3 * time.Minute, // 페이지네이션 타임아웃
	MaxEmbedFields: 25,              // Discord Embed 필드 최대 개수
}

// APIConfig 는 패키지 변수다.
var APIConfig = struct {
	HolodexBaseURL   string
	HolodexTimeout   time.Duration
	MaxRetryAttempts int
}{
	HolodexBaseURL:   "https://holodex.net/api/v2",
	HolodexTimeout:   15 * time.Second, // 간헐적 서버 지연 대응 (10s → 15s)
	MaxRetryAttempts: 3,
}

// HolodexTransportConfig 는 Holodex HTTP Transport 설정이다.
// 동시 요청 시 커넥션 풀 고갈 방지를 위해 디폴트(MaxIdleConnsPerHost=2)보다 높게 설정한다.
var HolodexTransportConfig = struct {
	MaxConnsPerHost     int
	MaxIdleConnsPerHost int
	IdleConnTimeout     time.Duration
}{
	MaxConnsPerHost:     50, // 최대 동시 연결 수 (maxConcurrency와 동일)
	MaxIdleConnsPerHost: 50, // 유휴 커넥션 유지 수
	IdleConnTimeout:     30 * time.Second,
}

// OfficialScheduleConfig 는 패키지 변수다.
var OfficialScheduleConfig = struct {
	BaseURL     string
	Timeout     time.Duration
	CacheExpiry time.Duration
}{
	BaseURL:     "https://schedule.hololive.tv",
	Timeout:     15 * time.Second,
	CacheExpiry: 30 * time.Minute,
}

// OfficialProfileConfig 는 패키지 변수다.
var OfficialProfileConfig = struct {
	BaseURL        string
	UserAgent      string
	AcceptLanguage string
	RequestTimeout time.Duration
	DelayBetween   time.Duration
	OutputFile     string
}{
	BaseURL:        "https://hololive.hololivepro.com/talents",
	UserAgent:      "Mozilla/5.0 (compatible; HololiveKakaoBot/1.0; +https://hololive.hololivepro.com)",
	AcceptLanguage: "ja,en;q=0.8,ko;q=0.6",
	RequestTimeout: 15 * time.Second,
	DelayBetween:   350 * time.Millisecond,
	OutputFile:     "internal/domain/data/official_profiles_raw.json",
}

// YouTubeConfig 는 패키지 변수다.
var YouTubeConfig = struct {
	DailyQuotaLimit       int
	SearchQuotaCost       int
	ChannelsQuotaCost     int
	MaxChannelsPerCall    int
	MaxConcurrentRequests int
	SearchMaxResults      int
	QuotaSafetyMargin     int
	CacheExpiration       time.Duration
}{
	DailyQuotaLimit:       10000,
	SearchQuotaCost:       100,
	ChannelsQuotaCost:     1,
	MaxChannelsPerCall:    20,
	MaxConcurrentRequests: 3,
	SearchMaxResults:      10,
	QuotaSafetyMargin:     2000,
	CacheExpiration:       2 * time.Hour,
}

// StringLimits 는 패키지 변수다.
var StringLimits = struct {
	EmbedTitle       int
	EmbedDescription int
	EmbedFieldName   int
	EmbedFieldValue  int
	StreamTitle      int
	NextStreamTitle  int
}{
	EmbedTitle:       256,
	EmbedDescription: 4096,
	EmbedFieldName:   256,
	EmbedFieldValue:  1024,
	StreamTitle:      100,
	NextStreamTitle:  40,
}

// MQConfig 는 패키지 변수다.
var MQConfig = struct {
	ReplyStreamKey    string
	ConsumerGroup     string
	ConnWriteTimeout  time.Duration
	BlockingPoolSize  int
	PipelineMultiplex int
	DialTimeout       time.Duration
	BlockTimeout      time.Duration
	ReadCount         int64
	WorkerCount       int
	IdempotencyTTL    time.Duration
	InitRetryCount    int
	RetryDelay        time.Duration
}{
	ReplyStreamKey:    "kakao:bot:reply",
	ConsumerGroup:     "hololive-bot-group",
	ConnWriteTimeout:  3 * time.Second,
	BlockingPoolSize:  50,
	PipelineMultiplex: 4,
	DialTimeout:       5 * time.Second,
	BlockTimeout:      5 * time.Second,
	ReadCount:         50,
	WorkerCount:       10,
	IdempotencyTTL:    24 * time.Hour,
	InitRetryCount:    10,
	RetryDelay:        1 * time.Second,
}

// AppTimeout 는 앱 빌드/종료 타임아웃 설정이다.
var AppTimeout = struct {
	Build    time.Duration
	Shutdown time.Duration
}{
	Build:    30 * time.Second,
	Shutdown: 10 * time.Second,
}

// ServerTimeout 는 HTTP 서버 타임아웃이다.
var ServerTimeout = struct {
	ReadHeader time.Duration
	Idle       time.Duration
}{
	ReadHeader: 5 * time.Second,
	Idle:       60 * time.Second,
}

// ServerConfig 는 서버 기본 설정이다.
var ServerConfig = struct {
	TrustedProxies []string
}{
	TrustedProxies: []string{"127.0.0.1", "::1"},
}

// CORSConfig 는 CORS 기본 설정이다.
var CORSConfig = struct {
	AllowOrigins []string
	AllowMethods []string
	AllowHeaders []string
}{
	AllowOrigins: []string{"http://localhost:5173"},
	AllowMethods: []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
	AllowHeaders: []string{"Origin", "Content-Type", "Accept", "Authorization"},
}

// AdminUIConfig 는 Admin UI 정적 파일/캐시 설정이다.
var AdminUIConfig = struct {
	AssetsRoute         string
	AssetsURLPrefix     string
	AssetsDir           string
	IndexPath           string
	FaviconRoute        string
	FaviconPath         string
	CacheControlAssets  string
	CacheControlHTML    string
	CacheControlFavicon string
}{
	AssetsRoute:         "/assets",
	AssetsURLPrefix:     "/assets/",
	AssetsDir:           "./admin-ui/dist/assets",
	IndexPath:           "./admin-ui/dist/index.html",
	FaviconRoute:        "/favicon.svg",
	FaviconPath:         "./admin-ui/dist/favicon.svg",
	CacheControlAssets:  "public, max-age=31536000, immutable",
	CacheControlHTML:    "no-store, no-cache, must-revalidate",
	CacheControlFavicon: "public, max-age=86400",
}

// RequestTimeout 는 HTTP 요청 및 서비스 타임아웃 설정
var RequestTimeout = struct {
	AdminRequest      time.Duration
	BotCommand        time.Duration
	BotAlarmCheck     time.Duration
	WebhookProcessing time.Duration
	AlarmService      time.Duration
	DatabasePing      time.Duration
}{
	AdminRequest:      10 * time.Second,
	BotCommand:        10 * time.Second,
	BotAlarmCheck:     2 * time.Minute,
	WebhookProcessing: 30 * time.Second,
	AlarmService:      10 * time.Second,
	DatabasePing:      5 * time.Second,
}

// SessionConfig 는 세션 관련 설정이다.
// ExpiryDuration: 세션 TTL (heartbeat 미수신 시 만료)
// HeartbeatInterval: 프론트엔드 heartbeat 주기
var SessionConfig = struct {
	ExpiryDuration    time.Duration
	HeartbeatInterval time.Duration
}{
	ExpiryDuration:    1 * time.Hour,    // 브라우저 닫기 → 1시간 후 Valkey 세션 만료
	HeartbeatInterval: 15 * time.Minute, // 프론트엔드 heartbeat 주기
}

// DatabaseConfig 는 데이터베이스 연결 설정이다.
var DatabaseConfig = struct {
	MaxOpenConns    int
	MaxIdleConns    int
	ConnMaxLifetime time.Duration
}{
	MaxOpenConns:    25,
	MaxIdleConns:    5,
	ConnMaxLifetime: 5 * time.Minute,
}

// DatabaseDefaults 는 PostgreSQL 기본값이다. (env 미설정 시)
var DatabaseDefaults = struct {
	Host     string
	Port     int
	User     string
	Password string
	Database string
}{
	Host:     "localhost",
	Port:     5432,
	User:     "holo_user",
	Password: "holo_password",
	Database: "holo_oshi_db",
}
