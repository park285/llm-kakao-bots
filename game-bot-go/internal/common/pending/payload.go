package pending

// BaseMessagePayload: 대기열 JSON payload의 공통 필드를 담는 구조체입니다.
// UserID/Timestamp는 Redis ZSET/HASH 메타데이터로 관리하여 JSON 중복을 줄이기 위함입니다.
type BaseMessagePayload struct {
	Content  string  `json:"content"`
	ThreadID *string `json:"threadId,omitempty"`
	Sender   *string `json:"sender,omitempty"`
}
