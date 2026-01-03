package di

import (
	"fmt"
	"log/slog"

	"github.com/park285/llm-kakao-bots/mcp-llm-server-go/internal/config"
	"github.com/park285/llm-kakao-bots/mcp-llm-server-go/internal/logging"
)

// ProvideLogger: 로거를 구성해 반환합니다.
// OTel이 활성화된 경우 로그에 trace_id/span_id가 자동으로 추가됩니다.
func ProvideLogger(cfg *config.Config) (*slog.Logger, error) {
	logger, err := logging.NewLoggerWithOTel(cfg.Logging, cfg.Telemetry.Enabled)
	if err != nil {
		return nil, fmt.Errorf("init logger: %w", err)
	}
	return logger, nil
}
