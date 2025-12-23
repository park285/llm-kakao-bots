package domain

import (
	"github.com/kapu/hololive-kakao-bot-go/internal/util"
)

// MemberIntentType 는 타입이다.
type MemberIntentType string

// MemberIntentType 상수 목록.
const (
	// MemberIntentUnknown 는 상수다.
	MemberIntentUnknown    MemberIntentType = "unknown"
	MemberIntentMemberInfo MemberIntentType = "member_info"
	MemberIntentOther      MemberIntentType = "other"
)

// MemberIntent 는 타입이다.
type MemberIntent struct {
	Intent     MemberIntentType `json:"intent"`
	Confidence float64          `json:"confidence"`
	Reasoning  string           `json:"reasoning"`
}

// NormalizeMemberIntent 는 동작을 수행한다.
func NormalizeMemberIntent(raw string) MemberIntentType {
	switch util.Normalize(raw) {
	case string(MemberIntentMemberInfo):
		return MemberIntentMemberInfo
	case string(MemberIntentOther):
		return MemberIntentOther
	default:
		return MemberIntentUnknown
	}
}

// IsMemberInfoIntent 는 동작을 수행한다.
func (mic *MemberIntent) IsMemberInfoIntent() bool {
	if mic == nil {
		return false
	}
	intent := NormalizeMemberIntent(string(mic.Intent))
	if intent != MemberIntentMemberInfo {
		return false
	}
	return mic.Confidence >= 0.35
}
