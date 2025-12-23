package main

import (
	"context"
	"log/slog"
	"os"

	"github.com/park285/llm-kakao-bots/game-bot-go/internal/common/bootstrap"
	tsapp "github.com/park285/llm-kakao-bots/game-bot-go/internal/turtlesoup/app"
	tsconfig "github.com/park285/llm-kakao-bots/game-bot-go/internal/turtlesoup/config"
)

func main() {
	logger := bootstrap.NewLogger()
	slog.SetDefault(logger)

	finalLogger, err := bootstrap.RunBotEntrypoint(
		context.Background(),
		logger,
		"turtlesoup.log",
		tsconfig.LoadFromEnv,
		func(cfg *tsconfig.Config) tsconfig.LogConfig { return cfg.Log },
		tsapp.Initialize,
	)
	if err != nil {
		logger = finalLogger
		logger.Error("fatal", "err", err)
		os.Exit(1)
	}
}
