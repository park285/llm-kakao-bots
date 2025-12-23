package domain

import (
	"context"
	_ "embed" // 멤버 데이터 임베드용
	"encoding/json"
	"fmt"
)

// Aliases 는 타입이다.
type Aliases struct {
	Ko []string `json:"ko"`
	Ja []string `json:"ja"`
}

// Member 는 타입이다.
type Member struct {
	ID          int      `json:"id,omitempty"`
	ChannelID   string   `json:"channelId"`
	Name        string   `json:"name"`
	Aliases     *Aliases `json:"aliases,omitempty"`
	NameJa      string   `json:"nameJa,omitempty"`
	NameKo      string   `json:"nameKo,omitempty"`
	IsGraduated bool     `json:"isGraduated,omitempty"`
}

// MembersData 는 타입이다.
type MembersData struct {
	Version     string    `json:"version"`
	LastUpdated string    `json:"lastUpdated"`
	Sources     []string  `json:"sources"`
	Members     []*Member `json:"members"`

	byChannelID map[string]*Member
	byName      map[string]*Member
}

//go:embed data/members.json
var membersJSON []byte

// GetAllAliases 는 동작을 수행한다.
func (m *Member) GetAllAliases() []string {
	if m.Aliases == nil {
		return []string{}
	}

	all := make([]string, 0, len(m.Aliases.Ko)+len(m.Aliases.Ja))
	all = append(all, m.Aliases.Ko...)
	all = append(all, m.Aliases.Ja...)
	return all
}

// HasAlias 는 동작을 수행한다.
func (m *Member) HasAlias(name string) bool {
	aliases := m.GetAllAliases()
	for _, alias := range aliases {
		if alias == name {
			return true
		}
	}
	return false
}

// LoadMembersData 는 동작을 수행한다.
func LoadMembersData() (*MembersData, error) {
	var data MembersData
	if err := json.Unmarshal(membersJSON, &data); err != nil {
		return nil, fmt.Errorf("failed to unmarshal members data: %w", err)
	}

	data.byChannelID = make(map[string]*Member, len(data.Members))
	data.byName = make(map[string]*Member, len(data.Members))

	for _, member := range data.Members {
		data.byChannelID[member.ChannelID] = member
		data.byName[member.Name] = member
	}

	return &data, nil
}

// FindMemberByChannelID 는 동작을 수행한다.
func (md *MembersData) FindMemberByChannelID(channelID string) *Member {
	return md.byChannelID[channelID]
}

// FindMemberByName 는 동작을 수행한다.
func (md *MembersData) FindMemberByName(name string) *Member {
	return md.byName[name]
}

// FindMemberByAlias 는 동작을 수행한다.
func (md *MembersData) FindMemberByAlias(alias string) *Member {
	for _, member := range md.Members {
		if member.HasAlias(alias) {
			return member
		}
	}
	return nil
}

// GetChannelIDs 는 동작을 수행한다.
func (md *MembersData) GetChannelIDs() []string {
	ids := make([]string, len(md.Members))
	for i, member := range md.Members {
		ids[i] = member.ChannelID
	}
	return ids
}

// GetAllMembers 는 동작을 수행한다.
func (md *MembersData) GetAllMembers() []*Member {
	return md.Members
}

// WithContext 는 동작을 수행한다.
func (md *MembersData) WithContext(ctx context.Context) MemberDataProvider {
	return md
}
