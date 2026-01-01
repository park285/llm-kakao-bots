package mq

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net"
	"strings"
	"time"

	"github.com/sourcegraph/conc/pool"
	"github.com/valkey-io/valkey-go"

	"github.com/kapu/hololive-kakao-bot-go/internal/constants"
	"github.com/kapu/hololive-kakao-bot-go/internal/iris"
	"github.com/kapu/hololive-kakao-bot-go/internal/service/cache"
)

// ValkeyMQConfig: Redis(Valkey) 스트림 기반 메시지 큐 연결 설정
type ValkeyMQConfig struct {
	Host          string
	Port          int
	Password      string
	StreamKey     string
	ConsumerGroup string
	ConsumerName  string
	ReadCount     int64
	BlockTimeout  time.Duration
	WorkerCount   int // Worker pool 크기
	// ReplyStreamMaxLen: reply stream의 최대 길이 (0 이하면 trim 비활성)
	ReplyStreamMaxLen int64
}

// newValkeyClient: 공통 Valkey 클라이언트 생성 로직
func newValkeyClient(host string, port int, password string, logger *slog.Logger) (valkey.Client, error) {
	addr := fmt.Sprintf("%s:%d", host, port)

	maxAttempts := constants.MQConfig.InitRetryCount
	if maxAttempts < 1 {
		maxAttempts = 1
	}

	var lastErr error
	for attempt := 1; attempt <= maxAttempts; attempt++ {
		client, err := valkey.NewClient(valkey.ClientOption{
			InitAddress:       []string{addr},
			Password:          password,
			SelectDB:          0,
			ConnWriteTimeout:  constants.MQConfig.ConnWriteTimeout,
			BlockingPoolSize:  constants.MQConfig.BlockingPoolSize,
			PipelineMultiplex: constants.MQConfig.PipelineMultiplex,
			Dialer:            net.Dialer{Timeout: constants.MQConfig.DialTimeout},
		})
		if err == nil {
			return client, nil
		}

		lastErr = err
		if logger != nil {
			logger.Warn("MQ_CLIENT_INIT_RETRY",
				slog.String("addr", addr),
				slog.Int("attempt", attempt),
				slog.Int("max_attempts", maxAttempts),
				slog.Any("error", err),
			)
		}

		if attempt < maxAttempts {
			// 초기화 단계에서는 context가 없으므로 time.Sleep 사용
			time.Sleep(constants.MQConfig.RetryDelay)
		}
	}

	return nil, fmt.Errorf("failed to create valkey client after retries: %w", lastErr)
}

// ValkeyMQClient: 메시지 큐에 메시지를 발행(Publish)하는 클라이언트
// Iris 서버로 보낼 응답 메시지를 Redis Stream에 적재합니다.
type ValkeyMQClient struct {
	cfg    ValkeyMQConfig
	client valkey.Client
	logger *slog.Logger
}

// NewValkeyMQClient: 새로운 ValkeyMQClient 인스턴스를 생성하고 연결을 초기화합니다.
func NewValkeyMQClient(ctx context.Context, cfg ValkeyMQConfig, logger *slog.Logger) *ValkeyMQClient {
	client, err := newValkeyClient(cfg.Host, cfg.Port, cfg.Password, logger)
	if err != nil {
		logger.Error("Failed to create MQ client", slog.Any("error", err))
		return nil
	}
	if err := mqLuaRegistry.Preload(ctx, client); err != nil && logger != nil {
		logger.Warn("MQ_LUA_PRELOAD_FAILED", slog.Any("error", err))
	}

	return &ValkeyMQClient{
		cfg:    cfg,
		client: client,
		logger: logger,
	}
}

// SendMessage: 지정된 채팅방으로 보낼 텍스트 메시지를 Redis Stream에 추가(발행)합니다.
func (c *ValkeyMQClient) SendMessage(ctx context.Context, room, message string) error {
	streamKey := constants.MQConfig.ReplyStreamKey

	fieldValues := []string{
		"chatId", room,
		"text", message,
		"threadId", "",
		"type", "final",
	}

	var args []string
	if c.cfg.ReplyStreamMaxLen > 0 {
		args = append(args, "MAXLEN", "~", fmt.Sprintf("%d", c.cfg.ReplyStreamMaxLen))
	}
	args = append(args, "*")
	args = append(args, fieldValues...)

	cmd := c.client.B().Arbitrary("XADD").Keys(streamKey).Args(args...).Build()

	if err := c.client.Do(ctx, cmd).Error(); err != nil {
		c.logger.Error("MQ_REPLY_ERROR",
			slog.String("stream", streamKey),
			slog.String("room", room),
			slog.Any("error", err),
		)
		return fmt.Errorf("failed to publish reply to message queue: %w", err)
	}

	c.logger.Info("MQ_REPLY_PUBLISHED",
		slog.String("stream", streamKey),
		slog.String("room", room),
	)

	return nil
}

// SendImage: 지정된 채팅방으로 보낼 이미지 메시지를 발행한다. (현재 로그만 남기고 스킵)
func (c *ValkeyMQClient) SendImage(ctx context.Context, room, imageBase64 string) error {
	c.logger.Info("MQ_SEND_IMAGE_SKIPPED",
		slog.String("room", room),
	)
	return nil
}

// Ping: Redis 서버와의 연결 상태를 점검합니다.
func (c *ValkeyMQClient) Ping(ctx context.Context) bool {
	return c.client.Do(ctx, c.client.B().Ping().Build()).Error() == nil
}

// GetConfig: 현재 설정 정보를 반환한다. (인터페이스 준수용, 빈 설정 반환)
func (c *ValkeyMQClient) GetConfig(ctx context.Context) (*iris.Config, error) {
	return &iris.Config{}, nil
}

// Decrypt: 암호화된 데이터를 복호화한다. (현재 로직 없음, Pass-through)
func (c *ValkeyMQClient) Decrypt(ctx context.Context, data string) (string, error) {
	return data, nil
}

// MessageHandler: 큐에서 수신한 메시지를 실제로 처리(Bot 비즈니스 로직 호출)하는 인터페이스
type MessageHandler interface {
	HandleMessage(ctx context.Context, msg *iris.Message)
}

// ValkeyMQConsumer: Redis 스트림을 구독하고 메시지를 수신하여 처리하는 컨슈머
// Worker Pool을 사용하여 병렬 처리를 지원하며, Consumer Group을 통해 로드 밸런싱을 수행합니다.
type ValkeyMQConsumer struct {
	cfg    ValkeyMQConfig
	client valkey.Client
	logger *slog.Logger
	bot    MessageHandler
	cache  *cache.Service
}

// NewValkeyMQConsumer: 새로운 ValkeyMQConsumer 인스턴스를 생성합니다.
func NewValkeyMQConsumer(ctx context.Context, cfg ValkeyMQConfig, logger *slog.Logger, handler MessageHandler, cacheService *cache.Service) *ValkeyMQConsumer {
	// Worker count 기본값
	if cfg.WorkerCount == 0 {
		cfg.WorkerCount = constants.MQConfig.WorkerCount
	}

	client, err := newValkeyClient(cfg.Host, cfg.Port, cfg.Password, logger)
	if err != nil {
		logger.Error("Failed to create MQ consumer", slog.Any("error", err))
		return nil
	}
	if err := mqLuaRegistry.Preload(ctx, client); err != nil && logger != nil {
		logger.Warn("MQ_LUA_PRELOAD_FAILED", slog.Any("error", err))
	}

	return &ValkeyMQConsumer{
		cfg:    cfg,
		client: client,
		logger: logger,
		bot:    handler,
		cache:  cacheService,
	}
}

// Start: 별도의 고루틴에서 메시지 수신 루프(run)를 시작합니다.
func (c *ValkeyMQConsumer) Start(ctx context.Context) {
	go c.run(ctx)
}

func (c *ValkeyMQConsumer) run(ctx context.Context) {
	streamKey := c.cfg.StreamKey
	group := c.cfg.ConsumerGroup
	consumer := c.cfg.ConsumerName
	readCount := c.cfg.ReadCount
	if readCount <= 0 {
		readCount = constants.MQConfig.ReadCount
	}
	blockTimeout := c.cfg.BlockTimeout
	if blockTimeout <= 0 {
		blockTimeout = constants.MQConfig.BlockTimeout
	}

	c.ensureGroup(ctx, streamKey, group)

	// Worker pool 생성
	workerPool := pool.New().WithMaxGoroutines(c.cfg.WorkerCount)

	for {
		select {
		case <-ctx.Done():
			c.logger.Info("MQ_CONSUMER_STOPPED",
				slog.String("stream", streamKey),
			)
			return
		default:
		}

		// XREADGROUP으로 메시지 읽기 (별도 timeout context 사용)
		cmd := c.client.B().Xreadgroup().
			Group(group, consumer).
			Count(readCount).
			Block(blockTimeout.Milliseconds()). // 블록 타임아웃
			Streams().
			Key(streamKey).
			Id(">").
			Build()

			// 별도 timeout context: parent context 취소와 분리
			// BlockTimeout + 2초 여유를 두어 Block 완료 후 응답 처리 시간 확보
		readTimeout := blockTimeout + 2*time.Second
		readCtx, readCancel := context.WithTimeout(context.WithoutCancel(ctx), readTimeout)
		resp := c.client.Do(readCtx, cmd)
		readCancel()

		if resp.Error() != nil && !valkey.IsValkeyNil(resp.Error()) {
			// Parent context가 취소되었으면 graceful shutdown
			if ctx.Err() != nil {
				c.logger.Info("MQ_CONSUMER_STOPPING",
					slog.String("stream", streamKey),
					slog.String("reason", "parent context canceled"),
				)
				return
			}

			// NOGROUP 에러 감지 시 consumer group 자동 재생성
			if isNogroupErr(resp.Error()) {
				c.logger.Warn("MQ_NOGROUP_DETECTED",
					slog.String("stream", streamKey),
					slog.String("group", group),
					slog.Any("error", resp.Error()),
				)
				c.ensureGroup(ctx, streamKey, group)
				if !sleepWithContext(ctx, constants.MQConfig.RetryDelay) {
					return // context 취소 시 종료
				}
				continue
			}

			// Read context timeout/canceled: 연결 지연 또는 일시적 문제
			if errors.Is(resp.Error(), context.Canceled) || errors.Is(resp.Error(), context.DeadlineExceeded) {
				c.logger.Warn("MQ_READ_TIMEOUT",
					slog.String("stream", streamKey),
					slog.Duration("timeout", readTimeout),
				)
				continue
			}

			// 실제 오류만 ERROR로 로깅
			c.logger.Error("MQ_READ_ERROR",
				slog.String("stream", streamKey),
				slog.Any("error", resp.Error()),
			)
			if !sleepWithContext(ctx, constants.MQConfig.RetryDelay) {
				return // context 취소 시 종료
			}
			continue
		}

		// 응답 파싱
		streams, err := resp.AsXRead()
		if err != nil {
			if !valkey.IsValkeyNil(err) {
				c.logger.Warn("MQ_PARSE_ERROR", slog.Any("error", err))
			}
			continue
		}

		if len(streams) == 0 {
			continue
		}

		// Worker pool로 병렬 처리
		for streamName, messages := range streams {
			for _, msg := range messages {
				streamNameCopy := streamName
				msgCopy := msg
				workerPool.Go(func() {
					c.handleEntry(ctx, streamNameCopy, group, msgCopy)
				})
			}
		}
	}
}

func (c *ValkeyMQConsumer) ensureGroup(ctx context.Context, streamKey, group string) {
	cmd := c.client.B().XgroupCreate().Key(streamKey).Group(group).Id("$").Mkstream().Build()
	err := c.client.Do(ctx, cmd).Error()

	if err != nil && !isBusyGroupErr(err) {
		c.logger.Warn("MQ_GROUP_CREATE_FAILED",
			slog.String("stream", streamKey),
			slog.String("group", group),
			slog.Any("error", err),
		)
		return
	}

	c.logger.Info("MQ_GROUP_READY",
		slog.String("stream", streamKey),
		slog.String("group", group),
	)
}

func isBusyGroupErr(err error) bool {
	if err == nil {
		return false
	}
	msg := err.Error()
	return msg == "BUSYGROUP Consumer Group name already exists" || msg == "BUSYGROUP Consumer Group name already exists."
}

func isNogroupErr(err error) bool {
	if err == nil {
		return false
	}
	msg := err.Error()
	return strings.Contains(msg, "NOGROUP")
}

func (c *ValkeyMQConsumer) handleEntry(ctx context.Context, streamKey, group string, msg valkey.XRangeEntry) {
	fields := msg.FieldValues

	text := getField(fields, "text")
	room := getField(fields, "room")
	sender := getField(fields, "sender")

	if text == "" || room == "" {
		c.logger.Warn("MQ_MESSAGE_SKIPPED",
			slog.String("stream", streamKey),
			slog.String("id", msg.ID),
		)
		_ = c.ackMessage(ctx, streamKey, group, msg.ID)
		return
	}

	// 멱등성 체크 및 처리 시작
	shouldProcess := c.checkIdempotency(ctx, streamKey, group, msg.ID)
	if shouldProcess != 1 {
		return
	}

	c.logger.Info("MQ_MESSAGE_RECEIVED",
		slog.String("stream", streamKey),
		slog.String("id", msg.ID),
		slog.String("room", room),
		slog.String("sender", sender),
	)

	// 메시지 처리
	c.processMessage(ctx, streamKey, group, msg.ID, text, room, sender)
}

// checkIdempotency: 멱등성 체크 (1=처리 필요, 0=이미 완료, -1=처리 중)
func (c *ValkeyMQConsumer) checkIdempotency(ctx context.Context, streamKey, group, msgID string) int64 {
	idempotencyKey := fmt.Sprintf("mq:processed:%s", msgID)

	processingTTLArg := fmt.Sprintf("%d", int64(constants.MQConfig.IdempotencyProcessingTTL.Seconds()))
	resp, err := mqLuaRegistry.Exec(
		ctx,
		c.client,
		scriptProcessWithIdempotency,
		[]string{idempotencyKey, streamKey},
		[]string{group, msgID, processingTTLArg},
	)
	if err != nil {
		c.logger.Error("MQ_IDEMPOTENCY_SCRIPT_MISSING",
			slog.String("stream", streamKey),
			slog.String("id", msgID),
			slog.Any("error", err),
		)
		return 0
	}

	shouldProcess, err := resp.AsInt64()
	if err != nil {
		c.logger.Error("MQ_IDEMPOTENCY_CHECK_FAILED",
			slog.String("stream", streamKey),
			slog.String("id", msgID),
			slog.Any("error", err),
		)
		return 0
	}

	switch shouldProcess {
	case 0:
		c.logger.Debug("MQ_MESSAGE_ALREADY_PROCESSED",
			slog.String("stream", streamKey),
			slog.String("id", msgID),
		)
	case -1:
		c.logger.Debug("MQ_MESSAGE_ALREADY_PROCESSING",
			slog.String("stream", streamKey),
			slog.String("id", msgID),
		)
	}

	return shouldProcess
}

// processMessage: 실제 메시지 처리 및 완료 마킹
func (c *ValkeyMQConsumer) processMessage(ctx context.Context, streamKey, group, msgID, text, room, sender string) {
	var senderPtr *string
	if sender != "" {
		senderPtr = &sender
	}

	irisMsg := &iris.Message{
		Msg:    text,
		Room:   room,
		Sender: senderPtr,
	}

	// 메시지 처리
	c.bot.HandleMessage(ctx, irisMsg)

	// 처리 완료 + ACK
	c.markComplete(ctx, streamKey, group, msgID)
}

// markComplete: 처리 완료 마킹 및 ACK
func (c *ValkeyMQConsumer) markComplete(ctx context.Context, streamKey, group, msgID string) {
	idempotencyKey := fmt.Sprintf("mq:processed:%s", msgID)
	retentionTTLArg := fmt.Sprintf("%d", int64(constants.MQConfig.IdempotencyTTL.Seconds()))

	completeResp, err := mqLuaRegistry.Exec(
		ctx,
		c.client,
		scriptCompleteProcessing,
		[]string{idempotencyKey, streamKey},
		[]string{group, msgID, retentionTTLArg},
	)
	if err != nil {
		c.logger.Error("MQ_COMPLETE_SCRIPT_MISSING",
			slog.String("stream", streamKey),
			slog.String("id", msgID),
			slog.Any("error", err),
		)
		return
	}
	if err := completeResp.Error(); err != nil {
		c.logger.Error("MQ_COMPLETE_PROCESSING_FAILED",
			slog.String("stream", streamKey),
			slog.String("id", msgID),
			slog.Any("error", err),
		)
	}
}

func (c *ValkeyMQConsumer) ackMessage(ctx context.Context, streamKey, group, msgID string) error {
	cmd := c.client.B().Xack().Key(streamKey).Group(group).Id(msgID).Build()
	if err := c.client.Do(ctx, cmd).Error(); err != nil {
		return fmt.Errorf("failed to acknowledge message: %w", err)
	}
	return nil
}

func getField(fields map[string]string, key string) string {
	if val, ok := fields[key]; ok {
		return val
	}
	return ""
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
