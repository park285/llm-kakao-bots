package domain

import (
	_ "embed" // 인재 데이터 임베드용
	"encoding/json"
	"fmt"
	"net/url"
	"path"
	"strings"

	"github.com/kapu/hololive-kakao-bot-go/internal/util"
)

// OfficialTalent 는 타입이다.
type OfficialTalent struct {
	Japanese string `json:"japanese"`
	English  string `json:"english"`
	Link     string `json:"link"`
	Status   string `json:"status"`
}

// Talents 는 타입이다.
type Talents struct {
	Talents   []*OfficialTalent
	byEnglish map[string]*OfficialTalent
}

//go:embed data/official_talents.json
var officialTalentsJSON []byte

// LoadTalents 는 동작을 수행한다.
func LoadTalents() (*Talents, error) {
	var talents []*OfficialTalent
	if err := json.Unmarshal(officialTalentsJSON, &talents); err != nil {
		return nil, fmt.Errorf("failed to unmarshal talents data: %w", err)
	}

	index := make(map[string]*OfficialTalent, len(talents))
	for _, talent := range talents {
		if talent == nil {
			continue
		}
		index[util.Normalize(talent.English)] = talent
	}

	return &Talents{
		Talents:   talents,
		byEnglish: index,
	}, nil
}

// FindByEnglish 는 동작을 수행한다.
func (ot *Talents) FindByEnglish(name string) *OfficialTalent {
	if ot == nil {
		return nil
	}
	return ot.byEnglish[util.Normalize(name)]
}

// Slug 는 동작을 수행한다.
func (ot *OfficialTalent) Slug() string {
	if ot == nil {
		return ""
	}

	if ot.Link == "" {
		return ""
	}

	u, err := url.Parse(ot.Link)
	if err == nil {
		segment := strings.Trim(path.Base(u.Path), "/")
		if segment != "" && segment != "." && segment != "/" {
			return segment
		}
	}

	fallback := util.Normalize(ot.English)
	fallback = strings.ReplaceAll(fallback, " ", "-")
	fallback = strings.ReplaceAll(fallback, "'", "")
	fallback = strings.ReplaceAll(fallback, ".", "")
	return fallback
}
