// Package errors: 바다거북스프(Turtle Soup) 게임에 특화된 에러 타입들을 정의합니다.
// 공통 에러 타입(RedisError, LockError 등)은 common/errors 패키지를 직접 사용합니다.
package errors

import (
	"errors"
	"fmt"

	cerrors "github.com/park285/llm-kakao-bots/game-bot-go/internal/common/errors"
)

// SessionNotFoundError: 세션을 찾을 수 없을 때 발생하는 에러
type SessionNotFoundError struct {
	SessionID string
}

func (e SessionNotFoundError) Error() string {
	return fmt.Sprintf("session not found: %s", e.SessionID)
}

// GameAlreadyStartedError: 이미 진행 중인 게임이 있을 때 발생하는 에러
type GameAlreadyStartedError struct {
	SessionID string
}

func (e GameAlreadyStartedError) Error() string {
	return fmt.Sprintf("game already started: %s", e.SessionID)
}

// GameNotStartedError: 진행 중인 게임이 없을 때 발생하는 에러
type GameNotStartedError struct {
	SessionID string
}

func (e GameNotStartedError) Error() string { return fmt.Sprintf("game not started: %s", e.SessionID) }

// GameAlreadySolvedError: 이미 해결된 게임에 대해 작업을 시도할 때 발생하는 에러
type GameAlreadySolvedError struct {
	SessionID string
}

func (e GameAlreadySolvedError) Error() string {
	return fmt.Sprintf("game already solved: %s", e.SessionID)
}

// MaxHintsReachedError: 힌트 사용 제한을 초과했을 때 발생하는 에러
type MaxHintsReachedError struct {
	MaxHints int
}

func (e MaxHintsReachedError) Error() string {
	return fmt.Sprintf("maximum hints reached: %d", e.MaxHints)
}

// PuzzleGenerationError: 퍼즐 자동 생성 중 발생한 에러
type PuzzleGenerationError struct {
	Err error
}

func (e PuzzleGenerationError) Error() string {
	if e.Err == nil {
		return "failed to generate puzzle"
	}
	return fmt.Sprintf("failed to generate puzzle: %v", e.Err)
}

func (e PuzzleGenerationError) Unwrap() error { return e.Err }

// IsExpectedUserBehavior: 에러가 사용자의 정상적인(예상된) 패턴 내의 실수인지 확인합니다.
// (로그 레벨을 낮추거나 사용자에게 친절한 메시지를 보내는 용도)
func IsExpectedUserBehavior(err error) bool {
	if err == nil {
		return false
	}

	// 공통 에러 체크
	if cerrors.IsExpectedUserBehavior(err) {
		return true
	}

	// 도메인 특화 에러 체크
	expectedTypes := []any{
		new(SessionNotFoundError),
		new(GameNotStartedError),
		new(GameAlreadyStartedError),
		new(GameAlreadySolvedError),
		new(MaxHintsReachedError),
	}

	for _, target := range expectedTypes {
		if errors.As(err, target) {
			return true
		}
	}
	return false
}
