package shared

import (
	"fmt"
	"strings"

	"github.com/park285/llm-kakao-bots/mcp-llm-server-go/internal/llm"
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
func BuildRecentQAHistoryContext(history []llm.HistoryEntry, header string, maxPairs int) string {
	if maxPairs <= 0 {
		return ""
	}
	historyLines := make([]string, 0, len(history))
	for _, entry := range history {
		content := entry.Content
		if strings.HasPrefix(content, "Q:") || strings.HasPrefix(content, "A:") {
			historyLines = append(historyLines, content)
		}
	}
	if len(historyLines) == 0 {
		return ""
	}

	maxLines := maxPairs * 2
	if len(historyLines) > maxLines {
		historyLines = historyLines[len(historyLines)-maxLines:]
	}
	return "\n\n" + header + "\n" + strings.Join(historyLines, "\n")
}

// ValueOrEmpty 는 nil 포인터면 빈 문자열을 반환한다.
func ValueOrEmpty(value *string) string {
	if value == nil {
		return ""
	}
	return *value
}
