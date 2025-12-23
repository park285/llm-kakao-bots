package bootstrap

import (
	"context"
	"fmt"
	"log/slog"
	"strings"

	commonconfig "github.com/park285/llm-kakao-bots/game-bot-go/internal/common/config"
)

// ConfigLoader 는 타입이다.
type ConfigLoader[C any] func() (*C, error)

// LogConfigGetter 는 타입이다.
type LogConfigGetter[C any] func(*C) commonconfig.LogConfig

// AppInitializer 는 타입이다.
type AppInitializer[C any] func(context.Context, *C, *slog.Logger) (*ServerApp, func(), error)

// RunBotEntrypoint 는 동작을 수행한다.
func RunBotEntrypoint[C any](
	ctx context.Context,
	logger *slog.Logger,
	logFileName string,
	loadConfig ConfigLoader[C],
	getLogConfig LogConfigGetter[C],
	initialize AppInitializer[C],
) (*slog.Logger, error) {
	if err := commonconfig.LoadDotenvIfPresent(); err != nil {
		return logger, fmt.Errorf("load dotenv failed: %w", err)
	}

	cfg, err := loadConfig()
	if err != nil {
		return logger, fmt.Errorf("load config failed: %w", err)
	}

	var logCfg commonconfig.LogConfig
	if getLogConfig != nil {
		logCfg = getLogConfig(cfg)
	}

	if strings.TrimSpace(logCfg.Dir) != "" {
		fileLogger, logErr := EnableFileLogging(logCfg, logFileName)
		if logErr != nil {
			return logger, fmt.Errorf("enable file logging failed: %w", logErr)
		}
		if fileLogger != nil {
			logger = fileLogger
		}
	}

	serverApp, cleanup, err := initialize(ctx, cfg, logger)
	if err != nil {
		return logger, fmt.Errorf("initialize app failed: %w", err)
	}
	if cleanup != nil {
		defer cleanup()
	}

	if err := serverApp.Run(ctx); err != nil {
		return logger, fmt.Errorf("run app failed: %w", err)
	}
	return logger, nil
}
