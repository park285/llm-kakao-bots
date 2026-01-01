package adapter

import (
	"fmt"
	"strings"

	"github.com/kapu/hololive-kakao-bot-go/internal/domain"
	"github.com/kapu/hololive-kakao-bot-go/internal/util"
)

// 번역된 값 우선 사용, 없으면 원본 반환
func getTranslatedText(translatedVal, rawVal string) string {
	if trimmed := util.TrimSpace(translatedVal); trimmed != "" {
		return trimmed
	}
	return util.TrimSpace(rawVal)
}

// 캐치프레이즈 섹션 포맷팅
func formatProfileCatchphrase(raw *domain.TalentProfile, translated *domain.Translated) string {
	catchphrase := ""
	if translated != nil {
		catchphrase = getTranslatedText(translated.Catchphrase, raw.Catchphrase)
	} else if raw != nil {
		catchphrase = util.TrimSpace(raw.Catchphrase)
	}

	if catchphrase == "" {
		return ""
	}
	return fmt.Sprintf("%s %s\n", DefaultEmoji.Speech, catchphrase)
}

// 요약 섹션 포맷팅
func formatProfileSummary(raw *domain.TalentProfile, translated *domain.Translated) string {
	summary := ""
	if translated != nil {
		summary = getTranslatedText(translated.Summary, raw.Description)
	} else if raw != nil {
		summary = util.TrimSpace(raw.Description)
	}

	if summary == "" {
		return ""
	}
	return summary + "\n"
}

// 하이라이트 섹션 포맷팅
func formatProfileHighlights(translated *domain.Translated) string {
	if translated == nil || len(translated.Highlights) == 0 {
		return ""
	}

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("\n%s 하이라이트\n", DefaultEmoji.Highlight))
	for _, highlight := range translated.Highlights {
		if trimmed := util.TrimSpace(highlight); trimmed != "" {
			sb.WriteString(fmt.Sprintf("- %s\n", trimmed))
		}
	}
	return sb.String()
}

// 번역된 데이터 또는 원본 데이터 반환
func getProfileDataEntries(raw *domain.TalentProfile, translated *domain.Translated) []domain.TranslatedProfileDataRow {
	if translated != nil && len(translated.Data) > 0 {
		return translated.Data
	}

	if raw == nil || len(raw.DataEntries) == 0 {
		return nil
	}

	entries := make([]domain.TranslatedProfileDataRow, 0)
	for _, entry := range raw.DataEntries {
		if util.TrimSpace(entry.Label) == "" || util.TrimSpace(entry.Value) == "" {
			continue
		}
		entries = append(entries, domain.TranslatedProfileDataRow(entry))
	}
	return entries
}

// 프로필 데이터 섹션 포맷팅 (최대 8개)
func formatProfileDataEntries(raw *domain.TalentProfile, translated *domain.Translated) string {
	dataEntries := getProfileDataEntries(raw, translated)
	if len(dataEntries) == 0 {
		return ""
	}

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("\n%s 프로필 데이터\n", DefaultEmoji.Data))

	maxRows := len(dataEntries)
	if maxRows > 8 {
		maxRows = 8
	}

	for i := 0; i < maxRows; i++ {
		row := dataEntries[i]
		label := util.TrimSpace(row.Label)
		value := util.TrimSpace(row.Value)
		if label == "" || value == "" {
			continue
		}

		if strings.Contains(value, "\n") {
			indented := "  " + strings.ReplaceAll(value, "\n", "\n  ")
			sb.WriteString(fmt.Sprintf("- %s:\n%s\n", label, indented))
		} else {
			sb.WriteString(fmt.Sprintf("- %s: %s\n", label, value))
		}
	}

	return sb.String()
}

// 소셜 링크 섹션 포맷팅 (최대 4개)
func formatProfileSocialLinks(raw *domain.TalentProfile) string {
	if raw == nil || len(raw.SocialLinks) == 0 {
		return ""
	}

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("\n%s 링크\n", DefaultEmoji.Link))

	maxLinks := len(raw.SocialLinks)
	if maxLinks > 4 {
		maxLinks = 4
	}

	for i := 0; i < maxLinks; i++ {
		link := raw.SocialLinks[i]
		if util.TrimSpace(link.Label) == "" || util.TrimSpace(link.URL) == "" {
			continue
		}
		translatedLabel := socialLinkLabel(link.Label)
		sb.WriteString(fmt.Sprintf("- %s: %s\n", translatedLabel, util.TrimSpace(link.URL)))
	}

	return sb.String()
}

// 공식 URL 섹션 포맷팅
func formatProfileOfficialURL(raw *domain.TalentProfile) string {
	if raw == nil || util.TrimSpace(raw.OfficialURL) == "" {
		return ""
	}
	return fmt.Sprintf("\n%s 공식 프로필: %s", DefaultEmoji.Web, util.TrimSpace(raw.OfficialURL))
}

// FormatTalentProfile: 탤런트 프로필 정보를 포맷팅하여 메시지 문자열을 생성합니다.
func (f *ResponseFormatter) FormatTalentProfile(raw *domain.TalentProfile, translated *domain.Translated) string {
	if raw == nil {
		return ErrorMessage(ErrDisplayProfileDataFailed)
	}

	var sb strings.Builder
	header := buildTalentHeader(raw, translated)
	sb.WriteString(header)
	sb.WriteString("\n")

	sb.WriteString(formatProfileCatchphrase(raw, translated))
	sb.WriteString(formatProfileSummary(raw, translated))
	sb.WriteString(formatProfileHighlights(translated))
	sb.WriteString(formatProfileDataEntries(raw, translated))
	sb.WriteString(formatProfileSocialLinks(raw))
	sb.WriteString(formatProfileOfficialURL(raw))

	content := util.TrimSpace(sb.String())
	if content == "" {
		return content
	}

	body := util.StripLeadingHeader(content, header)
	body = util.TrimSpace(body)
	if body == "" {
		return content
	}

	instructionBase := util.TrimSpace(header)
	if instructionBase == "" {
		instructionBase = DefaultEmoji.Member + " 멤버 정보"
	}

	return util.ApplyKakaoSeeMorePadding(body, instructionBase)
}

func socialLinkLabel(label string) string {
	translations := map[string]string{
		"歌の再生リスト":   "음악 플레이리스트",
		"公式グッズ":     "공식 굿즈",
		"オフィシャルグッズ": "공식 굿즈",
	}

	if korean, ok := translations[label]; ok {
		return korean
	}
	return label
}

func buildTalentHeader(raw *domain.TalentProfile, translated *domain.Translated) string {
	names := talentDisplayNames(raw, translated)
	return MemberHeader(names)
}

func talentDisplayNames(raw *domain.TalentProfile, translated *domain.Translated) []string {
	var names []string

	english := ""
	japanese := ""
	if raw != nil {
		english = util.TrimSpace(raw.EnglishName)
		japanese = util.TrimSpace(raw.JapaneseName)
	}

	display := ""
	if translated != nil {
		display = util.TrimSpace(translated.DisplayName)
	}

	if english != "" {
		addUniqueName(&names, english)
	}

	for _, candidate := range parseDisplayNameComponents(display) {
		addUniqueName(&names, candidate)
	}

	if japanese != "" {
		addUniqueName(&names, japanese)
	}

	return names
}

func parseDisplayNameComponents(display string) []string {
	display = util.TrimSpace(display)
	if display == "" {
		return nil
	}

	var rawParts []string

	openIdx := strings.Index(display, "(")
	closeIdx := strings.LastIndex(display, ")")
	if openIdx != -1 && closeIdx != -1 && closeIdx > openIdx {
		before := util.TrimSpace(display[:openIdx])
		inside := util.TrimSpace(display[openIdx+1 : closeIdx])
		after := util.TrimSpace(display[closeIdx+1:])

		if before != "" {
			rawParts = append(rawParts, before)
		}
		if inside != "" {
			rawParts = append(rawParts, inside)
		}
		if after != "" {
			rawParts = append(rawParts, after)
		}
	} else {
		rawParts = append(rawParts, display)
	}

	var result []string
	for _, part := range rawParts {
		segments := strings.Split(part, "/")
		for _, segment := range segments {
			candidate := util.TrimSpace(segment)
			if candidate != "" {
				result = append(result, candidate)
			}
		}
	}

	return result
}

func addUniqueName(names *[]string, candidate string) {
	candidate = util.TrimSpace(candidate)
	if candidate == "" {
		return
	}

	for _, existing := range *names {
		if strings.EqualFold(existing, candidate) {
			return
		}
	}

	*names = append(*names, candidate)
}
