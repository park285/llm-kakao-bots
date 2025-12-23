package mq

import (
	"github.com/park285/llm-kakao-bots/game-bot-go/internal/common/ptr"
	tsmessages "github.com/park285/llm-kakao-bots/game-bot-go/internal/turtlesoup/messages"
)

// CommandKind 는 타입이다.
type CommandKind int

// CommandKind 상수 목록.
const (
	// CommandStart 는 상수다.
	CommandStart CommandKind = iota
	CommandAsk
	CommandAnswer
	CommandHint
	CommandProblem
	CommandSurrender
	CommandAgree
	CommandSummary
	CommandHelp
	CommandUnknown
)

// Command 는 타입이다.
type Command struct {
	Kind            CommandKind
	Difficulty      *int
	HasInvalidInput bool
	Question        string
	Answer          string
}

// RequiresLock 는 동작을 수행한다.
func (c Command) RequiresLock() bool {
	switch c.Kind {
	case CommandHelp, CommandUnknown:
		return false
	default:
		return true
	}
}

// WaitingMessageKey 는 동작을 수행한다.
func (c Command) WaitingMessageKey() *string {
	switch c.Kind {
	case CommandStart:
		return ptr.String(tsmessages.StartWaiting)
	case CommandAsk:
		return ptr.String(tsmessages.ProcessingThinking)
	case CommandHint:
		return ptr.String(tsmessages.ProcessingGeneratingHint)
	case CommandAnswer:
		return ptr.String(tsmessages.ProcessingValidating)
	default:
		return nil
	}
}
