package shared

// 한국어 응답 메시지 상수
const (
	// MsgSafetyBlock 은 안전 차단 메시지다.
	MsgSafetyBlock = "답변이 제한됩니다. 다시 질문해주세요"

	// MsgInvalidQuestion 은 이해 불가 질문 메시지다.
	MsgInvalidQuestion = "이해할 수 없는 질문입니다"

	// MsgCannotAnswer 은 답변 불가 메시지다.
	MsgCannotAnswer = "답변할 수 없습니다"
)

// 기본값 상수
const (
	// DefaultCategory 기본 카테고리
	DefaultCategory = "MYSTERY"

	// DefaultDifficulty 기본 난이도
	DefaultDifficulty = 3

	// MinDifficulty 최소 난이도
	MinDifficulty = 1

	// MaxDifficulty 최대 난이도
	MaxDifficulty = 5

	// DefaultHistoryHeader 기본 히스토리 헤더
	DefaultHistoryHeader = "[이전 질문/답변 기록]"
)
