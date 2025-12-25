// Package errors: 스무고개(TwentyQ) 게임에 특화된 에러 타입들을 정의한다.
// 공통 에러 타입(RedisError, LockError 등)은 common/errors 패키지를 직접 사용한다.
package errors

import "fmt"

// SessionNotFoundError: 게임 세션을 찾을 수 없을 때 발생하는 에러
type SessionNotFoundError struct {
	ChatID string
}

func (e SessionNotFoundError) Error() string {
	if e.ChatID == "" {
		return "session not found"
	}
	return fmt.Sprintf("session not found chatId=%s", e.ChatID)
}

// DuplicateQuestionError: 동일한 세션에서 이미 질문했던 내용일 때 발생하는 에러
type DuplicateQuestionError struct{}

func (e DuplicateQuestionError) Error() string { return "duplicate question" }

// HintLimitExceededError: 게임별 최대 힌트 사용 횟수를 초과했을 때 발생하는 에러
type HintLimitExceededError struct {
	MaxHints  int
	HintCount int
	Remaining int
}

func (e HintLimitExceededError) Error() string {
	return fmt.Sprintf("hint limit exceeded hintCount=%d maxHints=%d remaining=%d", e.HintCount, e.MaxHints, e.Remaining)
}

// HintNotAvailableError: 힌트를 생성할 수 없는 상태(예: 초반 진행)일 때 발생하는 에러
type HintNotAvailableError struct{}

func (e HintNotAvailableError) Error() string { return "hint not available" }
