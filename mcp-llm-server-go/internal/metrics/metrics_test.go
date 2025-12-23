package metrics

import (
	"testing"
	"time"

	"github.com/park285/llm-kakao-bots/mcp-llm-server-go/internal/llm"
)

func TestStoreRecordsMetrics(t *testing.T) {
	store := NewStore()
	store.RecordSuccess(120*time.Millisecond, llm.Usage{InputTokens: 2, OutputTokens: 3, ReasoningTokens: 1})
	store.RecordError(50 * time.Millisecond)

	usage := store.UsageTotals()
	if usage.InputTokens != 2 || usage.OutputTokens != 3 || usage.ReasoningTokens != 1 {
		t.Fatalf("unexpected usage totals: %+v", usage)
	}

	snapshot := store.Snapshot()
	if snapshot["total_calls"] != 2 {
		t.Fatalf("expected total_calls 2, got %v", snapshot["total_calls"])
	}
	if snapshot["total_errors"] != 1 {
		t.Fatalf("expected total_errors 1, got %v", snapshot["total_errors"])
	}
}
