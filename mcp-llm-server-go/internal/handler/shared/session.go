package shared

import (
	"fmt"
	"strings"

	"github.com/park285/llm-kakao-bots/mcp-llm-server-go/internal/llm"
	"github.com/park285/llm-kakao-bots/mcp-llm-server-go/internal/prompt"
)

// ResolveSessionID 는 세션 ID를 결정한다.
// 반환: (effectiveSessionID, derived). derived가 true면 chatID+namespace로 생성된 것.
func ResolveSessionID(sessionID string, chatID string, namespace string, defaultNamespace string) (string, bool) {
	if sessionID != "" {
		return sessionID, false
	}
	if chatID == "" {
		return "", false
	}
	effectiveNamespace := namespace
	if effectiveNamespace == "" {
		effectiveNamespace = defaultNamespace
	}
	return fmt.Sprintf("%s:%s", effectiveNamespace, chatID), true
}

// BuildRecentQAHistoryContext 는 히스토리에서 최근 Q/A 쌍을 추출해 문자열로 변환한다.
// 확인된 속성 요약을 상단에 추가하여 일관성 유지를 돕는다.
func BuildRecentQAHistoryContext(history []llm.HistoryEntry, header string, maxPairs int) string {
	if maxPairs <= 0 {
		return ""
	}

	// Q&A 쌍 추출
	type qaPair struct {
		question string
		answer   string
	}
	pairs := make([]qaPair, 0)
	var currentQ string

	for _, entry := range history {
		content := strings.TrimSpace(entry.Content)
		if strings.HasPrefix(content, "Q:") {
			currentQ = strings.TrimSpace(strings.TrimPrefix(content, "Q:"))
			continue
		}
		if strings.HasPrefix(content, "A:") && currentQ != "" {
			answer := strings.TrimSpace(strings.TrimPrefix(content, "A:"))
			pairs = append(pairs, qaPair{question: currentQ, answer: answer})
			currentQ = ""
		}
	}

	if len(pairs) == 0 {
		return ""
	}

	// 최근 N개 쌍만 유지
	if len(pairs) > maxPairs {
		pairs = pairs[len(pairs)-maxPairs:]
	}

	// 확인된 속성 요약 (간결한 형태)
	summaryLines := make([]string, 0, len(pairs))
	for _, p := range pairs {
		// Q를 축약하여 속성처럼 표현
		summaryLines = append(summaryLines, fmt.Sprintf("• %s → %s", truncateQuestion(p.question, 20), p.answer))
	}

	// 상세 Q&A 목록
	historyLines := make([]string, 0, len(pairs)*2)
	for _, p := range pairs {
		historyLines = append(historyLines,
			prompt.WrapXML("q", p.question),
			prompt.WrapXML("a", p.answer))
	}

	var result strings.Builder
	result.WriteString("\n\n")
	result.WriteString(header)
	result.WriteString("\n[확인된 속성 요약]\n")
	result.WriteString(strings.Join(summaryLines, "\n"))
	result.WriteString("\n\n[상세 기록]\n")
	result.WriteString(strings.Join(historyLines, "\n"))

	return result.String()
}

// truncateQuestion 은 질문을 maxLen 글자로 축약한다.
func truncateQuestion(q string, maxLen int) string {
	runes := []rune(q)
	if len(runes) <= maxLen {
		return q
	}
	return string(runes[:maxLen]) + "..."
}

// ValueOrEmpty 는 nil 포인터면 빈 문자열을 반환한다.
func ValueOrEmpty(value *string) string {
	if value == nil {
		return ""
	}
	return *value
}
