package errors

import "fmt"

// SessionNotFoundError 는 타입이다.
type SessionNotFoundError struct {
	ChatID string
}

func (e SessionNotFoundError) Error() string {
	if e.ChatID == "" {
		return "session not found"
	}
	return fmt.Sprintf("session not found chatId=%s", e.ChatID)
}

// InvalidQuestionError 는 타입이다.
type InvalidQuestionError struct {
	Message string
}

func (e InvalidQuestionError) Error() string {
	if e.Message == "" {
		return "invalid question"
	}
	return "invalid question: " + e.Message
}

// DuplicateQuestionError 는 타입이다.
type DuplicateQuestionError struct{}

func (e DuplicateQuestionError) Error() string { return "duplicate question" }

// HintLimitExceededError 는 타입이다.
type HintLimitExceededError struct {
	MaxHints  int
	HintCount int
	Remaining int
}

func (e HintLimitExceededError) Error() string {
	return fmt.Sprintf("hint limit exceeded hintCount=%d maxHints=%d remaining=%d", e.HintCount, e.MaxHints, e.Remaining)
}

// HintNotAvailableError 는 타입이다.
type HintNotAvailableError struct{}

func (e HintNotAvailableError) Error() string { return "hint not available" }

// AccessDeniedError 는 타입이다.
type AccessDeniedError struct {
	Reason string
}

func (e AccessDeniedError) Error() string { return "access denied" }

// UserBlockedError 는 타입이다.
type UserBlockedError struct {
	UserID string
}

func (e UserBlockedError) Error() string { return "user blocked" }

// ChatBlockedError 는 타입이다.
type ChatBlockedError struct {
	ChatID string
}

func (e ChatBlockedError) Error() string { return "chat blocked" }

// LockError 는 타입이다.
type LockError struct {
	ChatID      string
	HolderName  *string
	Description string
}

func (e LockError) Error() string {
	desc := e.Description
	if desc == "" {
		desc = "lock error"
	}
	return desc
}

// RedisError 는 타입이다.
type RedisError struct {
	Operation string
	Err       error
}

func (e RedisError) Error() string {
	return fmt.Sprintf("redis error op=%s: %v", e.Operation, e.Err)
}

func (e RedisError) Unwrap() error { return e.Err }

// DatabaseError 는 타입이다.
type DatabaseError struct {
	Operation string
	Err       error
}

func (e DatabaseError) Error() string {
	return fmt.Sprintf("db error op=%s: %v", e.Operation, e.Err)
}

func (e DatabaseError) Unwrap() error { return e.Err }
