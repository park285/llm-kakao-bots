package accesscontrol

import (
	"slices"

	commonconfig "github.com/park285/llm-kakao-bots/game-bot-go/internal/common/config"
)

// DenialReason: 접근 거부 사유 열거형
type DenialReason int

// DenialReason 상수 목록.
const (
	// DenialReasonNone: 거부되지 않음 (허용)
	DenialReasonNone DenialReason = iota
	DenialReasonUserBlocked
	DenialReasonChatBlocked
	DenialReasonAccessDenied
)

// AccessControl: 설정 기반의 사용자/채팅방 접근 제어 관리자
type AccessControl struct {
	cfg commonconfig.AccessConfig
}

// New: 새로운 AccessControl 인스턴스를 생성합니다.
func New(cfg commonconfig.AccessConfig) *AccessControl {
	return &AccessControl{cfg: cfg}
}

// DenialReason: 사용자 및 채팅방의 접근 거부 사유를 반환합니다. (허용 시 DenialReasonNone)
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
