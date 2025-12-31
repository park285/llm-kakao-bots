package app

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/park285/llm-kakao-bots/game-bot-go/internal/common/bootstrap"
	"github.com/park285/llm-kakao-bots/game-bot-go/internal/common/di"
	"github.com/park285/llm-kakao-bots/game-bot-go/internal/common/httpserver"
	"github.com/park285/llm-kakao-bots/game-bot-go/internal/common/llmrest"
	"github.com/park285/llm-kakao-bots/game-bot-go/internal/common/messageprovider"
	commonmq "github.com/park285/llm-kakao-bots/game-bot-go/internal/common/mq"
	"github.com/park285/llm-kakao-bots/game-bot-go/internal/common/mqmsg"
	tsassets "github.com/park285/llm-kakao-bots/game-bot-go/internal/turtlesoup/assets"
	tsconfig "github.com/park285/llm-kakao-bots/game-bot-go/internal/turtlesoup/config"
	"github.com/park285/llm-kakao-bots/game-bot-go/internal/turtlesoup/httpapi"
	tsmodel "github.com/park285/llm-kakao-bots/game-bot-go/internal/turtlesoup/model"
	tsmq "github.com/park285/llm-kakao-bots/game-bot-go/internal/turtlesoup/mq"
	tsredis "github.com/park285/llm-kakao-bots/game-bot-go/internal/turtlesoup/redis"
	tssecurity "github.com/park285/llm-kakao-bots/game-bot-go/internal/turtlesoup/security"
	tssvc "github.com/park285/llm-kakao-bots/game-bot-go/internal/turtlesoup/service"
)

type turtleSoupStores struct {
	lockManager           *tsredis.LockManager
	sessionStore          *tsredis.SessionStore
	sessionManager        *tssvc.GameSessionManager
	processingLockService *tsredis.ProcessingLockService
	pendingStore          *tsredis.PendingMessageStore
	dedupStore            *tsredis.PuzzleDedupStore
	voteStore             *tsredis.SurrenderVoteStore
}

func newTurtleSoupStores(client di.DataValkeyClient, logger *slog.Logger) *turtleSoupStores {
	lockManager := tsredis.NewLockManager(client.Client, logger)
	sessionStore := tsredis.NewSessionStore(client.Client, logger)
	return &turtleSoupStores{
		lockManager:           lockManager,
		sessionStore:          sessionStore,
		sessionManager:        tssvc.NewGameSessionManager(sessionStore, lockManager),
		processingLockService: tsredis.NewProcessingLockService(client.Client, logger),
		pendingStore:          tsredis.NewPendingMessageStore(client.Client, logger),
		dedupStore:            tsredis.NewPuzzleDedupStore(client.Client, logger),
		voteStore:             tsredis.NewSurrenderVoteStore(client.Client, logger),
	}
}

type turtleSoupServices struct {
	gameService    *tssvc.GameService
	voteService    *tssvc.SurrenderVoteService
	accessControl  *tssecurity.AccessControl
	commandHandler *tsmq.GameCommandHandler
	commandParser  *tsmq.CommandParser
	messageSender  *tsmq.MessageSender
	replyPublisher *tsmq.ReplyPublisher
}

func newTurtleSoupServices(
	cfg *tsconfig.Config,
	restClient *llmrest.Client,
	msgProvider *messageprovider.Provider,
	replyPublisher *tsmq.ReplyPublisher,
	injectionGuard tssecurity.InjectionGuard,
	stores *turtleSoupStores,
	logger *slog.Logger,
) *turtleSoupServices {
	puzzleService := tssvc.NewPuzzleService(restClient, cfg.Puzzle, stores.dedupStore, logger)
	setupService := tssvc.NewGameSetupService(restClient, puzzleService, stores.sessionManager, logger)
	gameService := tssvc.NewGameService(restClient, stores.sessionManager, setupService, injectionGuard, logger)
	voteService := tssvc.NewSurrenderVoteService(stores.sessionManager, stores.voteStore)
	accessControl := tssecurity.NewAccessControl(cfg.Access)

	messageBuilder := tsmq.NewMessageBuilder(msgProvider)
	surrenderHandler := tsmq.NewSurrenderHandler(gameService, voteService, msgProvider)
	commandHandler := tsmq.NewGameCommandHandler(gameService, surrenderHandler, msgProvider, messageBuilder, logger)
	commandParser := tsmq.NewCommandParser(cfg.Commands.Prefix)
	messageSender := tsmq.NewMessageSender(msgProvider, replyPublisher.Publish)

	return &turtleSoupServices{
		gameService:    gameService,
		voteService:    voteService,
		accessControl:  accessControl,
		commandHandler: commandHandler,
		commandParser:  commandParser,
		messageSender:  messageSender,
		replyPublisher: replyPublisher,
	}
}

func newTurtleSoupGameService(services *turtleSoupServices) *tssvc.GameService {
	if services == nil {
		return nil
	}
	return services.gameService
}

func newTurtleSoupReplyPublisher(cfg *tsconfig.Config, mqValkey di.MQValkeyClient, logger *slog.Logger) *tsmq.ReplyPublisher {
	return commonmq.NewBotReplyPublisher(
		mqValkey.Client,
		logger,
		cfg.Valkey.ReplyStreamKey,
		cfg.Valkey.StreamMaxLen,
	)
}

func newTurtleSoupStreamConsumer(cfg *tsconfig.Config, mqValkey di.MQValkeyClient, logger *slog.Logger) *commonmq.StreamConsumer {
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

type turtleSoupQueuedExecutor struct {
	service       *tsmq.GameMessageService
	commandParser *tsmq.CommandParser
}

func (e *turtleSoupQueuedExecutor) Execute(
	ctx context.Context,
	chatID string,
	pending tsmodel.PendingMessage,
	emit func(mqmsg.OutboundMessage) error,
) error {
	if e.service == nil {
		return errors.New("game message service not initialized")
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

type turtleSoupMQPipeline struct {
	streamConsumer *commonmq.StreamConsumer
	streamHandler  *tsmq.StreamMessageHandler
}

func newTurtleSoupMQPipeline(
	restClient *llmrest.Client,
	msgProvider *messageprovider.Provider,
	stores *turtleSoupStores,
	services *turtleSoupServices,
	streamConsumer *commonmq.StreamConsumer,
	logger *slog.Logger,
) *turtleSoupMQPipeline {
	queueCoordinator := tsmq.NewMessageQueueCoordinator(stores.pendingStore, logger)
	queueNotifier := tsmq.NewMessageQueueNotifier(msgProvider, logger)
	executor := &turtleSoupQueuedExecutor{commandParser: services.commandParser}

	queueProcessor := tsmq.NewMessageQueueProcessor(
		queueCoordinator,
		stores.lockManager,
		stores.processingLockService,
		msgProvider,
		queueNotifier,
		executor.Execute,
		logger,
	)

	gameMessageService := tsmq.NewGameMessageService(
		services.commandHandler,
		services.messageSender,
		msgProvider,
		services.replyPublisher,
		services.accessControl,
		services.commandParser,
		stores.processingLockService,
		queueProcessor,
		restClient,
		logger,
	)
	executor.service = gameMessageService

	streamHandler := tsmq.NewStreamMessageHandler(gameMessageService, logger)
	return &turtleSoupMQPipeline{
		streamConsumer: streamConsumer,
		streamHandler:  streamHandler,
	}
}

func newTurtleSoupDataRedis(
	ctx context.Context,
	cfg *tsconfig.Config,
	logger *slog.Logger,
) (di.DataValkeyClient, func(), error) {
	client, closeFn, err := bootstrap.NewAndPingDataValkeyClient(ctx, cfg.Redis, logger)
	if err != nil {
		return di.DataValkeyClient{}, nil, fmt.Errorf("init valkey failed: %w", err)
	}
	return client, closeFn, nil
}

func newTurtleSoupMQValkey(
	ctx context.Context,
	cfg *tsconfig.Config,
	logger *slog.Logger,
) (di.MQValkeyClient, func(), error) {
	client, closeFn, err := bootstrap.NewAndPingMQValkeyClient(ctx, cfg.Valkey, logger)
	if err != nil {
		return di.MQValkeyClient{}, nil, fmt.Errorf("init valkey mq failed: %w", err)
	}
	return client, closeFn, nil
}

func newTurtleSoupRestClient(cfg *tsconfig.Config) (*llmrest.Client, error) {
	client, err := llmrest.NewFromConfig(cfg.Llm)
	if err != nil {
		return nil, fmt.Errorf("create llm rest client failed: %w", err)
	}
	return client, nil
}

func newTurtleSoupInjectionGuard(cfg *tsconfig.Config, restClient *llmrest.Client, logger *slog.Logger) tssecurity.InjectionGuard {
	base := tssecurity.NewMcpInjectionGuard(restClient, logger)
	return tssecurity.NewCachedInjectionGuard(
		base,
		cfg.InjectionGuard.CacheTTL,
		cfg.InjectionGuard.CacheMaxEntries,
		logger,
	)
}

func newTurtleSoupMessageProvider(cfg *tsconfig.Config) (*messageprovider.Provider, error) {
	msgYAML := strings.ReplaceAll(tsassets.GameMessagesYAML, "/스프", cfg.Commands.Prefix)
	provider, err := messageprovider.NewFromYAML(msgYAML)
	if err != nil {
		return nil, fmt.Errorf("load messages failed: %w", err)
	}
	return provider, nil
}

func newTurtleSoupHTTPMux(
	cfg *tsconfig.Config,
	restClient *llmrest.Client,
	gameService *tssvc.GameService,
	logger *slog.Logger,
) *http.ServeMux {
	mux := http.NewServeMux()
	httpapi.Register(mux, cfg.Llm, restClient, gameService, logger)
	return mux
}

func newTurtleSoupHTTPServer(cfg *tsconfig.Config, mux *http.ServeMux) *http.Server {
	addr := fmt.Sprintf("%s:%d", cfg.Server.Host, cfg.Server.Port)
	return httpserver.NewServer(addr, mux, httpserver.ServerOptions{
		UseH2C:            true,
		ReadHeaderTimeout: cfg.ServerTuning.ReadHeaderTimeout,
		IdleTimeout:       cfg.ServerTuning.IdleTimeout,
		MaxHeaderBytes:    cfg.ServerTuning.MaxHeaderBytes,
	})
}

func newTurtleSoupServerApp(
	logger *slog.Logger,
	server *http.Server,
	mqPipeline *turtleSoupMQPipeline,
) *bootstrap.ServerApp {
	return bootstrap.NewServerApp(
		"turtlesoup",
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
