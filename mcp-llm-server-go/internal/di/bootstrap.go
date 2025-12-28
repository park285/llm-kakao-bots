package di

import (
	"fmt"

	"github.com/park285/llm-kakao-bots/mcp-llm-server-go/internal/config"
	"github.com/park285/llm-kakao-bots/mcp-llm-server-go/internal/domain/turtlesoup"
	"github.com/park285/llm-kakao-bots/mcp-llm-server-go/internal/domain/twentyq"
	"github.com/park285/llm-kakao-bots/mcp-llm-server-go/internal/gemini"
	"github.com/park285/llm-kakao-bots/mcp-llm-server-go/internal/guard"
	"github.com/park285/llm-kakao-bots/mcp-llm-server-go/internal/handler"
	"github.com/park285/llm-kakao-bots/mcp-llm-server-go/internal/metrics"
	"github.com/park285/llm-kakao-bots/mcp-llm-server-go/internal/server"
	"github.com/park285/llm-kakao-bots/mcp-llm-server-go/internal/session"
	"github.com/park285/llm-kakao-bots/mcp-llm-server-go/internal/usage"
)

// InitializeApp 은 애플리케이션 의존성을 초기화하고 App 인스턴스를 반환한다.
func InitializeApp() (*App, error) {
	cfg, err := config.ProvideConfig()
	if err != nil {
		return nil, fmt.Errorf("config: %w", err)
	}

	metricsStore := metrics.NewStore()

	logger, err := ProvideLogger(cfg)
	if err != nil {
		return nil, fmt.Errorf("logger: %w", err)
	}

	usageRepository := usage.NewRepository(cfg, logger)
	usageRecorder := usage.NewRecorder(cfg, usageRepository, logger)

	geminiClient, err := gemini.NewClient(cfg, metricsStore, usageRecorder)
	if err != nil {
		return nil, fmt.Errorf("gemini client: %w", err)
	}

	injectionGuard, err := guard.NewGuard(cfg, logger)
	if err != nil {
		return nil, fmt.Errorf("guard: %w", err)
	}

	llmHandler := handler.NewLLMHandler(cfg, geminiClient, injectionGuard, metricsStore, usageRepository, logger)

	sessionStore, err := session.NewStore(cfg)
	if err != nil {
		return nil, fmt.Errorf("session store: %w", err)
	}

	sessionManager := session.NewManager(sessionStore, geminiClient, cfg, logger)
	sessionHandler := handler.NewSessionHandler(sessionManager, injectionGuard, logger)
	guardHandler := handler.NewGuardHandler(injectionGuard)
	usageHandler := handler.NewUsageHandler(cfg, usageRepository, logger)

	twentyqPrompts, err := twentyq.NewPrompts()
	if err != nil {
		return nil, fmt.Errorf("twentyq prompts: %w", err)
	}

	topicLoader, err := twentyq.NewTopicLoader()
	if err != nil {
		return nil, fmt.Errorf("topic loader: %w", err)
	}

	twentyQHandler := handler.NewTwentyQHandler(cfg, geminiClient, injectionGuard, sessionStore, twentyqPrompts, topicLoader, logger)

	turtlesoupPrompts, err := turtlesoup.NewPrompts()
	if err != nil {
		return nil, fmt.Errorf("turtlesoup prompts: %w", err)
	}

	puzzleLoader, err := turtlesoup.NewPuzzleLoader()
	if err != nil {
		return nil, fmt.Errorf("puzzle loader: %w", err)
	}

	turtleSoupHandler := handler.NewTurtleSoupHandler(cfg, geminiClient, injectionGuard, sessionStore, turtlesoupPrompts, puzzleLoader, logger)

	router := handler.NewRouter(cfg, logger, llmHandler, sessionHandler, guardHandler, usageHandler, twentyQHandler, turtleSoupHandler)
	httpServer := server.NewHTTPServer(cfg, router)

	return NewApp(httpServer, logger, cfg, sessionStore, usageRepository, usageRecorder), nil
}
