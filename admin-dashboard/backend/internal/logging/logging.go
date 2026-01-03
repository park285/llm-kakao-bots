// Package logging: 통합 로깅 기능을 제공합니다.
// tint 핸들러, lumberjack 로테이션, OTel 상관관계를 지원합니다.
package logging

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/lmittmann/tint"
	"go.opentelemetry.io/otel/trace"
	"gopkg.in/natefinch/lumberjack.v2"
)

// Config: 로깅 설정입니다.
type Config struct {
	Level      string // debug, info, warn, error
	Dir        string // 로그 디렉토리 (비어있으면 stdout만)
	MaxSizeMB  int    // 파일당 최대 크기 (MB)
	MaxBackups int    // 유지할 백업 파일 수
	MaxAgeDays int    // 보관 일수
	Compress   bool   // gzip 압축 여부
}

// DefaultConfig: 기본 로깅 설정을 반환합니다.
func DefaultConfig() Config {
	return Config{
		Level:      "info",
		Dir:        "/app/logs",
		MaxSizeMB:  50,
		MaxBackups: 3,
		MaxAgeDays: 7,
		Compress:   true,
	}
}

const (
	defaultLogFileName  = "admin.log"
	combinedLogFileName = "combined.log"
)

// NewLogger: 로거를 생성합니다.
func NewLogger(cfg Config) (*slog.Logger, error) {
	return NewLoggerWithOTel(cfg, false)
}

// NewLoggerWithOTel: OTel 상관관계 기능을 포함한 로거를 생성합니다.
// enableOTel이 true면 로그에 trace_id/span_id가 자동으로 추가됩니다.
func NewLoggerWithOTel(cfg Config, enableOTel bool) (*slog.Logger, error) {
	level := parseLevel(cfg.Level)
	logDir := strings.TrimSpace(cfg.Dir)
	if logDir == "" {
		logger := newLogger(os.Stdout, level, false, enableOTel)
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

	// 서비스별 로그 파일
	logFile := &lumberjack.Logger{
		Filename:   filepath.Join(logDir, defaultLogFileName),
		MaxSize:    cfg.MaxSizeMB,
		MaxBackups: cfg.MaxBackups,
		MaxAge:     cfg.MaxAgeDays,
		Compress:   cfg.Compress,
	}

	// 통합 로그 파일 (combined.log) - 모든 서비스의 로그가 여기에 모임
	combinedLogFile := &lumberjack.Logger{
		Filename:   filepath.Join(logDir, combinedLogFileName),
		MaxSize:    cfg.MaxSizeMB * 3, // 서비스 합산이므로 더 큰 용량
		MaxBackups: cfg.MaxBackups,
		MaxAge:     cfg.MaxAgeDays,
		Compress:   cfg.Compress,
	}

	// stdout + 서비스별 로그 + 통합 로그에 동시 출력
	writer := io.MultiWriter(os.Stdout, logFile, combinedLogFile)
	logger := newLogger(writer, level, true, enableOTel)
	slog.SetDefault(logger)
	logger.Info("file_logging_enabled",
		slog.String("path", logFile.Filename),
		slog.String("combined", combinedLogFile.Filename),
		slog.Bool("otel_correlation", enableOTel),
	)
	return logger, nil
}

func newLogger(writer io.Writer, level slog.Level, noColor bool, enableOTel bool) *slog.Logger {
	var handler slog.Handler
	handler = tint.NewHandler(writer, &tint.Options{
		Level:      level,
		TimeFormat: time.RFC3339,
		AddSource:  true,
		NoColor:    noColor,
	})

	// OTel 활성화 시 trace_id/span_id 자동 추가
	if enableOTel {
		handler = &OTelHandler{inner: handler}
	}

	return slog.New(handler)
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

// OTelHandler: slog.Handler를 래핑하여 trace_id/span_id를 자동으로 로그에 추가합니다.
type OTelHandler struct {
	inner slog.Handler
}

func (h *OTelHandler) Enabled(ctx context.Context, level slog.Level) bool {
	return h.inner.Enabled(ctx, level)
}

func (h *OTelHandler) Handle(ctx context.Context, record slog.Record) error {
	span := trace.SpanFromContext(ctx)
	if span.SpanContext().IsValid() {
		spanCtx := span.SpanContext()
		record.AddAttrs(
			slog.String("trace_id", spanCtx.TraceID().String()),
			slog.String("span_id", spanCtx.SpanID().String()),
		)
	}
	if err := h.inner.Handle(ctx, record); err != nil {
		return fmt.Errorf("handle log record: %w", err)
	}
	return nil
}

func (h *OTelHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	return &OTelHandler{inner: h.inner.WithAttrs(attrs)}
}

func (h *OTelHandler) WithGroup(name string) slog.Handler {
	return &OTelHandler{inner: h.inner.WithGroup(name)}
}
