package pending

// EnqueueResult Lua 스크립트 enqueue 결과.
type EnqueueResult int

// EnqueueResult: 메시지 등록 결과를 나타내는 상태 상수
const (
	// EnqueueSuccess: 메시지가 성공적으로 대기열에 등록됨
	EnqueueSuccess EnqueueResult = iota
	// EnqueueQueueFull: 대기열이 가득 차서 등록 실패
	EnqueueQueueFull
	// EnqueueDuplicate: 이미 해당 사용자의 대기 메시지가 존재함 (중복)
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

// DequeueStatus: 메시지 추출 결과를 나타내는 상태 상수
const (
	// DequeueEmpty: 대기열이 비어있음
	DequeueEmpty DequeueStatus = iota
	// DequeueExhausted: 처리 가능한 메시지를 찾지 못하고 탐색 종료 (스킵된 메시지만 있을 경우 등)
	DequeueExhausted
	// DequeueSuccess: 유효한 메시지를 성공적으로 추출함
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

// Message: 큐에 저장될 수 있는 메시지 객체의 공통 인터페이스
type Message interface {
	// GetUserID: 중복 체크를 위한 사용자 식별자를 반환한다.
	GetUserID() string
	// GetTimestamp: 오래된 메시지 처리를 위한 타임스탬프(Unix ms)를 반환한다.
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
