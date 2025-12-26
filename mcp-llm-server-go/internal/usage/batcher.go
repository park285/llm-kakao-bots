package usage

import (
	"context"
	"log/slog"
	"sync"
	"time"

	"github.com/park285/llm-kakao-bots/mcp-llm-server-go/internal/config"
)

// usageDelta 일별 토큰 사용량 델타
type usageDelta struct {
	inputTokens     int64
	outputTokens    int64
	reasoningTokens int64
	requestCount    int64
}

const defaultFlushTimeout = 5 * time.Second

// batcher 는 토큰 사용량을 배치로 DB에 플러시한다.
type batcher struct {
	repo                     *Repository
	logger                   *slog.Logger
	flushInterval            time.Duration
	flushTimeout             time.Duration
	maxPendingRequests       int
	maxBackoff               time.Duration
	errorLogMaxInterval      time.Duration
	mu                       sync.Mutex
	pending                  map[time.Time]*usageDelta
	pendingRequestsTotal     int
	wakeup                   chan struct{}
	stopCh                   chan struct{}
	doneCh                   chan struct{}
	consecutiveFlushFailures int
	nextFlushAllowedAt       time.Time
	lastErrorLoggedAt        time.Time
	flushSuccessTotal        int
	flushFailureTotal        int
	flushRequeuedTotal       int
	flushDroppedTotal        int
}

// newBatcher 새로운 배치 플러셔 생성
func newBatcher(cfg *config.Config, repo *Repository, logger *slog.Logger) *batcher {
	interval := time.Duration(cfg.Database.UsageBatchFlushIntervalSeconds) * time.Second
	if interval <= 0 {
		interval = time.Second
	}
	maxBackoff := time.Duration(cfg.Database.UsageBatchMaxBackoffSeconds) * time.Second
	if maxBackoff <= 0 {
		maxBackoff = interval
	}
	maxPending := cfg.Database.UsageBatchMaxPendingRequests
	if maxPending <= 0 {
		maxPending = 1
	}
	flushTimeout := defaultFlushTimeout
	if cfg.Database.UsageBatchFlushTimeoutSeconds > 0 {
		flushTimeout = time.Duration(cfg.Database.UsageBatchFlushTimeoutSeconds) * time.Second
	}
	if flushTimeout <= 0 {
		flushTimeout = interval
	}
	return &batcher{
		repo:                repo,
		logger:              logger,
		flushInterval:       interval,
		flushTimeout:        flushTimeout,
		maxPendingRequests:  maxPending,
		maxBackoff:          maxBackoff,
		errorLogMaxInterval: time.Duration(cfg.Database.UsageBatchErrorLogMaxIntervalSeconds) * time.Second,
		pending:             make(map[time.Time]*usageDelta),
		wakeup:              make(chan struct{}, 1),
		stopCh:              make(chan struct{}),
		doneCh:              make(chan struct{}),
	}
}

func (b *batcher) start() {
	go b.loop()
}

func (b *batcher) stop() {
	close(b.stopCh)
	<-b.doneCh
}

func (b *batcher) add(inputTokens int64, outputTokens int64, reasoningTokens int64, requestCount int64) {
	if inputTokens <= 0 && outputTokens <= 0 {
		return
	}

	targetDate := todayDate()
	b.mu.Lock()
	delta := b.pending[targetDate]
	if delta == nil {
		delta = &usageDelta{}
		b.pending[targetDate] = delta
	}
	delta.inputTokens += inputTokens
	delta.outputTokens += outputTokens
	delta.reasoningTokens += reasoningTokens
	delta.requestCount += requestCount
	b.pendingRequestsTotal += int(requestCount)
	shouldFlush := b.pendingRequestsTotal >= b.maxPendingRequests
	b.mu.Unlock()

	if shouldFlush {
		b.signal()
	}
}

func (b *batcher) loop() {
	ticker := time.NewTicker(b.flushInterval)
	defer func() {
		ticker.Stop()
		close(b.doneCh)
	}()

	for {
		select {
		case <-ticker.C:
			b.flush(false)
		case <-b.wakeup:
			b.flush(false)
		case <-b.stopCh:
			b.flush(true)
			return
		}
	}
}

func (b *batcher) signal() {
	select {
	case b.wakeup <- struct{}{}:
	default:
	}
}

func (b *batcher) flush(isShutdown bool) {
	if b.shouldSkipFlush(isShutdown) {
		return
	}

	snapshot := b.takeSnapshot()
	if len(snapshot) == 0 {
		return
	}

	hadFailure, firstErr := b.applySnapshot(snapshot, isShutdown)
	if hadFailure {
		b.registerFailure(firstErr)
		return
	}

	b.resetFailures()
}

func (b *batcher) shouldSkipFlush(isShutdown bool) bool {
	if isShutdown {
		return false
	}
	if b.nextFlushAllowedAt.IsZero() {
		return false
	}
	return time.Now().Before(b.nextFlushAllowedAt)
}

func (b *batcher) takeSnapshot() map[time.Time]usageDelta {
	snapshot := make(map[time.Time]usageDelta)
	b.mu.Lock()
	for date, delta := range b.pending {
		snapshot[date] = *delta
	}
	b.pending = make(map[time.Time]*usageDelta)
	b.pendingRequestsTotal = 0
	b.mu.Unlock()
	return snapshot
}

func (b *batcher) applySnapshot(snapshot map[time.Time]usageDelta, isShutdown bool) (bool, error) {
	hadFailure := false
	var firstErr error
	for date, delta := range snapshot {
		ctx := context.Background()
		cancel := func() {}
		if b.flushTimeout > 0 {
			ctx, cancel = context.WithTimeout(ctx, b.flushTimeout)
		}
		err := b.repo.RecordUsage(
			ctx,
			delta.inputTokens,
			delta.outputTokens,
			delta.reasoningTokens,
			delta.requestCount,
			date,
		)
		cancel()
		if err != nil {
			hadFailure = true
			if firstErr == nil {
				firstErr = err
			}
			b.flushFailureTotal++
			if isShutdown {
				b.flushDroppedTotal++
				continue
			}
			b.requeue(date, delta)
			b.flushRequeuedTotal++
			continue
		}
		b.flushSuccessTotal++
	}
	return hadFailure, firstErr
}

func (b *batcher) requeue(date time.Time, delta usageDelta) {
	b.mu.Lock()
	existing := b.pending[date]
	if existing == nil {
		existing = &usageDelta{}
		b.pending[date] = existing
	}
	existing.inputTokens += delta.inputTokens
	existing.outputTokens += delta.outputTokens
	existing.reasoningTokens += delta.reasoningTokens
	existing.requestCount += delta.requestCount
	b.pendingRequestsTotal += int(delta.requestCount)
	b.mu.Unlock()
}

func (b *batcher) registerFailure(firstErr error) {
	b.consecutiveFlushFailures++
	backoff := b.computeBackoff()
	b.nextFlushAllowedAt = time.Now().Add(backoff)

	if b.shouldLogFailure() {
		b.lastErrorLoggedAt = time.Now()
		if b.logger != nil {
			b.logger.Warn(
				"usage_db_batch_flush_failed",
				"failures", b.consecutiveFlushFailures,
				"backoff", backoff,
				"pending_requests", b.pendingRequestsTotal,
				"err", firstErr,
			)
		}
	}
}

func (b *batcher) computeBackoff() time.Duration {
	backoff := b.flushInterval * time.Duration(1<<max(0, b.consecutiveFlushFailures-1))
	if backoff > b.maxBackoff {
		backoff = b.maxBackoff
	}
	if backoff <= 0 {
		backoff = b.flushInterval
	}
	return backoff
}

func (b *batcher) resetFailures() {
	b.consecutiveFlushFailures = 0
	b.nextFlushAllowedAt = time.Time{}
}

func (b *batcher) shouldLogFailure() bool {
	if b.consecutiveFlushFailures <= 0 {
		return false
	}
	if isPowerOfTwo(b.consecutiveFlushFailures) {
		return true
	}
	if b.errorLogMaxInterval <= 0 {
		return false
	}
	return time.Since(b.lastErrorLoggedAt) >= b.errorLogMaxInterval
}

// isPowerOfTwo 2의 거듭제곱인지 확인
func isPowerOfTwo(value int) bool {
	return value > 0 && (value&(value-1)) == 0
}
