package app

import (
	"context"
	"log/slog"

	"github.com/park285/llm-kakao-bots/game-bot-go/internal/common/bootstrap"
	"github.com/park285/llm-kakao-bots/game-bot-go/internal/turtlesoup/config"
)

// Initialize: TurtleSoup 애플리케이션 의존성을 초기화하고 ServerApp을 반환합니다.
func Initialize(ctx context.Context, cfg *config.Config, logger *slog.Logger) (*bootstrap.ServerApp, func(), error) {
	restClient, err := newTurtleSoupRestClient(cfg)
	if err != nil {
		return nil, nil, err
	}

	msgProvider, err := newTurtleSoupMessageProvider(cfg)
	if err != nil {
		return nil, nil, err
	}

	mqValkeyClient, cleanupMQValkey, err := newTurtleSoupMQValkey(ctx, cfg, logger)
	if err != nil {
		return nil, nil, err
	}

	replyPublisher := newTurtleSoupReplyPublisher(cfg, mqValkeyClient, logger)
	injectionGuard := newTurtleSoupInjectionGuard(cfg, restClient, logger)

	dataValkeyClient, cleanupDataValkey, err := newTurtleSoupDataRedis(ctx, cfg, logger)
	if err != nil {
		cleanupMQValkey()
		return nil, nil, err
	}

	stores := newTurtleSoupStores(dataValkeyClient, logger)
	services := newTurtleSoupServices(cfg, restClient, msgProvider, replyPublisher, injectionGuard, stores, logger)
	gameService := newTurtleSoupGameService(services)

	httpMux := newTurtleSoupHTTPMux(cfg, restClient, gameService, logger)
	httpServer := newTurtleSoupHTTPServer(cfg, httpMux)

	streamConsumer := newTurtleSoupStreamConsumer(cfg, mqValkeyClient, logger)
	mqPipeline := newTurtleSoupMQPipeline(restClient, msgProvider, stores, services, streamConsumer, logger)

	serverApp := newTurtleSoupServerApp(logger, httpServer, mqPipeline)

	cleanup := func() {
		cleanupDataValkey()
		cleanupMQValkey()
	}

	return serverApp, cleanup, nil
}
