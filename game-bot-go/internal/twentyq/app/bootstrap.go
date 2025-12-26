package app

import (
	"context"
	"log/slog"

	"github.com/park285/llm-kakao-bots/game-bot-go/internal/common/bootstrap"
	"github.com/park285/llm-kakao-bots/game-bot-go/internal/twentyq/config"
)

// Initialize 는 TwentyQ 애플리케이션 의존성을 초기화하고 ServerApp을 반환한다.
func Initialize(ctx context.Context, cfg *config.Config, logger *slog.Logger) (*bootstrap.ServerApp, func(), error) {
	restClient, err := newTwentyQRestClient(cfg)
	if err != nil {
		return nil, nil, err
	}

	msgProvider, err := newTwentyQMessageProvider()
	if err != nil {
		return nil, nil, err
	}

	dataValkeyClient, cleanupDataValkey, err := newTwentyQDataRedis(ctx, cfg, logger)
	if err != nil {
		return nil, nil, err
	}

	stores := newTwentyQStores(dataValkeyClient, logger)

	db, cleanupDB, err := newTwentyQDB(ctx, cfg, logger)
	if err != nil {
		cleanupDataValkey()
		return nil, nil, err
	}

	repository, err := newTwentyQRepository(ctx, db)
	if err != nil {
		cleanupDB()
		cleanupDataValkey()
		return nil, nil, err
	}

	statsRecorder, cleanupStats := newTwentyQStatsRecorder(cfg, repository, logger)

	riddleService := newTwentyQRiddleService(cfg, restClient, msgProvider, stores, statsRecorder, logger)

	httpMux := newTwentyQHTTPMux(riddleService, db, msgProvider, logger)
	httpServer := newTwentyQHTTPServer(cfg, httpMux)

	mqValkeyClient, cleanupMQValkey, err := newTwentyQMQValkey(ctx, cfg, logger)
	if err != nil {
		cleanupStats()
		cleanupDB()
		cleanupDataValkey()
		return nil, nil, err
	}

	adminServices := newTwentyQAdminServices(cfg, db, restClient, msgProvider, stores, riddleService, logger)
	mqPipeline := newTwentyQMQPipeline(cfg, mqValkeyClient, restClient, msgProvider, stores, riddleService, adminServices, logger)

	serverApp := newTwentyQServerApp(logger, httpServer, mqPipeline)

	cleanup := func() {
		cleanupMQValkey()
		cleanupStats()
		cleanupDB()
		cleanupDataValkey()
	}

	return serverApp, cleanup, nil
}
