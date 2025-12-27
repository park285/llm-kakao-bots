package app

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"

	"github.com/park285/llm-kakao-bots/game-bot-go/internal/common/bootstrap"
	"github.com/park285/llm-kakao-bots/game-bot-go/internal/common/di"
	"github.com/park285/llm-kakao-bots/game-bot-go/internal/common/httpserver"
	"github.com/park285/llm-kakao-bots/game-bot-go/internal/common/llmrest"
	"github.com/park285/llm-kakao-bots/game-bot-go/internal/common/messageprovider"
	commonmq "github.com/park285/llm-kakao-bots/game-bot-go/internal/common/mq"
	"github.com/park285/llm-kakao-bots/game-bot-go/internal/common/mqmsg"
	qassets "github.com/park285/llm-kakao-bots/game-bot-go/internal/twentyq/assets"
	qconfig "github.com/park285/llm-kakao-bots/game-bot-go/internal/twentyq/config"
	qhttpapi "github.com/park285/llm-kakao-bots/game-bot-go/internal/twentyq/httpapi"
	qmodel "github.com/park285/llm-kakao-bots/game-bot-go/internal/twentyq/model"
	qmq "github.com/park285/llm-kakao-bots/game-bot-go/internal/twentyq/mq"
	qredis "github.com/park285/llm-kakao-bots/game-bot-go/internal/twentyq/redis"
	qrepo "github.com/park285/llm-kakao-bots/game-bot-go/internal/twentyq/repository"
	qsecurity "github.com/park285/llm-kakao-bots/game-bot-go/internal/twentyq/security"
	qsvc "github.com/park285/llm-kakao-bots/game-bot-go/internal/twentyq/service"
)

type twentyQStores struct {
	lockManager           *qredis.LockManager
	processingLockService *qredis.ProcessingLockService
	pendingStore          *qredis.PendingMessageStore

	sessionStore      *qredis.SessionStore
	categoryStore     *qredis.CategoryStore
	historyStore      *qredis.HistoryStore
	hintCountStore    *qredis.HintCountStore
	playerStore       *qredis.PlayerStore
	wrongGuessStore   *qredis.WrongGuessStore
	topicHistoryStore *qredis.TopicHistoryStore
	voteStore         *qredis.SurrenderVoteStore
	guessRateLimiter  *qredis.GuessRateLimiter
}

func newTwentyQStores(client di.DataValkeyClient, logger *slog.Logger) *twentyQStores {
	return &twentyQStores{
		lockManager:           qredis.NewLockManager(client.Client, logger),
		processingLockService: qredis.NewProcessingLockService(client.Client, logger),
		pendingStore:          qredis.NewPendingMessageStore(client.Client, logger),
		sessionStore:          qredis.NewSessionStore(client.Client, logger),
		categoryStore:         qredis.NewCategoryStore(client.Client, logger),
		historyStore:          qredis.NewHistoryStore(client.Client, logger),
		hintCountStore:        qredis.NewHintCountStore(client.Client, logger),
		playerStore:           qredis.NewPlayerStore(client.Client, logger),
		wrongGuessStore:       qredis.NewWrongGuessStore(client.Client, logger),
		topicHistoryStore:     qredis.NewTopicHistoryStore(client.Client, logger),
		voteStore:             qredis.NewSurrenderVoteStore(client.Client, logger),
		guessRateLimiter:      qredis.NewGuessRateLimiter(client.Client, "twentyq"),
	}
}

func newTwentyQRiddleService(
	cfg *qconfig.Config,
	restClient *llmrest.Client,
	msgProvider *messageprovider.Provider,
	stores *twentyQStores,
	statsRecorder *qsvc.StatsRecorder,
	logger *slog.Logger,
) *qsvc.RiddleService {
	topicSelector := qsvc.NewTopicSelector(logger)
	return qsvc.NewRiddleService(
		restClient,
		cfg.Commands.Prefix,
		msgProvider,
		stores.lockManager,
		stores.sessionStore,
		stores.categoryStore,
		stores.historyStore,
		stores.hintCountStore,
		stores.playerStore,
		stores.wrongGuessStore,
		stores.topicHistoryStore,
		stores.voteStore,
		stores.guessRateLimiter,
		topicSelector,
		statsRecorder,
		logger,
	)
}

type twentyQAdminServices struct {
	statsService *qsvc.StatsService
	adminHandler *qsvc.AdminHandler
	usageHandler *qsvc.UsageHandler
}

func newTwentyQAdminServices(
	cfg *qconfig.Config,
	db *gorm.DB,
	restClient *llmrest.Client,
	msgProvider *messageprovider.Provider,
	stores *twentyQStores,
	riddleService *qsvc.RiddleService,
	logger *slog.Logger,
) *twentyQAdminServices {
	statsService := qsvc.NewStatsService(db, stores.sessionStore, msgProvider, logger)
	adminHandler := qsvc.NewAdminHandler(cfg.Admin.UserIDs, riddleService, stores.sessionStore, msgProvider, logger)
	usageHandler := qsvc.NewUsageHandler(cfg.Admin.UserIDs, restClient, msgProvider, nil, logger)
	return &twentyQAdminServices{
		statsService: statsService,
		adminHandler: adminHandler,
		usageHandler: usageHandler,
	}
}

func newTwentyQReplyPublisher(cfg *qconfig.Config, mqValkey di.MQValkeyClient, logger *slog.Logger) *qmq.ReplyPublisher {
	return commonmq.NewBotReplyPublisher(
		mqValkey.Client,
		logger,
		cfg.Valkey.ReplyStreamKey,
		cfg.Valkey.StreamMaxLen,
	)
}

func newTwentyQStreamConsumer(cfg *qconfig.Config, mqValkey di.MQValkeyClient, logger *slog.Logger) *commonmq.StreamConsumer {
	return commonmq.NewBotStreamConsumer(
		mqValkey.Client,
		logger,
		cfg.Valkey.StreamKey,
		cfg.Valkey.ConsumerGroup,
		cfg.Valkey.ConsumerName,
		cfg.Valkey.BatchSize,
		cfg.Valkey.BlockTimeout,
		cfg.Valkey.Concurrency,
		cfg.Valkey.ResetConsumerGroupOnStartup,
	)
}

type twentyQQueuedExecutor struct {
	service       *qmq.GameMessageService
	commandParser *qmq.CommandParser
}

func (e *twentyQQueuedExecutor) Execute(
	ctx context.Context,
	chatID string,
	pending qmodel.PendingMessage,
	emit func(mqmsg.OutboundMessage) error,
) error {
	if e.service == nil {
		return errors.New("game message service not initialized")
	}

	if pending.IsChainBatch && len(pending.BatchQuestions) > 0 {
		if err := e.service.HandleQueuedChainBatch(ctx, chatID, pending, emit); err != nil {
			return fmt.Errorf("handle queued chain batch failed: %w", err)
		}
		return nil
	}

	cmd := e.commandParser.Parse(pending.Content)
	if cmd == nil {
		return nil
	}

	inbound := mqmsg.InboundMessage{
		ChatID:   chatID,
		UserID:   pending.UserID,
		Content:  pending.Content,
		ThreadID: pending.ThreadID,
		Sender:   pending.Sender,
	}
	if err := e.service.HandleQueuedCommand(ctx, inbound, *cmd, emit); err != nil {
		return fmt.Errorf("handle queued command failed: %w", err)
	}
	return nil
}

type twentyQMQPipeline struct {
	streamConsumer *commonmq.StreamConsumer
	streamHandler  *qmq.StreamMessageHandler
}

func newTwentyQMQPipeline(
	cfg *qconfig.Config,
	mqValkey di.MQValkeyClient,
	restClient *llmrest.Client,
	msgProvider *messageprovider.Provider,
	stores *twentyQStores,
	riddleService *qsvc.RiddleService,
	adminServices *twentyQAdminServices,
	logger *slog.Logger,
) *twentyQMQPipeline {
	accessControl := qsecurity.NewAccessControl(cfg.Access)
	replyPublisher := newTwentyQReplyPublisher(cfg, mqValkey, logger)

	commandParser := qmq.NewCommandParser(cfg.Commands.Prefix)
	messageSender := qmq.NewMessageSender(msgProvider, replyPublisher.Publish)
	queueCoordinator := qmq.NewMessageQueueCoordinator(stores.pendingStore, logger)

	chainedQuestionHandler := qmq.NewChainedQuestionHandler(
		riddleService,
		queueCoordinator,
		msgProvider,
		logger,
	)

	commandHandler := qmq.NewGameCommandHandler(
		riddleService,
		adminServices.statsService,
		adminServices.adminHandler,
		adminServices.usageHandler,
		chainedQuestionHandler,
		msgProvider,
		logger,
	)

	queueNotifier := qmq.NewMessageQueueNotifier(msgProvider, cfg.Commands.Prefix, logger)
	executor := &twentyQQueuedExecutor{commandParser: commandParser}

	queueProcessor := qmq.NewMessageQueueProcessor(
		queueCoordinator,
		stores.lockManager,
		stores.processingLockService,
		commandParser,
		msgProvider,
		queueNotifier,
		executor.Execute,
		logger,
	)

	gameMessageService := qmq.NewGameMessageService(
		commandHandler,
		riddleService,
		messageSender,
		msgProvider,
		replyPublisher,
		accessControl,
		commandParser,
		stores.lockManager,
		stores.processingLockService,
		queueProcessor,
		restClient,
		cfg.Commands.Prefix,
		logger,
	)
	executor.service = gameMessageService

	streamHandler := qmq.NewStreamMessageHandler(gameMessageService, logger)
	streamConsumer := newTwentyQStreamConsumer(cfg, mqValkey, logger)
	return &twentyQMQPipeline{
		streamConsumer: streamConsumer,
		streamHandler:  streamHandler,
	}
}

func newTwentyQDataRedis(
	ctx context.Context,
	cfg *qconfig.Config,
	logger *slog.Logger,
) (di.DataValkeyClient, func(), error) {
	client, closeFn, err := bootstrap.NewAndPingDataValkeyClient(ctx, cfg.Redis, logger)
	if err != nil {
		return di.DataValkeyClient{}, nil, fmt.Errorf("init valkey failed: %w", err)
	}
	return client, closeFn, nil
}

func newTwentyQMQValkey(
	ctx context.Context,
	cfg *qconfig.Config,
	logger *slog.Logger,
) (di.MQValkeyClient, func(), error) {
	client, closeFn, err := bootstrap.NewAndPingMQValkeyClient(ctx, cfg.Valkey, logger)
	if err != nil {
		return di.MQValkeyClient{}, nil, fmt.Errorf("init valkey mq failed: %w", err)
	}
	return client, closeFn, nil
}

func newTwentyQRestClient(cfg *qconfig.Config) (*llmrest.Client, error) {
	client, err := llmrest.NewFromConfig(cfg.LlmRest)
	if err != nil {
		return nil, fmt.Errorf("create llm rest client failed: %w", err)
	}
	return client, nil
}

func newTwentyQMessageProvider() (*messageprovider.Provider, error) {
	provider, err := messageprovider.NewFromYAMLAtPath(qassets.GameMessagesYAML, "toon")
	if err != nil {
		return nil, fmt.Errorf("load messages failed: %w", err)
	}
	return provider, nil
}

func newTwentyQDB(
	ctx context.Context,
	cfg *qconfig.Config,
	logger *slog.Logger,
) (*gorm.DB, func(), error) {
	db, sqlDB, err := openPostgres(ctx, cfg.Postgres)
	if err != nil {
		return nil, nil, fmt.Errorf("open postgres failed: %w", err)
	}

	closeFn := func() {
		if closeErr := sqlDB.Close(); closeErr != nil {
			logger.Warn("postgres_close_failed", "err", closeErr)
		}
	}
	return db, closeFn, nil
}

func newTwentyQRepository(ctx context.Context, db *gorm.DB) (*qrepo.Repository, error) {
	repo := qrepo.New(db)
	if err := repo.AutoMigrate(ctx); err != nil {
		return nil, fmt.Errorf("auto migrate failed: %w", err)
	}
	return repo, nil
}

func newTwentyQStatsRecorder(cfg *qconfig.Config, repo *qrepo.Repository, logger *slog.Logger) (*qsvc.StatsRecorder, func()) {
	recorder := qsvc.NewStatsRecorder(repo, logger, cfg.Stats)
	cleanup := func() {
		if recorder != nil {
			recorder.Shutdown()
		}
	}
	return recorder, cleanup
}

func newTwentyQHTTPMux(
	riddleService *qsvc.RiddleService,
	db *gorm.DB,
	msgProvider *messageprovider.Provider,
	logger *slog.Logger,
) *http.ServeMux {
	mux := http.NewServeMux()
	qhttpapi.Register(mux, riddleService, db, msgProvider, logger)
	return mux
}

func newTwentyQHTTPServer(cfg *qconfig.Config, mux *http.ServeMux) *http.Server {
	addr := fmt.Sprintf("%s:%d", cfg.Server.Host, cfg.Server.Port)
	return httpserver.NewServer(addr, mux, httpserver.ServerOptions{
		UseH2C:            true,
		ReadHeaderTimeout: cfg.ServerTuning.ReadHeaderTimeout,
		IdleTimeout:       cfg.ServerTuning.IdleTimeout,
		MaxHeaderBytes:    cfg.ServerTuning.MaxHeaderBytes,
	})
}

func newTwentyQServerApp(
	logger *slog.Logger,
	server *http.Server,
	mqPipeline *twentyQMQPipeline,
) *bootstrap.ServerApp {
	return bootstrap.NewServerApp(
		"twentyq",
		logger,
		server,
		10*time.Second,
		bootstrap.BackgroundTask{
			Name:        "mq_consumer",
			ErrorLogKey: "mq_consumer_failed",
			Run: func(ctx context.Context) error {
				return mqPipeline.streamConsumer.Run(ctx, mqPipeline.streamHandler.HandleStreamMessage)
			},
		},
	)
}

func openPostgres(ctx context.Context, cfg qconfig.PostgresConfig) (*gorm.DB, *sql.DB, error) {
	dsn := fmt.Sprintf(
		"host=%s port=%d user=%s password=%s dbname=%s sslmode=%s",
		cfg.Host,
		cfg.Port,
		cfg.User,
		cfg.Password,
		cfg.Name,
		cfg.SSLMode,
	)

	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		return nil, nil, fmt.Errorf("gorm open failed: %w", err)
	}

	sqlDB, err := db.DB()
	if err != nil {
		return nil, nil, fmt.Errorf("get sql db failed: %w", err)
	}

	pingCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	if err := sqlDB.PingContext(pingCtx); err != nil {
		return nil, nil, fmt.Errorf("db ping failed: %w", err)
	}

	return db, sqlDB, nil
}
