package valkeyx

import (
	cerrors "github.com/park285/llm-kakao-bots/game-bot-go/internal/common/errors"
)

// WrapRedisError: Redis 관련 에러를 공통 타입으로 감싼다.
func WrapRedisError(operation string, err error) error {
	if err == nil {
		return nil
	}
	return cerrors.RedisError{Operation: operation, Err: err}
}
