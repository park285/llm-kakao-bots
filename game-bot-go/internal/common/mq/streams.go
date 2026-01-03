package mq

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"sync"
	"time"

	"github.com/valkey-io/valkey-go"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"

	"github.com/park285/llm-kakao-bots/game-bot-go/internal/common/telemetry"
	"github.com/park285/llm-kakao-bots/game-bot-go/internal/common/valkeyx"
)

// StreamConsumerConfig: 스트림 소비자 설정 구조체
type StreamConsumerConfig struct {
	Stream string
	Group  string
	Name   string

	BatchSize   int64
	Block       time.Duration
	Concurrency int

	ResetGroupOnStartup bool
	AckOnError          bool

	AckMaxRetries  int
	AckRetryDelay  time.Duration
	GroupStartFrom string

	// Backoff: 연결 에러 재시도 설정
	BackoffInitial time.Duration // 초기 대기 시간 (기본: 1초)
	BackoffMax     time.Duration // 최대 대기 시간 (기본: 30초)
	BackoffFactor  float64       // 대기 시간 증가 배수 (기본: 2.0)
}

// XMessage: Redis 스트림에서 읽어온 메시지 구조체
type XMessage struct {
	ID     string
	Values map[string]string
}

// StreamConsumer: Redis Consumer Group을 사용하여 스트림 메시지를 처리하는 소비자
type StreamConsumer struct {
	client valkey.Client
	logger *slog.Logger
	cfg    StreamConsumerConfig
}

// NewStreamConsumer: 새로운 StreamConsumer 인스턴스를 생성합니다.
func NewStreamConsumer(client valkey.Client, logger *slog.Logger, cfg StreamConsumerConfig) *StreamConsumer {
	return &StreamConsumer{
		client: client,
		logger: logger,
		cfg:    cfg,
	}
}

// Run: 메시지 소비 루프를 실행합니다. (블로킹 방식)
func (c *StreamConsumer) Run(ctx context.Context, handler func(ctx context.Context, msg XMessage) error) error {
	cfg, err := c.normalizedConfig()
	if err != nil {
		return err
	}

	if err := c.prepareGroup(ctx, cfg); err != nil {
		return err
	}

	sem := make(chan struct{}, cfg.Concurrency)
	var wg sync.WaitGroup
	defer wg.Wait()

	// 지수 백오프 상태
	backoff := cfg.BackoffInitial

	for {
		if ctx.Err() != nil {
			return nil
		}

		messages, err := c.readBatch(ctx, cfg)
		if err != nil {
			if valkeyx.IsNil(err) || (errors.Is(err, context.DeadlineExceeded) && ctx.Err() == nil) {
				backoff = cfg.BackoffInitial // 타임아웃은 정상 상황, 리셋
				continue
			}
			if errors.Is(err, context.Canceled) && ctx.Err() != nil {
				return nil
			}

			// NOGROUP 에러 시 컨슈머 그룹 자동 재생성 시도
			if isNoGroupOrNoStreamErr(err) {
				c.logger.Info("consumer_group_missing_recreating", "stream", cfg.Stream, "group", cfg.Group)
				if recreateErr := c.ensureGroup(ctx, cfg); recreateErr != nil {
					c.logger.Warn("consumer_group_recreate_failed", "err", recreateErr, "stream", cfg.Stream, "group", cfg.Group)
				} else {
					c.logger.Info("consumer_group_recreated", "stream", cfg.Stream, "group", cfg.Group)
					backoff = cfg.BackoffInitial // 성공 시 백오프 리셋
					continue
				}
			}

			c.logger.Warn("xreadgroup_failed", "err", err, "stream", cfg.Stream, "group", cfg.Group, "backoff", backoff)

			// 지수 백오프 대기 후 재시도
			if !sleepWithContext(ctx, backoff) {
				return nil // context 취소 시 종료
			}
			backoff = time.Duration(float64(backoff) * cfg.BackoffFactor)
			if backoff > cfg.BackoffMax {
				backoff = cfg.BackoffMax
			}
			continue
		}

		// 연결 성공 시 backoff 리셋
		backoff = cfg.BackoffInitial

		for _, msg := range messages {
			if ctx.Err() != nil {
				return nil
			}
			c.spawnHandler(ctx, cfg, sem, &wg, msg, handler)
		}
	}
}

func (c *StreamConsumer) prepareGroup(ctx context.Context, cfg StreamConsumerConfig) error {
	if cfg.ResetGroupOnStartup {
		return c.resetGroup(ctx, cfg)
	}
	return c.ensureGroup(ctx, cfg)
}

func (c *StreamConsumer) readBatch(ctx context.Context, cfg StreamConsumerConfig) ([]XMessage, error) {
	cmd := c.client.B().Xreadgroup().
		Group(cfg.Group, cfg.Name).
		Count(cfg.BatchSize).
		Block(cfg.Block.Milliseconds()).
		Streams().Key(cfg.Stream).Id(">").
		Build()

	result, err := c.client.Do(ctx, cmd).AsXRead()
	if err != nil {
		return nil, fmt.Errorf("xreadgroup failed: %w", err)
	}

	var messages []XMessage
	for stream, entries := range result {
		if stream != cfg.Stream {
			continue
		}
		for _, entry := range entries {
			messages = append(messages, XMessage{
				ID:     entry.ID,
				Values: entry.FieldValues,
			})
		}
	}
	return messages, nil
}

func (c *StreamConsumer) spawnHandler(
	ctx context.Context,
	cfg StreamConsumerConfig,
	sem chan struct{},
	wg *sync.WaitGroup,
	msg XMessage,
	handler func(ctx context.Context, msg XMessage) error,
) {
	select {
	case sem <- struct{}{}:
	case <-ctx.Done():
		return
	}
	wg.Add(1)

	go func(m XMessage) {
		defer wg.Done()
		defer func() { <-sem }()

		c.handleMessage(ctx, cfg, m, handler)
	}(msg)
}

func (c *StreamConsumer) handleMessage(
	ctx context.Context,
	cfg StreamConsumerConfig,
	msg XMessage,
	handler func(ctx context.Context, msg XMessage) error,
) {
	tracer := otel.Tracer("game-bot-go/valkey-consumer")

	// 1. 메시지 헤더에서 부모 Context 추출 (미래 확장용)
	// 현재는 메시지에 TraceContext가 없으므로 새 Trace 시작
	carrier := telemetry.MapCarrier(msg.Values)
	parentCtx := telemetry.ExtractContext(ctx, carrier)

	// 2. Root Span 시작 (SpanKindConsumer로 표시)
	spanCtx, span := tracer.Start(parentCtx, "Valkey.ProcessMessage",
		trace.WithSpanKind(trace.SpanKindConsumer),
		trace.WithAttributes(
			attribute.String("messaging.system", "valkey"),
			attribute.String("messaging.destination", cfg.Stream),
			attribute.String("messaging.message_id", msg.ID),
			attribute.String("messaging.consumer_group", cfg.Group),
		),
	)
	defer span.End()

	// 3. 생성된 spanCtx를 비즈니스 로직에 전달
	handleErr := handler(spanCtx, msg)
	if handleErr != nil {
		span.RecordError(handleErr)
		span.SetStatus(codes.Error, handleErr.Error())
		c.logger.ErrorContext(spanCtx, "message_handler_failed",
			"err", handleErr,
			"stream", cfg.Stream,
			"id", msg.ID,
		)
		if !cfg.AckOnError {
			return
		}
	} else {
		span.SetStatus(codes.Ok, "")
	}

	if errAck := c.ackWithRetry(spanCtx, cfg, msg.ID); errAck != nil {
		c.logger.WarnContext(spanCtx, "xack_failed",
			"err", errAck,
			"stream", cfg.Stream,
			"id", msg.ID,
		)
	}
}

func (c *StreamConsumer) normalizedConfig() (StreamConsumerConfig, error) {
	cfg := c.cfg
	cfg.Stream = strings.TrimSpace(cfg.Stream)
	cfg.Group = strings.TrimSpace(cfg.Group)
	cfg.Name = strings.TrimSpace(cfg.Name)
	if cfg.Stream == "" || cfg.Group == "" || cfg.Name == "" {
		return StreamConsumerConfig{}, errors.New("stream/group/name must be set")
	}
	if cfg.BatchSize <= 0 {
		cfg.BatchSize = 10
	}
	if cfg.Block <= 0 {
		cfg.Block = 5 * time.Second
	}
	if cfg.Concurrency <= 0 {
		cfg.Concurrency = 10
	}
	if cfg.AckMaxRetries <= 0 {
		cfg.AckMaxRetries = 1
	}
	if cfg.AckRetryDelay <= 0 {
		cfg.AckRetryDelay = 100 * time.Millisecond
	}
	if strings.TrimSpace(cfg.GroupStartFrom) == "" {
		cfg.GroupStartFrom = "$"
	}
	// Backoff 기본값 설정
	if cfg.BackoffInitial <= 0 {
		cfg.BackoffInitial = 1 * time.Second
	}
	if cfg.BackoffMax <= 0 {
		cfg.BackoffMax = 30 * time.Second
	}
	if cfg.BackoffFactor <= 0 {
		cfg.BackoffFactor = 2.0
	}
	return cfg, nil
}

func (c *StreamConsumer) ensureGroup(ctx context.Context, cfg StreamConsumerConfig) error {
	cmd := c.client.B().XgroupCreate().Key(cfg.Stream).Group(cfg.Group).Id(cfg.GroupStartFrom).Mkstream().Build()
	err := c.client.Do(ctx, cmd).Error()
	if err != nil && !isBusyGroupErr(err) {
		return fmt.Errorf("xgroup create failed stream=%s group=%s: %w", cfg.Stream, cfg.Group, err)
	}

	consumerCmd := c.client.B().XgroupCreateconsumer().Key(cfg.Stream).Group(cfg.Group).Consumer(cfg.Name).Build()
	_ = c.client.Do(ctx, consumerCmd).Error()
	return nil
}

func (c *StreamConsumer) resetGroup(ctx context.Context, cfg StreamConsumerConfig) error {
	destroyCmd := c.client.B().XgroupDestroy().Key(cfg.Stream).Group(cfg.Group).Build()
	if err := c.client.Do(ctx, destroyCmd).Error(); err != nil && !isNoGroupOrNoStreamErr(err) {
		return fmt.Errorf("xgroup destroy failed stream=%s group=%s: %w", cfg.Stream, cfg.Group, err)
	}
	return c.ensureGroup(ctx, cfg)
}

func (c *StreamConsumer) ackWithRetry(ctx context.Context, cfg StreamConsumerConfig, id string) error {
	var lastErr error
	for attempt := 0; attempt < cfg.AckMaxRetries; attempt++ {
		if ctx.Err() != nil {
			return nil
		}

		cmd := c.client.B().Xack().Key(cfg.Stream).Group(cfg.Group).Id(id).Build()
		err := c.client.Do(ctx, cmd).Error()
		if err == nil {
			return nil
		}
		lastErr = err

		if attempt < cfg.AckMaxRetries-1 {
			timer := time.NewTimer(cfg.AckRetryDelay)
			select {
			case <-timer.C:
			case <-ctx.Done():
				timer.Stop()
				return nil
			}
		}
	}
	return lastErr
}

func isBusyGroupErr(err error) bool {
	return valkeyx.IsBusyGroup(err)
}

func isNoGroupOrNoStreamErr(err error) bool {
	if err == nil {
		return false
	}
	msg := err.Error()
	return strings.Contains(msg, "NOGROUP") || strings.Contains(strings.ToLower(msg), "no such key") || strings.Contains(msg, "requires the key to exist")
}

// sleepWithContext: context 취소를 지원하는 sleep
// 정상 대기 완료 시 true, context 취소 시 false 반환
func sleepWithContext(ctx context.Context, delay time.Duration) bool {
	timer := time.NewTimer(delay)
	defer timer.Stop()
	select {
	case <-timer.C:
		return true
	case <-ctx.Done():
		return false
	}
}
