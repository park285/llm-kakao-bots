package bot

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/kapu/hololive-kakao-bot-go/internal/adapter"
	"github.com/kapu/hololive-kakao-bot-go/internal/command"
	"github.com/kapu/hololive-kakao-bot-go/internal/config"
	"github.com/kapu/hololive-kakao-bot-go/internal/constants"
	"github.com/kapu/hololive-kakao-bot-go/internal/domain"
	"github.com/kapu/hololive-kakao-bot-go/internal/iris"
	"github.com/kapu/hololive-kakao-bot-go/internal/service/acl"
	"github.com/kapu/hololive-kakao-bot-go/internal/service/cache"
	"github.com/kapu/hololive-kakao-bot-go/internal/service/database"
	"github.com/kapu/hololive-kakao-bot-go/internal/service/holodex"
	"github.com/kapu/hololive-kakao-bot-go/internal/service/matcher"
	"github.com/kapu/hololive-kakao-bot-go/internal/service/member"
	"github.com/kapu/hololive-kakao-bot-go/internal/service/notification"
	"github.com/kapu/hololive-kakao-bot-go/internal/service/youtube"
	"github.com/kapu/hololive-kakao-bot-go/internal/util"
	appErrors "github.com/kapu/hololive-kakao-bot-go/pkg/errors"
)

var (
	numericRoomRegex = regexp.MustCompile(`^\d+$`)
)

// Bot: 홀로라이브 봇의 핵심 상태와 의존성(서비스, 캐시, 핸들러 등)을 관리하는 메인 구조체
type Bot struct {
	config           *config.Config
	logger           *slog.Logger
	irisClient       iris.Client
	messageAdapter   *adapter.MessageAdapter
	formatter        *adapter.ResponseFormatter
	cache            *cache.Service
	postgres         *database.PostgresService
	holodex          *holodex.Service
	officialProfiles *member.ProfileService
	alarm            *notification.AlarmService
	matcher          *matcher.MemberMatcher
	commandRegistry  *command.Registry
	statsRepo        *youtube.StatsRepository
	acl              *acl.Service
	alarmTicker      *time.Ticker
	alarmStopCh      chan struct{}
	alarmMutex       sync.Mutex
	membersData      domain.MemberDataProvider
	stopCh           chan struct{}
	doneCh           chan struct{}
	selfSender       string
}

// NewBot: 필요한 의존성(Dependencies)을 주입받아 새로운 Bot 인스턴스를 생성하고 초기화한다.
func NewBot(deps *Dependencies) (*Bot, error) {
	if deps == nil {
		return nil, fmt.Errorf("bot dependencies are required")
	}

	deps.Logger.Info("Bot dependency snapshot", slog.Bool("stats_repo", deps.YouTubeStatsRepo != nil))
	if deps.Config == nil {
		return nil, fmt.Errorf("config dependency is required")
	}
	if deps.Logger == nil {
		return nil, fmt.Errorf("logger dependency is required")
	}
	if deps.Client == nil {
		return nil, fmt.Errorf("iris client dependency is required")
	}
	if deps.MessageAdapter == nil {
		return nil, fmt.Errorf("message adapter dependency is required")
	}
	if deps.Formatter == nil {
		return nil, fmt.Errorf("response formatter dependency is required")
	}
	if deps.Cache == nil {
		return nil, fmt.Errorf("cache dependency is required")
	}
	if deps.Postgres == nil {
		return nil, fmt.Errorf("postgres dependency is required")
	}
	if deps.Holodex == nil {
		return nil, fmt.Errorf("holodex dependency is required")
	}
	if deps.Profiles == nil {
		return nil, fmt.Errorf("profile service dependency is required")
	}
	if deps.Alarm == nil {
		return nil, fmt.Errorf("alarm service dependency is required")
	}
	if deps.Matcher == nil {
		return nil, fmt.Errorf("matcher dependency is required")
	}
	if deps.MembersData == nil {
		return nil, fmt.Errorf("member data dependency is required")
	}

	bot := &Bot{
		config:           deps.Config,
		logger:           deps.Logger,
		irisClient:       deps.Client,
		messageAdapter:   deps.MessageAdapter,
		formatter:        deps.Formatter,
		cache:            deps.Cache,
		postgres:         deps.Postgres,
		holodex:          deps.Holodex,
		officialProfiles: deps.Profiles,
		alarm:            deps.Alarm,
		matcher:          deps.Matcher,
		statsRepo:        deps.YouTubeStatsRepo,
		acl:              deps.ACL,
		membersData:      deps.MembersData,
		stopCh:           make(chan struct{}),
		doneCh:           make(chan struct{}),
		selfSender:       util.Normalize(deps.Config.Bot.SelfUser),
	}

	bot.initializeCommands()

	return bot, nil
}

func (b *Bot) initializeCommands() {
	registry := command.NewRegistry()
	b.commandRegistry = registry

	if b.statsRepo == nil && b.postgres != nil {
		b.logger.Info("Stats repository missing in dependencies; creating fallback instance")
		b.statsRepo = youtube.NewYouTubeStatsRepository(b.postgres, b.logger)
	}

	deps := &command.Dependencies{
		Holodex:          b.holodex,
		Cache:            b.cache,
		Alarm:            b.alarm,
		Matcher:          b.matcher,
		OfficialProfiles: b.officialProfiles,
		StatsRepo:        b.statsRepo,
		MembersData:      b.membersData,
		Formatter:        b.formatter,
		SendMessage:      b.sendMessage,
		SendImage:        b.sendImage,
		SendError:        b.sendError,
		Logger:           b.logger,
	}

	deps.Dispatcher = command.NewSequentialDispatcher(registry, b.normalizeCommand)

	b.logger.Info("Stats repository detected", slog.Bool("available", deps.StatsRepo != nil))

	commandsList := []command.Command{
		command.NewHelpCommand(deps),
		command.NewLiveCommand(deps),
		command.NewUpcomingCommand(deps),
		command.NewScheduleCommand(deps),
		command.NewAlarmCommand(deps),
		command.NewMemberInfoCommand(deps),
	}

	if deps.StatsRepo != nil {
		b.logger.Info("Stats command enabled")
		commandsList = append(commandsList, command.NewStatsCommand(deps))
	}

	for _, cmd := range commandsList {
		registry.Register(cmd)
	}

	b.logger.Info("Commands initialized", slog.Int("count", registry.Count()))
}

// Start: 봇 서비스를 시작한다. Redis/Iris 연결 확인, 알림 스케줄러 실행 등을 수행하며 Context가 종료될 때까지 대기한다.
func (b *Bot) Start(ctx context.Context) error {
	b.logger.Info("Starting Hololive KakaoTalk Bot...")

	if err := b.cache.WaitUntilReady(ctx, constants.ValkeyConfig.ReadyTimeout); err != nil {
		return fmt.Errorf("valkey connection timeout: %w", err)
	}
	b.logger.Info("Valkey connected")

	if !b.irisClient.Ping(ctx) {
		return fmt.Errorf("iris server connection failed")
	}
	b.logger.Info("Iris server connected")

	b.startAlarmChecker(ctx)

	b.logger.Info("Bot started successfully")

	select {
	case <-ctx.Done():
		b.logger.Info("Context canceled, shutting down...")
		return fmt.Errorf("context canceled: %w", ctx.Err())
	case <-b.stopCh:
		b.logger.Info("Stop signal received")
		return nil
	}
}

// HandleMessage: Iris webhook으로부터 수신한 메시지를 처리합니다.
// HTTP webhook 핸들러에서 호출하기 위해 public으로 노출됩니다.
func (b *Bot) HandleMessage(ctx context.Context, message *iris.Message) {
	commandType := "unknown"

	isNumericRoom := message.Room != "" && numericRoomRegex.MatchString(message.Room)
	chatID := message.Room
	if !isNumericRoom && message.JSON != nil {
		chatID = message.JSON.ChatID
	}

	// 한글 방 이름 유지
	roomName := message.Room

	// userID와 userName 분리
	userID := "unknown"
	userName := userID // 기본값

	if message.JSON != nil && message.JSON.UserID != "" {
		userID = message.JSON.UserID // 숫자 ID
		userName = userID            // userName도 업데이트
	}

	if message.Sender != nil {
		userName = *message.Sender // 한글 이름 우선
	}

	defer func() {
		if r := recover(); r != nil {
			b.logger.Error("Panic in handleMessage",
				slog.Any("panic", r),
				slog.String("command", commandType),
			)
		}
	}()

	if b.isSelfSender(userName) {
		b.logger.Debug("Skipping self-issued message",
			slog.String("user", userName),
			slog.String("room", chatID),
			slog.String("payload", message.Msg),
		)
		return
	}

	// ACL: 허용된 방이 아니면 메시지 무시
	if b.acl != nil && !b.acl.IsRoomAllowed(roomName, chatID) {
		b.logger.Debug("Room not in ACL whitelist, ignoring message",
			slog.String("room", chatID),
			slog.String("room_name", roomName),
			slog.String("user_name", userName),
		)
		return
	}

	parsed := b.messageAdapter.ParseMessage(message)
	commandType = parsed.Type.String()

	if parsed.Type == domain.CommandUnknown {
		b.logger.Debug("Unknown command ignored",
			slog.String("msg", message.Msg),
			slog.String("room", chatID),
			slog.String("user_name", userName),
		)
		return // 알 수 없는 명령어는 무시함
	}

	b.logger.Info("Command received",
		slog.String("raw", parsed.RawMessage),
		slog.String("type", commandType),
		slog.String("user_id", userID),
		slog.String("user_name", userName),
		slog.String("room", chatID),
		slog.String("room_name", roomName),
	)

	cmdCtx := domain.NewCommandContext(chatID, roomName, userID, userName, message.Msg, false)

	if err := b.executeCommand(ctx, cmdCtx, parsed.Type, parsed.Params); err != nil {
		b.logger.Error("Failed to execute command", slog.Any("error", err))
		errorMsg := b.getErrorMessage(err, commandType)
		if chatID != "" {
			b.sendError(ctx, chatID, errorMsg)
		}
	}
}

func (b *Bot) executeCommand(ctx context.Context, cmdCtx *domain.CommandContext, cmdType domain.CommandType, params map[string]any) error {
	if b.commandRegistry == nil {
		return fmt.Errorf("command registry is not initialized")
	}

	key, normalizedParams := b.normalizeCommand(cmdType, params)

	if err := b.commandRegistry.Execute(ctx, cmdCtx, key, normalizedParams); err != nil {
		if errors.Is(err, command.ErrUnknownCommand) {
			b.logger.Warn("Unknown command", slog.String("type", cmdType.String()))
			if sendErr := b.sendMessage(ctx, cmdCtx.Room, adapter.ErrUnknownCommand); sendErr != nil {
				return fmt.Errorf("failed to send unknown command message: %w", sendErr)
			}
			return nil
		}
		return fmt.Errorf("execute command: %w", err)
	}

	return nil
}

func (b *Bot) normalizeCommand(cmdType domain.CommandType, params map[string]any) (string, map[string]any) {
	typeStr := util.Normalize(cmdType.String())

	if strings.HasPrefix(typeStr, "alarm_") {
		action := strings.TrimPrefix(typeStr, "alarm_")
		newParams := make(map[string]any)
		for k, v := range params {
			newParams[k] = v
		}
		newParams["action"] = action
		return "alarm", newParams
	}

	if typeStr == "alarm" {
		if _, hasAction := params["action"]; !hasAction {
			newParams := make(map[string]any)
			for k, v := range params {
				newParams[k] = v
			}
			newParams["action"] = "list"
			return "alarm", newParams
		}
	}

	return typeStr, params
}

func (b *Bot) isSelfSender(sender string) bool {
	canonical := util.Normalize(sender)
	if canonical == "" || b.selfSender == "" {
		return false
	}
	return canonical == b.selfSender
}

func (b *Bot) sendMessage(ctx context.Context, room, message string) error {
	ctx, cancel := context.WithTimeout(ctx, constants.RequestTimeout.BotCommand)
	defer cancel()

	if err := b.irisClient.SendMessage(ctx, room, message); err != nil {
		serviceErr := appErrors.NewServiceError("failed to send message", "iris", "send_message", err)
		return fmt.Errorf("failed to send message to room %s: %w", room, serviceErr)
	}
	return nil
}

func (b *Bot) sendImage(ctx context.Context, room, imageBase64 string) error {
	ctx, cancel := context.WithTimeout(ctx, constants.RequestTimeout.BotCommand)
	defer cancel()

	if err := b.irisClient.SendImage(ctx, room, imageBase64); err != nil {
		serviceErr := appErrors.NewServiceError("failed to send image", "iris", "send_image", err)
		return fmt.Errorf("failed to send image to room %s: %w", room, serviceErr)
	}
	return nil
}

func (b *Bot) sendError(ctx context.Context, room, errorMsg string) error {
	message := b.formatter.FormatError(errorMsg)
	if err := b.sendMessage(ctx, room, message); err != nil {
		return fmt.Errorf("failed to send error message: %w", err)
	}
	return nil
}

func (b *Bot) getErrorMessage(err error, commandType string) string {
	if err == nil {
		return ""
	}

	msg := err.Error()

	if strings.Contains(msg, "외부 AI 서비스 장애") {
		return msg
	}

	// 서비스 에러 체크 (Iris 연결 실패)
	var serviceErr *appErrors.ServiceError
	if errors.As(err, &serviceErr) && strings.EqualFold(serviceErr.Service, "iris") {
		return adapter.ErrIrisConnectionFailed
	}

	// API 에러 체크 (외부 API 호출 실패)
	var apiErr *appErrors.APIError
	if errors.As(err, &apiErr) {
		return adapter.ErrExternalAPICallFailed
	}

	// 키 로테이션 에러 체크
	var keyRotationErr *appErrors.KeyRotationError
	if errors.As(err, &keyRotationErr) {
		return adapter.ErrExternalAPICallFailed
	}

	// 캐시 에러 체크
	var cacheErr *appErrors.CacheError
	if errors.As(err, &cacheErr) {
		return adapter.ErrCacheConnectionFailed
	}

	// 유효성 검사 에러 체크
	var validationErr *appErrors.ValidationError
	if errors.As(err, &validationErr) {
		return msg
	}

	if strings.Contains(msg, "Valkey") || strings.Contains(msg, "cache") {
		return adapter.ErrCacheConnectionFailed
	}

	return fmt.Sprintf(adapter.ErrCommandProcessingFailed, commandType)
}

func (b *Bot) startAlarmChecker(ctx context.Context) {
	interval := b.config.Notification.CheckInterval
	b.alarmTicker = time.NewTicker(interval)
	b.alarmStopCh = make(chan struct{})

	b.logger.Info("Alarm checker started", slog.Duration("interval", interval))

	go func() {
		for {
			select {
			case <-b.alarmTicker.C:
				b.performAlarmCheck(ctx)
			case <-b.alarmStopCh:
				b.logger.Info("Alarm checker stopped")
				return
			case <-ctx.Done():
				b.logger.Info("Alarm checker context canceled")
				return
			}
		}
	}()
}

func (b *Bot) performAlarmCheck(ctx context.Context) {
	if !b.alarmMutex.TryLock() {
		b.logger.Debug("Alarm check already in progress, skipping")
		return
	}
	defer b.alarmMutex.Unlock()

	childCtx, cancel := context.WithTimeout(ctx, constants.RequestTimeout.BotAlarmCheck)
	defer cancel()

	notifications, err := b.alarm.CheckUpcomingStreams(childCtx)
	if err != nil {
		b.logger.Error("Alarm check failed", slog.Any("error", err))
		return
	}

	grouped := groupAlarmNotifications(notifications)

	// 병렬 알림 전송 (CONVENTIONS.md Phase 2 최적화)
	var wg sync.WaitGroup
	for _, group := range grouped {
		if len(group.notifications) == 0 {
			continue
		}

		wg.Add(1)
		go func(g alarmNotificationGroup) {
			defer wg.Done()

			var message string
			if len(g.notifications) == 1 {
				message = b.formatter.AlarmNotification(g.notifications[0])
			} else {
				message = b.formatter.AlarmNotificationGroup(g.minutesUntil, g.notifications)
			}

			if util.TrimSpace(message) == "" {
				return
			}

			if err := b.sendMessage(childCtx, g.roomID, message); err != nil {
				b.logger.Error("Failed to send alarm notification",
					slog.String("room", g.roomID),
					slog.Int("notifications", len(g.notifications)),
					slog.Any("error", err),
				)
				return
			}

			for _, notif := range g.notifications {
				if notif == nil || notif.Stream == nil || notif.Stream.StartScheduled == nil {
					continue
				}
				if err := b.alarm.MarkAsNotified(childCtx, notif.Stream.ID, *notif.Stream.StartScheduled, notif.MinutesUntil); err != nil {
					b.logger.Warn("Failed to mark as notified",
						slog.String("stream_id", notif.Stream.ID),
						slog.Any("error", err),
					)
				}
			}
		}(*group)
	}
	wg.Wait()
}

type alarmNotificationGroup struct {
	roomID        string
	minutesUntil  int
	notifications []*domain.AlarmNotification
}

func groupAlarmNotifications(notifications []*domain.AlarmNotification) []*alarmNotificationGroup {
	if len(notifications) == 0 {
		return []*alarmNotificationGroup{}
	}

	groups := make([]*alarmNotificationGroup, 0)
	index := make(map[string]int)

	for _, notif := range notifications {
		if notif == nil {
			continue
		}

		key := buildAlarmGroupKey(notif)
		if idx, ok := index[key]; ok {
			group := groups[idx]
			group.notifications = append(group.notifications, notif)
			if notif.MinutesUntil >= 0 && (group.minutesUntil < 0 || notif.MinutesUntil < group.minutesUntil) {
				group.minutesUntil = notif.MinutesUntil
			}
			continue
		}

		group := &alarmNotificationGroup{
			roomID:        notif.RoomID,
			minutesUntil:  notif.MinutesUntil,
			notifications: []*domain.AlarmNotification{notif},
		}

		groups = append(groups, group)
		index[key] = len(groups) - 1
	}

	return groups
}

func buildAlarmGroupKey(notif *domain.AlarmNotification) string {
	if notif == nil {
		return ""
	}

	if notif.Stream != nil && notif.Stream.StartScheduled != nil {
		scheduled := notif.Stream.StartScheduled.Truncate(time.Minute)
		return fmt.Sprintf("%s|scheduled|%d", notif.RoomID, scheduled.Unix())
	}

	return fmt.Sprintf("%s|minutes|%d", notif.RoomID, notif.MinutesUntil)
}

// Shutdown: 봇의 리소스를 정리하고 실행 중인 작업(알림 체커 등)을 안전하게 종료한다.
func (b *Bot) Shutdown(ctx context.Context) error {
	b.logger.Info("Shutting down bot...")

	if b.alarmTicker != nil {
		b.alarmTicker.Stop()
	}
	if b.alarmStopCh != nil {
		close(b.alarmStopCh)
	}

	if b.cache != nil {
		if err := b.cache.Close(); err != nil {
			b.logger.Warn("Error closing cache", slog.Any("error", err))
		}
	}

	if b.postgres != nil {
		if err := b.postgres.Close(); err != nil {
			b.logger.Warn("Error closing postgres", slog.Any("error", err))
		}
	}

	b.logger.Info("Bot shutdown complete")
	close(b.doneCh)
	return nil
}
