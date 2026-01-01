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

// InjectionGuard: 사용자 입력의 악성 여부를 검사하는 인터페이스입니다.
type InjectionGuard interface {
	IsMalicious(ctx context.Context, input string) (bool, error)
	ValidateOrThrow(ctx context.Context, input string) (string, error)
}

// McpInjectionGuard: MCP LLM 서버를 통해 Injection 검사를 수행하는 구현체입니다.
type McpInjectionGuard struct {
	restClient *llmrest.Client
	logger     *slog.Logger
}

// NewMcpInjectionGuard: McpInjectionGuard 인스턴스를 생성합니다.
func NewMcpInjectionGuard(restClient *llmrest.Client, logger *slog.Logger) *McpInjectionGuard {
	if logger == nil {
		logger = slog.Default()
	}
	return &McpInjectionGuard{
		restClient: restClient,
		logger:     logger,
	}
}

// IsMalicious: 입력이 악성인지 검사합니다.
func (g *McpInjectionGuard) IsMalicious(ctx context.Context, input string) (bool, error) {
	malicious, err := g.restClient.GuardIsMalicious(ctx, input)
	if err != nil {
		return false, fmt.Errorf("guard isMalicious failed: %w", err)
	}
	return malicious, nil
}

// ValidateOrThrow: 입력을 검증하고 악성이면 에러를 반환합니다.
// 정상적인 입력은 정규화하여 반환합니다.
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
