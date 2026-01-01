package mq

import (
	"github.com/park285/llm-kakao-bots/game-bot-go/internal/common/ptr"
	tsmessages "github.com/park285/llm-kakao-bots/game-bot-go/internal/turtlesoup/messages"
)

// CommandKind: 명령어의 종류를 나타내는 열거형
type CommandKind int

// CommandKind 상수 목록.
const (
	// CommandStart: 게임 시작 명령어 (새로운 퍼즐 생성)
	CommandStart CommandKind = iota
	// CommandAsk: "예/아니오"로 대답할 수 있는 질문 하기
	CommandAsk
	// CommandAnswer: 정답 맞추기 시도
	CommandAnswer
	// CommandHint: 힌트 요청
	CommandHint
	// CommandProblem: 현재 문제(상황) 다시 보기
	CommandProblem
	// CommandSurrender: 게임 포기(항복) 투표 시작 또는 찬성
	CommandSurrender
	// CommandAgree: 항복 투표에 찬성 (별칭)
	CommandAgree
	// CommandSummary: 지금까지의 질문과 답변 요약 보기
	CommandSummary
	// CommandHelp: 도움말 보기
	CommandHelp
	// CommandUnknown: 알 수 없는 명령어
	CommandUnknown
)

// Command: 사용자 입력을 파싱하여 정제된 명령어 정보를 담는 구조체
type Command struct {
	Kind            CommandKind
	Difficulty      *int
	HasInvalidInput bool
	Question        string
	Answer          string
}

// RequiresLock: 이 명령어를 실행할 때 게임 상태 보호를 위한 분산 락(Write Lock)이 필요한지 여부를 반환합니다.
// 단순 조회나 도움말 등은 락이 필요 없습니다.
func (c Command) RequiresLock() bool {
	switch c.Kind {
	case CommandHelp, CommandUnknown:
		return false
	default:
		return true
	}
}

// WaitingMessageKey: 명령어가 처리되는 동안(AI 생성 등) 사용자에게 보여줄 '처리 중...' 메시지 키를 반환합니다.
// 즉시 처리되는 명령어의 경우 nil을 반환합니다.
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
