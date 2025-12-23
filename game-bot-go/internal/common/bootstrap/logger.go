package bootstrap

import (
	"fmt"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/lmittmann/tint"
	"gopkg.in/natefinch/lumberjack.v2"

	commonconfig "github.com/park285/llm-kakao-bots/game-bot-go/internal/common/config"
)

// NewLogger 는 동작을 수행한다.
func NewLogger() *slog.Logger {
	return slog.New(tint.NewHandler(os.Stdout, &tint.Options{
		Level:      slog.LevelInfo,
		TimeFormat: time.RFC3339,
		AddSource:  true,
	}))
}

// EnableFileLogging 는 동작을 수행한다.
func EnableFileLogging(cfg commonconfig.LogConfig, fileName string) (*slog.Logger, error) {
	logDir := strings.TrimSpace(cfg.Dir)
	if logDir == "" {
		return nil, nil
	}
	if cfg.MaxSizeMB <= 0 || cfg.MaxBackups <= 0 || cfg.MaxAgeDays <= 0 {
		return nil, fmt.Errorf("invalid log config: size=%d backups=%d age_days=%d", cfg.MaxSizeMB, cfg.MaxBackups, cfg.MaxAgeDays)
	}

	if err := os.MkdirAll(logDir, 0755); err != nil {
		return nil, fmt.Errorf("create log dir failed: %w", err)
	}

	logFile := &lumberjack.Logger{
		Filename:   filepath.Join(logDir, fileName),
		MaxSize:    cfg.MaxSizeMB,  // megabytes
		MaxBackups: cfg.MaxBackups, // files
		MaxAge:     cfg.MaxAgeDays, // days
		Compress:   cfg.Compress,
	}

	w := io.MultiWriter(os.Stdout, logFile)
	logger := slog.New(tint.NewHandler(w, &tint.Options{
		Level:      slog.LevelInfo,
		TimeFormat: time.RFC3339,
		AddSource:  true,
		NoColor:    true,
	}))
	slog.SetDefault(logger)
	logger.Info("file_logging_enabled", "path", logFile.Filename)
	return logger, nil
}
