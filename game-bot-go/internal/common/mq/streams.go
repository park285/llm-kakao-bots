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

	for {
		if ctx.Err() != nil {
			return nil
		}

		messages, err := c.readBatch(ctx, cfg)
		if err != nil {
			if valkeyx.IsNil(err) || (errors.Is(err, context.DeadlineExceeded) && ctx.Err() == nil) {
				continue
			}
			if errors.Is(err, context.Canceled) && ctx.Err() != nil {
				return nil
			}
			c.logger.Warn("xreadgroup_failed", "err", err, "stream", cfg.Stream, "group", cfg.Group)
			continue
		}

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
	handleErr := handler(ctx, msg)
	if handleErr != nil {
		c.logger.Error("message_handler_failed", "err", handleErr, "stream", cfg.Stream, "group", cfg.Group, "id", msg.ID)
		if !cfg.AckOnError {
			return
		}
	}

	if errAck := c.ackWithRetry(ctx, cfg, msg.ID); errAck != nil {
		c.logger.Warn("xack_failed", "err", errAck, "stream", cfg.Stream, "group", cfg.Group, "id", msg.ID)
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
