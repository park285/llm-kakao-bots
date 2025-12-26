package usage

import (
	"context"
	"log/slog"
	"time"

	"github.com/park285/llm-kakao-bots/mcp-llm-server-go/internal/config"
)

// Recorder 는 요청별 토큰 사용량을 저장하거나 배치로 적재한다.
type Recorder struct {
	repo    *Repository
	batcher *batcher
	logger  *slog.Logger
}

// NewRecorder 는 설정에 따라 배치 사용 여부를 결정해 Recorder를 생성한다.
func NewRecorder(cfg *config.Config, repo *Repository, logger *slog.Logger) *Recorder {
	recorder := &Recorder{
		repo:   repo,
		logger: logger,
	}

	if cfg != nil && cfg.Database.UsageBatchEnabled {
		recorder.batcher = newBatcher(cfg, repo, logger)
		recorder.batcher.start()
		if logger != nil {
			logger.Info(
				"usage_db_batch_enabled",
				"flush_interval_seconds", cfg.Database.UsageBatchFlushIntervalSeconds,
				"flush_timeout_seconds", cfg.Database.UsageBatchFlushTimeoutSeconds,
				"max_pending_requests", cfg.Database.UsageBatchMaxPendingRequests,
				"max_backoff_seconds", cfg.Database.UsageBatchMaxBackoffSeconds,
				"error_log_max_interval_seconds", cfg.Database.UsageBatchErrorLogMaxIntervalSeconds,
			)
		}
	}

	return recorder
}

// Record 는 1회 요청의 토큰 사용량을 기록한다.
func (r *Recorder) Record(ctx context.Context, inputTokens int64, outputTokens int64, reasoningTokens int64) {
	if r == nil || r.repo == nil {
		return
	}
	if inputTokens <= 0 && outputTokens <= 0 {
		return
	}

	if r.batcher != nil {
		r.batcher.add(inputTokens, outputTokens, reasoningTokens, 1)
		return
	}

	if err := r.repo.RecordUsage(ctx, inputTokens, outputTokens, reasoningTokens, 1, time.Time{}); err != nil {
		if r.logger != nil {
			r.logger.Warn("usage_db_save_failed", "err", err)
		}
	}
}

// Close 는 배치 플러셔를 중지한다.
func (r *Recorder) Close() {
	if r == nil || r.batcher == nil {
		return
	}
	r.batcher.stop()
}
