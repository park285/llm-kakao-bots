package security

import (
	"github.com/park285/llm-kakao-bots/game-bot-go/internal/common/accesscontrol"
	"github.com/park285/llm-kakao-bots/game-bot-go/internal/common/ptr"
	tsconfig "github.com/park285/llm-kakao-bots/game-bot-go/internal/turtlesoup/config"
	tsmessages "github.com/park285/llm-kakao-bots/game-bot-go/internal/turtlesoup/messages"
)

// AccessControl 는 타입이다.
type AccessControl struct {
	control *accesscontrol.AccessControl
}

// NewAccessControl 는 동작을 수행한다.
func NewAccessControl(cfg tsconfig.AccessConfig) *AccessControl {
	return &AccessControl{control: accesscontrol.New(cfg)}
}

// GetDenialReason 는 동작을 수행한다.
func (a *AccessControl) GetDenialReason(userID string, chatID string) *string {
	switch a.control.DenialReason(userID, chatID) {
	case accesscontrol.DenialReasonUserBlocked:
		return ptr.String(tsmessages.ErrorUserBlocked)
	case accesscontrol.DenialReasonChatBlocked:
		return ptr.String(tsmessages.ErrorChatBlocked)
	case accesscontrol.DenialReasonAccessDenied:
		return ptr.String(tsmessages.ErrorAccessDenied)
	default:
		return nil
	}
}
