package pending

// EnqueueResult Lua 스크립트 enqueue 결과.
type EnqueueResult int

// EnqueueResult 상수 목록.
const (
	// EnqueueSuccess 는 상수다.
	EnqueueSuccess EnqueueResult = iota
	EnqueueQueueFull
	EnqueueDuplicate
)

func (r EnqueueResult) String() string {
	switch r {
	case EnqueueSuccess:
		return "SUCCESS"
	case EnqueueQueueFull:
		return "QUEUE_FULL"
	case EnqueueDuplicate:
		return "DUPLICATE"
	default:
		return "UNKNOWN"
	}
}

// DequeueStatus Lua 스크립트 dequeue 결과 상태.
type DequeueStatus int

const (
	// DequeueEmpty 큐가 비어있음.
	DequeueEmpty DequeueStatus = iota
	// DequeueExhausted 루프 제한 도달 (뒤에 데이터 있을 수 있음).
	DequeueExhausted
	// DequeueSuccess 유효한 메시지 반환.
	DequeueSuccess
)

func (s DequeueStatus) String() string {
	switch s {
	case DequeueEmpty:
		return "EMPTY"
	case DequeueExhausted:
		return "EXHAUSTED"
	case DequeueSuccess:
		return "SUCCESS"
	default:
		return "UNKNOWN"
	}
}

// Message 큐에 저장되는 메시지 인터페이스.
// UserID와 Timestamp는 Lua 스크립트에서 중복 체크/stale 검사에 필수.
type Message interface {
	// GetUserID 중복 체크용 사용자 ID.
	GetUserID() string
	// GetTimestamp stale 체크용 타임스탬프 (Unix ms).
	GetTimestamp() int64
}

// Config PendingMessageStore 설정.
type Config struct {
	// KeyPrefix Redis 키 프리픽스 (예: "pending:twentyq", "pending:turtlesoup").
	KeyPrefix string
	// MaxQueueSize 큐 최대 크기.
	MaxQueueSize int
	// QueueTTLSeconds 큐 TTL (초).
	QueueTTLSeconds int
	// StaleThresholdMS stale 메시지 임계값 (밀리초).
	StaleThresholdMS int64
	// MaxDequeueIterations dequeue 루프 최대 반복 횟수.
	MaxDequeueIterations int
}

// DefaultConfig 기본 설정.
func DefaultConfig(keyPrefix string) Config {
	return Config{
		KeyPrefix:            keyPrefix,
		MaxQueueSize:         5,
		QueueTTLSeconds:      300,
		StaleThresholdMS:     3600_000, // 1시간
		MaxDequeueIterations: 50,
	}
}
