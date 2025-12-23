package accesscontrol

import (
	"slices"

	commonconfig "github.com/park285/llm-kakao-bots/game-bot-go/internal/common/config"
)

// DenialReason 는 타입이다.
type DenialReason int

// DenialReason 상수 목록.
const (
	// DenialReasonNone 는 상수다.
	DenialReasonNone DenialReason = iota
	DenialReasonUserBlocked
	DenialReasonChatBlocked
	DenialReasonAccessDenied
)

// AccessControl 는 타입이다.
type AccessControl struct {
	cfg commonconfig.AccessConfig
}

// New 는 동작을 수행한다.
func New(cfg commonconfig.AccessConfig) *AccessControl {
	return &AccessControl{cfg: cfg}
}

// DenialReason 는 동작을 수행한다.
func (a *AccessControl) DenialReason(userID string, chatID string) DenialReason {
	if a == nil {
		return DenialReasonNone
	}
	if a.cfg.Passthrough {
		return DenialReasonNone
	}
	if slices.Contains(a.cfg.BlockedUserIDs, userID) {
		return DenialReasonUserBlocked
	}
	if !a.cfg.Enabled {
		return DenialReasonNone
	}
	if slices.Contains(a.cfg.BlockedChatIDs, chatID) {
		return DenialReasonChatBlocked
	}

	if len(a.cfg.AllowedChatIDs) == 0 {
		return DenialReasonNone
	}
	if slices.Contains(a.cfg.AllowedChatIDs, chatID) {
		return DenialReasonNone
	}
	return DenialReasonAccessDenied
}
