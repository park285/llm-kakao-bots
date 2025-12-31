package pending

import (
	"fmt"
	"testing"
)

func TestFormatQueueDetails(t *testing.T) {
	alice := "Alice"

	tests := []struct {
		name        string
		entries     []string
		displayName func(userID string, sender *string) string
		parse       QueueDetailsParser
		want        string
	}{
		{
			name:    "empty",
			entries: nil,
			parse: func(_ string) (QueueDetailsItem, bool) {
				return QueueDetailsItem{}, true
			},
			want: "",
		},
		{
			name:    "formats with displayName and preserves index",
			entries: []string{"0|u1|p1", "invalid", "0|u2|p2"},
			displayName: func(userID string, sender *string) string {
				if sender != nil {
					return fmt.Sprintf("%s(%s)", userID, *sender)
				}
				return userID
			},
			parse: func(jsonPart string) (QueueDetailsItem, bool) {
				switch jsonPart {
				case "p1":
					return QueueDetailsItem{Sender: &alice, Content: "hello"}, true
				case "p2":
					return QueueDetailsItem{Sender: nil, Content: "world"}, true
				default:
					return QueueDetailsItem{}, false
				}
			},
			want: "1. u1(Alice) - hello\n3. u2 - world",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := FormatQueueDetails(tt.entries, tt.displayName, tt.parse)
			if got != tt.want {
				t.Fatalf("unexpected result: got=%q want=%q", got, tt.want)
			}
		})
	}
}

