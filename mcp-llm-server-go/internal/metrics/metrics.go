package metrics

import (
	"sync/atomic"
	"time"

	"github.com/park285/llm-kakao-bots/mcp-llm-server-go/internal/llm"
)

// Store: LLM 호출 통계를 저장합니다.
type Store struct {
	totalCalls           int64
	totalErrors          int64
	totalInputTokens     int64
	totalOutputTokens    int64
	totalReasoningTokens int64
	totalCachedTokens    int64 // 암시적 캐싱된 토큰 누적
	totalDurationMs      int64
}

// NewStore: 통계 저장소를 생성합니다.
func NewStore() *Store {
	return &Store{}
}

// RecordSuccess: 성공 호출 통계를 기록합니다.
func (s *Store) RecordSuccess(duration time.Duration, usage llm.Usage) {
	atomic.AddInt64(&s.totalCalls, 1)
	atomic.AddInt64(&s.totalInputTokens, int64(usage.InputTokens))
	atomic.AddInt64(&s.totalOutputTokens, int64(usage.OutputTokens))
	atomic.AddInt64(&s.totalReasoningTokens, int64(usage.ReasoningTokens))
	atomic.AddInt64(&s.totalCachedTokens, int64(usage.CachedTokens))
	atomic.AddInt64(&s.totalDurationMs, duration.Milliseconds())
}

// RecordError: 실패 호출 통계를 기록합니다.
func (s *Store) RecordError(duration time.Duration) {
	atomic.AddInt64(&s.totalCalls, 1)
	atomic.AddInt64(&s.totalErrors, 1)
	atomic.AddInt64(&s.totalDurationMs, duration.Milliseconds())
}

// UsageTotals: 누적 사용량을 반환합니다.
func (s *Store) UsageTotals() llm.Usage {
	input := atomic.LoadInt64(&s.totalInputTokens)
	output := atomic.LoadInt64(&s.totalOutputTokens)
	reasoning := atomic.LoadInt64(&s.totalReasoningTokens)
	cached := atomic.LoadInt64(&s.totalCachedTokens)
	return llm.Usage{
		InputTokens:     int(input),
		OutputTokens:    int(output),
		TotalTokens:     int(input + output),
		ReasoningTokens: int(reasoning),
		CachedTokens:    int(cached),
	}
}

// Snapshot: 통계 스냅샷을 반환합니다.
func (s *Store) Snapshot() map[string]float64 {
	totalCalls := atomic.LoadInt64(&s.totalCalls)
	totalErrors := atomic.LoadInt64(&s.totalErrors)
	input := atomic.LoadInt64(&s.totalInputTokens)
	output := atomic.LoadInt64(&s.totalOutputTokens)
	reasoning := atomic.LoadInt64(&s.totalReasoningTokens)
	cached := atomic.LoadInt64(&s.totalCachedTokens)
	durationMs := atomic.LoadInt64(&s.totalDurationMs)

	avgDuration := 0.0
	if totalCalls > 0 {
		avgDuration = float64(durationMs) / float64(totalCalls)
	}

	// 캐시 적중률 계산 (InputTokens 대비 CachedTokens 비율)
	cacheHitRatio := 0.0
	if input > 0 {
		cacheHitRatio = float64(cached) / float64(input)
	}

	return map[string]float64{
		"total_calls":            float64(totalCalls),
		"total_errors":           float64(totalErrors),
		"total_input_tokens":     float64(input),
		"total_output_tokens":    float64(output),
		"total_reasoning_tokens": float64(reasoning),
		"total_cached_tokens":    float64(cached), // 캐시된 토큰 누적
		"cache_hit_ratio":        cacheHitRatio,   // 캐시 적중률 (0.0 ~ 1.0)
		"total_tokens":           float64(input + output),
		"total_duration_ms":      float64(durationMs),
		"avg_duration_ms":        avgDuration,
	}
}
