package di

import (
	"fmt"
	"log/slog"

	"github.com/park285/llm-kakao-bots/mcp-llm-server-go/internal/config"
	"github.com/park285/llm-kakao-bots/mcp-llm-server-go/internal/logging"
)

// ProvideLogger: 로거를 구성해 반환합니다.
func ProvideLogger(cfg *config.Config) (*slog.Logger, error) {
	logger, err := logging.NewLogger(cfg.Logging)
	if err != nil {
		return nil, fmt.Errorf("init logger: %w", err)
	}
	return logger, nil
}
