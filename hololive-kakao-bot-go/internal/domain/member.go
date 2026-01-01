package domain

import (
	"context"
	_ "embed" // 멤버 데이터 임베드용
	"fmt"

	"github.com/goccy/go-json"
)

// Aliases: 멤버의 국가별(한국어, 일본어) 별명 목록
type Aliases struct {
	Ko []string `json:"ko"`
	Ja []string `json:"ja"`
}

// Member: Hololive 멤버의 기본 정보(ID, 채널, 이름 등)를 담는 구조체
type Member struct {
	ID          int      `json:"id,omitempty"`
	ChannelID   string   `json:"channelId"`
	Name        string   `json:"name"`
	Aliases     *Aliases `json:"aliases,omitempty"`
	NameJa      string   `json:"nameJa,omitempty"`
	NameKo      string   `json:"nameKo,omitempty"`
	IsGraduated bool     `json:"isGraduated,omitempty"`
}

// MembersData: 전체 멤버 데이터의 메타데이터 및 목록, 빠른 조회를 위한 맵(Map)을 포함합니다.
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

// GetAllAliases: 멤버의 한국어 및 일본어 별명을 모두 합쳐 하나의 슬라이스로 반환합니다.
func (m *Member) GetAllAliases() []string {
	if m.Aliases == nil {
		return []string{}
	}

	all := make([]string, 0, len(m.Aliases.Ko)+len(m.Aliases.Ja))
	all = append(all, m.Aliases.Ko...)
	all = append(all, m.Aliases.Ja...)
	return all
}

// HasAlias: 주어진 이름이 해당 멤버의 별명 목록에 포함되어 있는지 확인합니다.
func (m *Member) HasAlias(name string) bool {
	aliases := m.GetAllAliases()
	for _, alias := range aliases {
		if alias == name {
			return true
		}
	}
	return false
}

// LoadMembersData: 임베딩된 JSON 데이터(members.json)를 파싱하여 MembersData 구조체를 생성하고 초기화합니다.
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

// FindMemberByChannelID: 채널 ID로 멤버를 검색하여 반환한다. (O(1) 조회)
func (md *MembersData) FindMemberByChannelID(channelID string) *Member {
	return md.byChannelID[channelID]
}

// FindMemberByName: 멤버 이름으로 검색하여 반환한다. (O(1) 조회)
func (md *MembersData) FindMemberByName(name string) *Member {
	return md.byName[name]
}

// FindMemberByAlias: 주어진 별명을 가진 멤버를 검색하여 반환한다. (선형 탐색)
func (md *MembersData) FindMemberByAlias(alias string) *Member {
	for _, member := range md.Members {
		if member.HasAlias(alias) {
			return member
		}
	}
	return nil
}

// GetChannelIDs: 등록된 모든 멤버의 채널 ID 목록을 추출하여 반환합니다.
func (md *MembersData) GetChannelIDs() []string {
	ids := make([]string, len(md.Members))
	for i, member := range md.Members {
		ids[i] = member.ChannelID
	}
	return ids
}

// GetAllMembers: 전체 멤버 목록을 반환합니다.
func (md *MembersData) GetAllMembers() []*Member {
	return md.Members
}

// WithContext: 현재 데이터 제공자를 새로운 컨텍스트와 함께 반환한다. (MembersData는 상태가 불변이므로 자신을 그대로 반환)
func (md *MembersData) WithContext(ctx context.Context) MemberDataProvider {
	return md
}
