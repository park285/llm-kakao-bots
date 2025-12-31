package pending

import (
	"fmt"
	"strings"
)

// QueueDetailsItem: 대기열 요약 문자열을 구성하는 데 필요한 최소 정보 묶음입니다.
type QueueDetailsItem struct {
	Sender  *string
	Content string
}

// QueueDetailsParser: raw JSON payload를 QueueDetailsItem으로 변환하는 함수 타입입니다.
type QueueDetailsParser func(jsonPart string) (QueueDetailsItem, bool)

// FormatQueueDetails: `GetRawEntries` 결과를 사람이 읽을 수 있는 문자열로 포맷팅합니다.
// 도메인별 payload 구조 차이를 `parse` 콜백으로 위임하여 중복 로직을 줄이기 위함입니다.
func FormatQueueDetails(
	entries []string,
	displayName func(userID string, sender *string) string,
	parse QueueDetailsParser,
) string {
	if len(entries) == 0 {
		return ""
	}

	lines := make([]string, 0, len(entries))
	for idx, entry := range entries {
		entryUserID, jsonPart, ok := ExtractUserIDAndJSON(entry)
		if !ok {
			continue
		}

		item, ok := parse(jsonPart)
		if !ok {
			continue
		}

		name := entryUserID
		if displayName != nil {
			name = displayName(entryUserID, item.Sender)
		}

		lines = append(lines, fmt.Sprintf("%d. %s - %s", idx+1, name, item.Content))
	}

	return strings.Join(lines, "\n")
}
