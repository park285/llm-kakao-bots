package main

import (
	"context"
	"log/slog"
	"os"

	"github.com/park285/llm-kakao-bots/game-bot-go/internal/common/bootstrap"
	qapp "github.com/park285/llm-kakao-bots/game-bot-go/internal/twentyq/app"
	qconfig "github.com/park285/llm-kakao-bots/game-bot-go/internal/twentyq/config"
)

func main() {
	logger := bootstrap.NewLogger()
	slog.SetDefault(logger)

	finalLogger, err := bootstrap.RunBotEntrypoint(
		context.Background(),
		logger,
		"twentyq.log",
		qconfig.LoadFromEnv,
		func(cfg *qconfig.Config) qconfig.LogConfig { return cfg.Log },
		qapp.Initialize,
	)
	if err != nil {
		logger = finalLogger
		logger.Error("fatal", "err", err)
		os.Exit(1)
	}
}
