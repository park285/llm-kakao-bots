package mq

import (
	"errors"

	cerrors "github.com/park285/llm-kakao-bots/game-bot-go/internal/common/errors"
	"github.com/park285/llm-kakao-bots/game-bot-go/internal/common/messageprovider"
	tsconfig "github.com/park285/llm-kakao-bots/game-bot-go/internal/turtlesoup/config"
	tserrors "github.com/park285/llm-kakao-bots/game-bot-go/internal/turtlesoup/errors"
	tsmessages "github.com/park285/llm-kakao-bots/game-bot-go/internal/turtlesoup/messages"
)

// ErrorMapping: 에러 메시지 키와 포맷팅 파라미터를 포함하는 매핑 구조체
type ErrorMapping struct {
	Key    string
	Params []messageprovider.Param
}

// GetErrorMapping: 발생한 에러를 분석하여 사용자에게 보여줄 적절한 메시지 매핑을 반환한다.
func GetErrorMapping(err error) ErrorMapping {
	var (
		sessionNotFound  *tserrors.SessionNotFoundError
		invalidQuestion  *cerrors.InvalidQuestionError
		invalidAnswer    *cerrors.InvalidAnswerError
		maxHints         *tserrors.MaxHintsReachedError
		gameAlreadyStart *tserrors.GameAlreadyStartedError
		gameSolved       *tserrors.GameAlreadySolvedError
		puzzleGen        *tserrors.PuzzleGenerationError
		accessDenied     *cerrors.AccessDeniedError
		userBlocked      *cerrors.UserBlockedError
		chatBlocked      *cerrors.ChatBlockedError
	)

	switch {
	case errors.As(err, &sessionNotFound):
		return ErrorMapping{Key: tsmessages.ErrorNoSession}
	case errors.As(err, &invalidQuestion):
		return ErrorMapping{
			Key: tsmessages.ErrorInvalidQuestion,
			Params: []messageprovider.Param{
				messageprovider.P("minLength", tsconfig.ValidationMinQuestionLength),
				messageprovider.P("maxLength", tsconfig.ValidationMaxQuestionLength),
			},
		}
	case errors.As(err, &invalidAnswer):
		return ErrorMapping{
			Key: tsmessages.ErrorInvalidAnswer,
			Params: []messageprovider.Param{
				messageprovider.P("minLength", tsconfig.ValidationMinAnswerLength),
				messageprovider.P("maxLength", tsconfig.ValidationMaxAnswerLength),
			},
		}
	case errors.As(err, &maxHints):
		return ErrorMapping{
			Key: tsmessages.ErrorMaxHints,
			Params: []messageprovider.Param{
				messageprovider.P("maxHints", tsconfig.GameMaxHints),
			},
		}
	case errors.As(err, &gameAlreadyStart):
		return ErrorMapping{Key: tsmessages.ErrorGameAlreadyStarted}
	case errors.As(err, &gameSolved):
		return ErrorMapping{Key: tsmessages.ErrorGameAlreadySolved}
	case errors.As(err, &puzzleGen):
		return ErrorMapping{Key: tsmessages.ErrorPuzzleGeneration}
	case errors.As(err, &accessDenied):
		return ErrorMapping{Key: tsmessages.ErrorAccessDenied}
	case errors.As(err, &userBlocked):
		return ErrorMapping{Key: tsmessages.ErrorUserBlocked}
	case errors.As(err, &chatBlocked):
		return ErrorMapping{Key: tsmessages.ErrorChatBlocked}
	default:
		return ErrorMapping{Key: tsmessages.ErrorInternal}
	}
}
