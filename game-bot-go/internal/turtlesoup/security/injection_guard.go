package security

import (
	"context"
	"fmt"
	"log/slog"
	"strings"

	cerrors "github.com/park285/llm-kakao-bots/game-bot-go/internal/common/errors"
	"github.com/park285/llm-kakao-bots/game-bot-go/internal/common/llmrest"
)

const logTextLimit = 100

// InjectionGuard 는 타입이다.
type InjectionGuard interface {
	IsMalicious(ctx context.Context, input string) (bool, error)
	ValidateOrThrow(ctx context.Context, input string) (string, error)
}

// McpInjectionGuard 는 타입이다.
type McpInjectionGuard struct {
	restClient *llmrest.Client
	logger     *slog.Logger
}

// NewMcpInjectionGuard 는 동작을 수행한다.
func NewMcpInjectionGuard(restClient *llmrest.Client, logger *slog.Logger) *McpInjectionGuard {
	if logger == nil {
		logger = slog.Default()
	}
	return &McpInjectionGuard{
		restClient: restClient,
		logger:     logger,
	}
}

// IsMalicious 는 동작을 수행한다.
func (g *McpInjectionGuard) IsMalicious(ctx context.Context, input string) (bool, error) {
	malicious, err := g.restClient.GuardIsMalicious(ctx, input)
	if err != nil {
		return false, fmt.Errorf("guard isMalicious failed: %w", err)
	}
	return malicious, nil
}

// ValidateOrThrow 는 동작을 수행한다.
func (g *McpInjectionGuard) ValidateOrThrow(ctx context.Context, input string) (string, error) {
	if strings.TrimSpace(input) == "" {
		return "", cerrors.MalformedInputError{Message: "empty input"}
	}

	malicious, err := g.IsMalicious(ctx, input)
	if err != nil {
		return "", err
	}
	if malicious {
		g.logger.Warn("injection_blocked", "input", truncateForLog(input))
		return "", cerrors.InputInjectionError{Message: "potentially malicious input detected"}
	}

	return sanitize(input), nil
}

func sanitize(input string) string {
	return strings.Join(strings.Fields(strings.TrimSpace(input)), " ")
}

func truncateForLog(input string) string {
	text := strings.TrimSpace(input)
	if len(text) <= logTextLimit {
		return text
	}
	return text[:logTextLimit]
}
