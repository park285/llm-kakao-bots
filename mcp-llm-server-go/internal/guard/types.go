package guard

import "fmt"

// Match: 매칭된 규칙 정보를 담습니다.
type Match struct {
	ID     string  `json:"id"`
	Weight float64 `json:"weight"`
}

// Evaluation: 검사 결과를 담습니다.
type Evaluation struct {
	Score     float64 `json:"score"`
	Hits      []Match `json:"hits"`
	Threshold float64 `json:"threshold"`
}

// Malicious: 위험 여부를 반환합니다.
func (e Evaluation) Malicious() bool {
	return e.Score >= e.Threshold
}

// BlockedError: 차단된 입력 오류입니다.
type BlockedError struct {
	Score     float64
	Threshold float64
}

// Error: 오류 메시지를 반환합니다.
func (e *BlockedError) Error() string {
	return fmt.Sprintf("input blocked by injection guard (score=%.2f, threshold=%.2f)", e.Score, e.Threshold)
}
