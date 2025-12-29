package logging

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

	"github.com/park285/llm-kakao-bots/mcp-llm-server-go/internal/config"
)

const (
	defaultLogFileName = "server.log"
)

// NewLogger: 로거를 생성합니다.
func NewLogger(cfg config.LoggingConfig) (*slog.Logger, error) {
	level := parseLevel(cfg.Level)
	logDir := strings.TrimSpace(cfg.LogDir)
	if logDir == "" {
		logger := newLogger(os.Stdout, level, false)
		slog.SetDefault(logger)
		return logger, nil
	}

	if cfg.MaxSizeMB <= 0 || cfg.MaxBackups <= 0 || cfg.MaxAgeDays <= 0 {
		return nil, fmt.Errorf(
			"invalid log config: size=%d backups=%d age_days=%d",
			cfg.MaxSizeMB,
			cfg.MaxBackups,
			cfg.MaxAgeDays,
		)
	}

	if err := os.MkdirAll(logDir, 0755); err != nil {
		return nil, fmt.Errorf("create log dir failed: %w", err)
	}

	logFile := &lumberjack.Logger{
		Filename:   filepath.Join(logDir, defaultLogFileName),
		MaxSize:    cfg.MaxSizeMB,
		MaxBackups: cfg.MaxBackups,
		MaxAge:     cfg.MaxAgeDays,
		Compress:   cfg.Compress,
	}

	writer := io.MultiWriter(os.Stdout, logFile)
	logger := newLogger(writer, level, true)
	slog.SetDefault(logger)
	logger.Info("file_logging_enabled", "path", logFile.Filename)
	return logger, nil
}

func newLogger(writer io.Writer, level slog.Level, noColor bool) *slog.Logger {
	return slog.New(tint.NewHandler(writer, &tint.Options{
		Level:      level,
		TimeFormat: time.RFC3339,
		AddSource:  true,
		NoColor:    noColor,
	}))
}

func parseLevel(level string) slog.Level {
	switch strings.ToLower(strings.TrimSpace(level)) {
	case "debug":
		return slog.LevelDebug
	case "warn", "warning":
		return slog.LevelWarn
	case "error":
		return slog.LevelError
	default:
		return slog.LevelInfo
	}
}
