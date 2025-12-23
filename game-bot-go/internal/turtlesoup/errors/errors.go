package errors

import (
	"errors"
	"fmt"
)

// SessionNotFoundError 는 타입이다.
type SessionNotFoundError struct {
	SessionID string
}

func (e SessionNotFoundError) Error() string {
	return fmt.Sprintf("session not found: %s", e.SessionID)
}

// InvalidQuestionError 는 타입이다.
type InvalidQuestionError struct {
	Message string
}

func (e InvalidQuestionError) Error() string { return e.Message }

// InvalidAnswerError 는 타입이다.
type InvalidAnswerError struct {
	Message string
}

func (e InvalidAnswerError) Error() string { return e.Message }

// GameAlreadyStartedError 는 타입이다.
type GameAlreadyStartedError struct {
	SessionID string
}

func (e GameAlreadyStartedError) Error() string {
	return fmt.Sprintf("game already started: %s", e.SessionID)
}

// GameNotStartedError 는 타입이다.
type GameNotStartedError struct {
	SessionID string
}

func (e GameNotStartedError) Error() string { return fmt.Sprintf("game not started: %s", e.SessionID) }

// GameAlreadySolvedError 는 타입이다.
type GameAlreadySolvedError struct {
	SessionID string
}

func (e GameAlreadySolvedError) Error() string {
	return fmt.Sprintf("game already solved: %s", e.SessionID)
}

// MaxHintsReachedError 는 타입이다.
type MaxHintsReachedError struct {
	MaxHints int
}

func (e MaxHintsReachedError) Error() string {
	return fmt.Sprintf("maximum hints reached: %d", e.MaxHints)
}

// PuzzleGenerationError 는 타입이다.
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

// RedisError 는 타입이다.
type RedisError struct {
	Operation string
	Err       error
}

func (e RedisError) Error() string {
	if e.Err == nil {
		return fmt.Sprintf("redis error operation=%s", e.Operation)
	}
	return fmt.Sprintf("redis error operation=%s: %v", e.Operation, e.Err)
}

func (e RedisError) Unwrap() error { return e.Err }

// LockError 는 타입이다.
type LockError struct {
	SessionID   string
	HolderName  *string
	Description string
}

func (e LockError) Error() string {
	msg := e.Description
	if msg == "" {
		msg = "failed to acquire lock"
	}
	if e.SessionID != "" {
		msg = fmt.Sprintf("%s session=%s", msg, e.SessionID)
	}
	if e.HolderName != nil && *e.HolderName != "" {
		msg = fmt.Sprintf("%s holder=%s", msg, *e.HolderName)
	}
	return msg
}

// AccessDeniedError 는 타입이다.
type AccessDeniedError struct{ Reason string }

func (e AccessDeniedError) Error() string { return fmt.Sprintf("access denied: %s", e.Reason) }

// UserBlockedError 는 타입이다.
type UserBlockedError struct{ UserID string }

func (e UserBlockedError) Error() string { return fmt.Sprintf("user blocked: %s", e.UserID) }

// ChatBlockedError 는 타입이다.
type ChatBlockedError struct{ ChatID string }

func (e ChatBlockedError) Error() string { return fmt.Sprintf("chat blocked: %s", e.ChatID) }

// InputInjectionError 는 타입이다.
type InputInjectionError struct {
	Message string
}

func (e InputInjectionError) Error() string { return e.Message }

// MalformedInputError 는 타입이다.
type MalformedInputError struct {
	Message string
}

func (e MalformedInputError) Error() string { return e.Message }

// IsExpectedUserBehavior 는 동작을 수행한다.
func IsExpectedUserBehavior(err error) bool {
	if err == nil {
		return false
	}

	var (
		_ SessionNotFoundError
		_ GameNotStartedError
		_ GameAlreadyStartedError
		_ GameAlreadySolvedError
		_ MaxHintsReachedError
		_ InvalidQuestionError
		_ InvalidAnswerError
		_ MalformedInputError
	)

	switch {
	case errors.As(err, new(SessionNotFoundError)):
		return true
	case errors.As(err, new(GameNotStartedError)):
		return true
	case errors.As(err, new(GameAlreadyStartedError)):
		return true
	case errors.As(err, new(GameAlreadySolvedError)):
		return true
	case errors.As(err, new(MaxHintsReachedError)):
		return true
	case errors.As(err, new(InvalidQuestionError)):
		return true
	case errors.As(err, new(InvalidAnswerError)):
		return true
	case errors.As(err, new(MalformedInputError)):
		return true
	default:
		return false
	}
}
