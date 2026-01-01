package config

import "time"

// ServerConfig: HTTP 서버 주소/포트 설정입니다.
type ServerConfig struct {
	Host string // 서버 바인딩 호스트
	Port int    // 서버 리스닝 포트
}

// CommandsConfig: 봇 명령어 설정입니다.
type CommandsConfig struct {
	Prefix string // 명령어 접두사 (ex: "/스프")
}

// LlmConfig: LLM 서버 gRPC 통신 설정입니다.
type LlmConfig struct {
	BaseURL        string
	APIKey         string
	Timeout        time.Duration
	ConnectTimeout time.Duration
}

// RedisConfig: Redis/Valkey 캐시 연결 설정입니다.
type RedisConfig struct {
	Host       string // 서버 호스트
	Port       int    // 서버 포트
	Password   string // 인증 패스워드
	DB         int    // 사용할 DB 번호
	SocketPath string // UDS 경로 (비어있으면 TCP 사용)

	DialTimeout  time.Duration // 연결 타임아웃
	ReadTimeout  time.Duration // 읽기 타임아웃
	WriteTimeout time.Duration // 쓰기 타임아웃

	PoolSize     int // 커넥션 풀 크기
	MinIdleConns int // 최소 유휴 커넥션 수
}

// ValkeyMQConfig: Valkey Streams 기반 메시지 큐 설정입니다.
type ValkeyMQConfig struct {
	Host     string // MQ 서버 호스트
	Port     int    // MQ 서버 포트
	Password string // 인증 패스워드
	DB       int    // 사용할 DB 번호

	Timeout                     time.Duration // 명령 타임아웃
	DialTimeout                 time.Duration // 연결 타임아웃
	PoolSize                    int           // 커넥션 풀 크기
	MinIdleConns                int           // 최소 유휴 커넥션 수
	ConsumerGroup               string        // Consumer Group 이름
	ConsumerName                string        // Consumer 식별자
	ResetConsumerGroupOnStartup bool          // 시작 시 Consumer Group 초기화 여부
	StreamKey                   string        // 인바운드 스트림 키
	ReplyStreamKey              string        // 아웃바운드(응답) 스트림 키

	BatchSize    int64         // 한 번에 읽을 메시지 수
	BlockTimeout time.Duration // XREAD 블록 타임아웃
	Concurrency  int           // 동시 처리 워커 수
	StreamMaxLen int64         // 스트림 최대 길이 (MAXLEN)
}

// AccessConfig: 채팅방/사용자 접근 제어 설정입니다.
type AccessConfig struct {
	Enabled        bool     // 접근 제어 활성화 여부
	AllowedChatIDs []string // 허용된 채팅방 ID 목록
	BlockedChatIDs []string // 차단된 채팅방 ID 목록
	BlockedUserIDs []string // 차단된 사용자 ID 목록
	Passthrough    bool     // Passthrough 모드 (검사 스킵)
}

// LogConfig: 파일 로그 로테이션 설정입니다.
type LogConfig struct {
	Dir string // 로그 파일 디렉터리

	MaxSizeMB  int  // 단일 파일 최대 크기 (MB)
	MaxBackups int  // 보관할 백업 파일 수
	MaxAgeDays int  // 백업 파일 보관 일수
	Compress   bool // 백업 파일 압축 여부
}

// ServerTuningConfig: HTTP 서버 튜닝 설정(Timeouts, Limits)입니다.
type ServerTuningConfig struct {
	ReadHeaderTimeout time.Duration
	IdleTimeout       time.Duration
	MaxHeaderBytes    int
}
