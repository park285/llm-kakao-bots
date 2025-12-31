package security

import (
	"github.com/park285/llm-kakao-bots/game-bot-go/internal/common/accesscontrol"
	"github.com/park285/llm-kakao-bots/game-bot-go/internal/common/ptr"
	tsconfig "github.com/park285/llm-kakao-bots/game-bot-go/internal/turtlesoup/config"
	tsmessages "github.com/park285/llm-kakao-bots/game-bot-go/internal/turtlesoup/messages"
)

// AccessControl: 바다거북스프 게임의 사용자/채팅방 접근을 설정 기반으로 제어합니다.
// 운영 환경에서 허용 채팅방/차단 사용자 정책을 일관되게 적용하기 위함입니다.
type AccessControl struct {
	control *accesscontrol.AccessControl
}

// NewAccessControl: 새로운 AccessControl 인스턴스를 생성합니다.
func NewAccessControl(cfg tsconfig.AccessConfig) *AccessControl {
	return &AccessControl{control: accesscontrol.New(cfg)}
}

// GetDenialReason: 접근 거부 사유에 따른 오류 메시지를 반환합니다.
// 접근이 허용된 경우 nil을 반환합니다.
func (a *AccessControl) GetDenialReason(userID string, chatID string) *string {
	msg, ok := accesscontrol.DenialReasonMessage(
		a.control.DenialReason(userID, chatID),
		accesscontrol.DenialReasonMessages{
			UserBlocked:  tsmessages.ErrorUserBlocked,
			ChatBlocked:  tsmessages.ErrorChatBlocked,
			AccessDenied: tsmessages.ErrorAccessDenied,
		},
	)
	if !ok {
		return nil
	}
	return ptr.String(msg)
}
