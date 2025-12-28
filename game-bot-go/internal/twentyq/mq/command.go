package mq

import (
	"github.com/park285/llm-kakao-bots/game-bot-go/internal/common/ptr"
	qmessages "github.com/park285/llm-kakao-bots/game-bot-go/internal/twentyq/messages"
	qmodel "github.com/park285/llm-kakao-bots/game-bot-go/internal/twentyq/model"
)

// CommandKind: 스무고개 게임 명령어의 종류를 정의하는 열거형
type CommandKind int

// CommandStart: 게임 시작 명령어
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

	// CommandUserStats: 사용자 전적 조회 명령
	CommandUserStats
	CommandRoomStats

	// 관리자 명령어

	// CommandAdminForceEnd: 관리자 강제 종료 명령
	CommandAdminForceEnd
	CommandAdminClearAll
	CommandAdminUsage
)

// Command: 사용자 입력에서 파싱된 게임 명령어 정보를 담는 구조체
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

// WaitingMessageKey: 명령어를 처리하는 동안 사용자에게 즉시 보여줄 '대기 중' 메시지의 키를 반환한다.
// 반환값이 nil이면 별도의 대기 메시지를 보내지 않는다.
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
