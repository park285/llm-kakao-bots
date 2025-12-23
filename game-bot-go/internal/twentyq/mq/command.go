package mq

import (
	"github.com/park285/llm-kakao-bots/game-bot-go/internal/common/ptr"
	qmessages "github.com/park285/llm-kakao-bots/game-bot-go/internal/twentyq/messages"
	qmodel "github.com/park285/llm-kakao-bots/game-bot-go/internal/twentyq/model"
)

// CommandKind 는 타입이다.
type CommandKind int

// CommandStart 는 명령 종류 상수 목록이다.
const (
	CommandStart CommandKind = iota
	CommandHints
	CommandAsk
	CommandChainedQuestion // 체인 질문 (쉼표로 구분된 여러 질문)
	CommandSurrender
	CommandAgree
	CommandReject
	CommandStatus
	CommandModelInfo
	CommandHelp
	CommandUnknown

	// 전적 조회

	// CommandUserStats 는 사용자 전적 조회 명령이다.
	CommandUserStats
	CommandRoomStats

	// 관리자 명령어

	// CommandAdminForceEnd 는 관리자 강제 종료 명령이다.
	CommandAdminForceEnd
	CommandAdminClearAll
	CommandAdminUsage
)

// Command 는 타입이다.
type Command struct {
	Kind       CommandKind
	Categories []string
	HintCount  int
	Question   string
	// 체인 질문용
	ChainQuestions []string              // 쉼표로 구분된 질문 목록
	ChainCondition qmodel.ChainCondition // 실행 조건 (ALWAYS, IF_TRUE)
	// 전적 조회용
	TargetNickname *string            // 다른 사용자 전적 조회 시
	RoomPeriod     qmodel.StatsPeriod // 룸 전적 기간
	// 사용량 조회용
	UsagePeriod   qmodel.UsagePeriod
	ModelOverride *string
}

// RequiresWriteLock 는 동작을 수행한다.
func (c Command) RequiresWriteLock() bool {
	switch c.Kind {
	case CommandStatus,
		CommandUserStats, CommandRoomStats, CommandAdminUsage:
		return false
	default:
		return true
	}
}

// WaitingMessageKey 는 동작을 수행한다.
func (c Command) WaitingMessageKey() *string {
	switch c.Kind {
	case CommandStart:
		return ptr.String(qmessages.StartWaiting)
	case CommandHints:
		return ptr.String(qmessages.HintWaiting)
	case CommandAsk:
		return ptr.String(qmessages.ProcessingWaiting)
	default:
		return nil
	}
}
