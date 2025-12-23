package usage

import "testing"

func TestDailyUsageTotals(t *testing.T) {
	row := DailyUsage{InputTokens: 2, OutputTokens: 3}
	if row.TotalTokens() != 5 {
		t.Fatalf("unexpected total tokens")
	}
}
