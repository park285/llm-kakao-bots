package metrics

import (
	"sync/atomic"
	"time"

	"github.com/park285/llm-kakao-bots/mcp-llm-server-go/internal/llm"
)

// Store 는 LLM 호출 통계를 저장한다.
type Store struct {
	totalCalls           int64
	totalErrors          int64
	totalInputTokens     int64
	totalOutputTokens    int64
	totalReasoningTokens int64
	totalDurationMs      int64
}

// NewStore 는 통계 저장소를 생성한다.
func NewStore() *Store {
	return &Store{}
}

// RecordSuccess 는 성공 호출 통계를 기록한다.
func (s *Store) RecordSuccess(duration time.Duration, usage llm.Usage) {
	atomic.AddInt64(&s.totalCalls, 1)
	atomic.AddInt64(&s.totalInputTokens, int64(usage.InputTokens))
	atomic.AddInt64(&s.totalOutputTokens, int64(usage.OutputTokens))
	atomic.AddInt64(&s.totalReasoningTokens, int64(usage.ReasoningTokens))
	atomic.AddInt64(&s.totalDurationMs, duration.Milliseconds())
}

// RecordError 는 실패 호출 통계를 기록한다.
func (s *Store) RecordError(duration time.Duration) {
	atomic.AddInt64(&s.totalCalls, 1)
	atomic.AddInt64(&s.totalErrors, 1)
	atomic.AddInt64(&s.totalDurationMs, duration.Milliseconds())
}

// UsageTotals 는 누적 사용량을 반환한다.
func (s *Store) UsageTotals() llm.Usage {
	input := atomic.LoadInt64(&s.totalInputTokens)
	output := atomic.LoadInt64(&s.totalOutputTokens)
	reasoning := atomic.LoadInt64(&s.totalReasoningTokens)
	return llm.Usage{
		InputTokens:     int(input),
		OutputTokens:    int(output),
		TotalTokens:     int(input + output),
		ReasoningTokens: int(reasoning),
	}
}

// Snapshot 는 통계 스냅샷을 반환한다.
func (s *Store) Snapshot() map[string]float64 {
	totalCalls := atomic.LoadInt64(&s.totalCalls)
	totalErrors := atomic.LoadInt64(&s.totalErrors)
	input := atomic.LoadInt64(&s.totalInputTokens)
	output := atomic.LoadInt64(&s.totalOutputTokens)
	reasoning := atomic.LoadInt64(&s.totalReasoningTokens)
	durationMs := atomic.LoadInt64(&s.totalDurationMs)

	avgDuration := 0.0
	if totalCalls > 0 {
		avgDuration = float64(durationMs) / float64(totalCalls)
	}

	return map[string]float64{
		"total_calls":            float64(totalCalls),
		"total_errors":           float64(totalErrors),
		"total_input_tokens":     float64(input),
		"total_output_tokens":    float64(output),
		"total_reasoning_tokens": float64(reasoning),
		"total_tokens":           float64(input + output),
		"total_duration_ms":      float64(durationMs),
		"avg_duration_ms":        avgDuration,
	}
}
