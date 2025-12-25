// Package errors: 게임 서비스 전체에서 공용으로 사용되는 에러 타입들을 정의한다.
// twentyq, turtlesoup 등 도메인 간 공유되는 인프라스트럭처 에러 타입을 포함한다.
package errors

import (
	"errors"
	"fmt"
)

// RedisError: Redis 작업을 수행하는 도중 발생한 에러
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

// DatabaseError: 데이터베이스(PostgreSQL 등) 작업을 수행하는 도중 발생한 에러
type DatabaseError struct {
	Operation string
	Err       error
}

func (e DatabaseError) Error() string {
	if e.Err == nil {
		return fmt.Sprintf("db error operation=%s", e.Operation)
	}
	return fmt.Sprintf("db error operation=%s: %v", e.Operation, e.Err)
}

func (e DatabaseError) Unwrap() error { return e.Err }

// LockError: 분산 락 획득 실패 등 락 관련 처리 중 발생하는 에러
type LockError struct {
	SessionID   string // 세션 ID 또는 ChatID
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

// AccessDeniedError: 접근 권한이 없을 때 발생하는 에러
type AccessDeniedError struct {
	Reason string
}

func (e AccessDeniedError) Error() string {
	if e.Reason == "" {
		return "access denied"
	}
	return fmt.Sprintf("access denied: %s", e.Reason)
}

// UserBlockedError: 차단된 사용자의 접근 시 발생하는 에러
type UserBlockedError struct {
	UserID string
}

func (e UserBlockedError) Error() string { return fmt.Sprintf("user blocked: %s", e.UserID) }

// ChatBlockedError: 차단된 채팅방의 접근 시 발생하는 에러
type ChatBlockedError struct {
	ChatID string
}

func (e ChatBlockedError) Error() string { return fmt.Sprintf("chat blocked: %s", e.ChatID) }

// InputInjectionError: 프롬프트 인젝션 의심 입력 시 발생하는 에러
type InputInjectionError struct {
	Message string
}

func (e InputInjectionError) Error() string { return e.Message }

// MalformedInputError: 입력 형식이 올바르지 않을 때 발생하는 에러
type MalformedInputError struct {
	Message string
}

func (e MalformedInputError) Error() string { return e.Message }

// InvalidQuestionError: 부적절한 형식이나 내용의 질문일 때 발생하는 에러
type InvalidQuestionError struct {
	Message string
}

func (e InvalidQuestionError) Error() string {
	if e.Message == "" {
		return "invalid question"
	}
	return "invalid question: " + e.Message
}

// InvalidAnswerError: 부적절한 형식의 정답 제출일 때 발생하는 에러
type InvalidAnswerError struct {
	Message string
}

func (e InvalidAnswerError) Error() string { return e.Message }

// expectedUserBehaviorTypes: 사용자의 정상적인 패턴 내 실수로 간주되는 에러 타입들
// IsExpectedUserBehavior 함수에서 공통으로 체크하는 타입 리스트
var expectedUserBehaviorTypes = []func() any{
	func() any { return new(InvalidQuestionError) },
	func() any { return new(InvalidAnswerError) },
	func() any { return new(MalformedInputError) },
}

// IsExpectedUserBehavior: 에러가 사용자의 정상적인(예상된) 패턴 내의 실수인지 확인한다.
// (로그 레벨을 낮추거나 사용자에게 친절한 메시지를 보내는 용도)
// 공통 에러 타입만 체크하며, 도메인 특화 에러는 각 패키지에서 확장하여 사용한다.
func IsExpectedUserBehavior(err error) bool {
	if err == nil {
		return false
	}
	for _, targetFn := range expectedUserBehaviorTypes {
		if errors.As(err, targetFn()) {
			return true
		}
	}
	return false
}
