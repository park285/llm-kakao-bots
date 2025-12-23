//go:build wireinject

package di

import (
	"github.com/google/wire"

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

func InitializeApp() (*App, error) {
	wire.Build(
		config.ProvideConfig,
		ProvideLogger,
		metrics.NewStore,
		usage.NewRepository,
		usage.NewRecorder,
		guard.NewGuard,
		gemini.NewClient,
		wire.Bind(new(handler.LLMClient), new(*gemini.Client)),
		session.NewStore,
		session.NewManager,
		twentyq.NewPrompts,
		turtlesoup.NewPrompts,
		turtlesoup.NewPuzzleLoader,
		handler.NewLLMHandler,
		handler.NewSessionHandler,
		handler.NewGuardHandler,
		handler.NewUsageHandler,
		handler.NewTwentyQHandler,
		handler.NewTurtleSoupHandler,
		handler.NewRouter,
		server.NewHTTPServer,
		NewApp,
	)
	return nil, nil
}
