package security

import (
	"github.com/park285/llm-kakao-bots/game-bot-go/internal/common/accesscontrol"
	"github.com/park285/llm-kakao-bots/game-bot-go/internal/common/ptr"
	qconfig "github.com/park285/llm-kakao-bots/game-bot-go/internal/twentyq/config"
	qmessages "github.com/park285/llm-kakao-bots/game-bot-go/internal/twentyq/messages"
)

// AccessControl: 20고개 게임에 대한 사용자 및 채팅방 접근 제어 관리자
type AccessControl struct {
	control *accesscontrol.AccessControl
}

// NewAccessControl: 새로운 AccessControl 인스턴스를 생성합니다.
func NewAccessControl(cfg qconfig.AccessConfig) *AccessControl {
	return &AccessControl{control: accesscontrol.New(cfg)}
}

// GetDenialReason: 접근 거부 사유에 따른 오류 메시지를 반환합니다.
// 접근이 허용된 경우 nil을 반환합니다.
func (a *AccessControl) GetDenialReason(userID string, chatID string) *string {
	msg, ok := accesscontrol.DenialReasonMessage(
		a.control.DenialReason(userID, chatID),
		accesscontrol.DenialReasonMessages{
			UserBlocked:  qmessages.ErrorUserBlocked,
			ChatBlocked:  qmessages.ErrorChatBlocked,
			AccessDenied: qmessages.ErrorAccessDenied,
		},
	)
	if !ok {
		return nil
	}
	return ptr.String(msg)
}
