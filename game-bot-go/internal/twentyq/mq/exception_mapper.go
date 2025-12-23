package mq

import (
	"context"
	"errors"

	"github.com/park285/llm-kakao-bots/game-bot-go/internal/common/messageprovider"
	qerrors "github.com/park285/llm-kakao-bots/game-bot-go/internal/twentyq/errors"
	qmessages "github.com/park285/llm-kakao-bots/game-bot-go/internal/twentyq/messages"
)

// ErrorMapping 는 타입이다.
type ErrorMapping struct {
	Key    string
	Params []messageprovider.Param
}

// GetErrorMapping 는 동작을 수행한다.
func GetErrorMapping(err error, commandPrefix string) ErrorMapping {
	var (
		sessionNotFound qerrors.SessionNotFoundError
		invalidQuestion qerrors.InvalidQuestionError
		duplicate       qerrors.DuplicateQuestionError
		hintLimit       qerrors.HintLimitExceededError
		hintNA          qerrors.HintNotAvailableError
	)

	switch {
	case errors.As(err, &sessionNotFound):
		return ErrorMapping{
			Key: qmessages.ErrorNoSession,
			Params: []messageprovider.Param{
				messageprovider.P("prefix", commandPrefix),
			},
		}
	case errors.As(err, &duplicate):
		return ErrorMapping{Key: qmessages.ErrorDuplicateQuestion}
	case errors.As(err, &invalidQuestion):
		return ErrorMapping{Key: qmessages.ErrorInvalidQuestion}
	case errors.As(err, &hintLimit):
		return ErrorMapping{
			Key: qmessages.ErrorHintLimitExceeded,
			Params: []messageprovider.Param{
				messageprovider.P("maxHints", hintLimit.MaxHints),
				messageprovider.P("hintCount", hintLimit.HintCount),
				messageprovider.P("remaining", hintLimit.Remaining),
			},
		}
	case errors.As(err, &hintNA):
		return ErrorMapping{Key: qmessages.ErrorHintNotAvailable}
	case errors.Is(err, context.DeadlineExceeded):
		return ErrorMapping{Key: qmessages.ErrorAITimeout}
	default:
		return ErrorMapping{Key: qmessages.ErrorGeneric}
	}
}
