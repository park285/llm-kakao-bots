package domain

import "context"

// MemberDataProvider 는 타입이다.
type MemberDataProvider interface {
	FindMemberByChannelID(channelID string) *Member
	FindMemberByName(name string) *Member
	FindMemberByAlias(alias string) *Member
	GetChannelIDs() []string
	GetAllMembers() []*Member // For iteration (legacy compatibility)
	WithContext(ctx context.Context) MemberDataProvider
}

var _ MemberDataProvider = (*MembersData)(nil)
