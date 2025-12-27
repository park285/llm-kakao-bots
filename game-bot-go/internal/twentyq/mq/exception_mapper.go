package mq

import (
	"context"
	"errors"

	cerrors "github.com/park285/llm-kakao-bots/game-bot-go/internal/common/errors"
	"github.com/park285/llm-kakao-bots/game-bot-go/internal/common/messageprovider"
	qerrors "github.com/park285/llm-kakao-bots/game-bot-go/internal/twentyq/errors"
	qmessages "github.com/park285/llm-kakao-bots/game-bot-go/internal/twentyq/messages"
)

// ErrorMapping: 사용자에게 보여줄 에러 메시지 키와 포맷팅 파라미터를 담는 구조체
type ErrorMapping struct {
	Key    string
	Params []messageprovider.Param
}

// GetErrorMapping: 발생한 에러를 분석하여 사용자에게 전달할 적절한 메시지 키와 파라미터를 매핑하여 반환한다.
func GetErrorMapping(err error, commandPrefix string) ErrorMapping {
	var (
		sessionNotFound qerrors.SessionNotFoundError
		invalidQuestion cerrors.InvalidQuestionError
		duplicate       qerrors.DuplicateQuestionError
		hintLimit       qerrors.HintLimitExceededError
		hintNA          qerrors.HintNotAvailableError
		guessRateLimit  qerrors.GuessRateLimitError
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
	case errors.As(err, &guessRateLimit):
		return ErrorMapping{
			Key: qmessages.ErrorGuessRateLimit,
			Params: []messageprovider.Param{
				messageprovider.P("remainingSeconds", guessRateLimit.RemainingSeconds),
				messageprovider.P("totalSeconds", guessRateLimit.TotalSeconds),
			},
		}
	case errors.Is(err, context.DeadlineExceeded):
		return ErrorMapping{Key: qmessages.ErrorAITimeout}
	default:
		return ErrorMapping{Key: qmessages.ErrorGeneric}
	}
}
