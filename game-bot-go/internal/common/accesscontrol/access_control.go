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

// DenialReasonMessages: DenialReason에 대응하는 사용자 표시 메시지 묶음입니다.
type DenialReasonMessages struct {
	UserBlocked  string
	ChatBlocked  string
	AccessDenied string
}

// DenialReasonMessage: DenialReason에 해당하는 메시지를 반환합니다.
// reason이 DenialReasonNone이거나 메시지가 비어있으면 ok=false를 반환합니다.
func DenialReasonMessage(reason DenialReason, messages DenialReasonMessages) (msg string, ok bool) {
	switch reason {
	case DenialReasonUserBlocked:
		if messages.UserBlocked == "" {
			return "", false
		}
		return messages.UserBlocked, true
	case DenialReasonChatBlocked:
		if messages.ChatBlocked == "" {
			return "", false
		}
		return messages.ChatBlocked, true
	case DenialReasonAccessDenied:
		if messages.AccessDenied == "" {
			return "", false
		}
		return messages.AccessDenied, true
	default:
		return "", false
	}
}

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
