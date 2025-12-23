package command

import (
	"context"
	"fmt"

	"github.com/kapu/hololive-kakao-bot-go/internal/domain"
)

// HelpCommand 는 타입이다.
type HelpCommand struct {
	deps *Dependencies
}

// NewHelpCommand 는 동작을 수행한다.
func NewHelpCommand(deps *Dependencies) *HelpCommand {
	return &HelpCommand{deps: deps}
}

// Name 는 동작을 수행한다.
func (c *HelpCommand) Name() string {
	return "help"
}

// Description 는 동작을 수행한다.
func (c *HelpCommand) Description() string {
	return "도움말을 표시합니다"
}

// Execute 는 동작을 수행한다.
func (c *HelpCommand) Execute(ctx context.Context, cmdCtx *domain.CommandContext, params map[string]any) error {
	if err := c.ensureDeps(); err != nil {
		return err
	}
	message := c.deps.Formatter.FormatHelp()
	return c.deps.SendMessage(ctx, cmdCtx.Room, message)
}

func (c *HelpCommand) ensureDeps() error {
	if c == nil || c.deps == nil {
		return fmt.Errorf("help command dependencies not configured")
	}

	if c.deps.SendMessage == nil {
		return fmt.Errorf("message callback not configured")
	}

	if c.deps.Formatter == nil {
		return fmt.Errorf("formatter not configured")
	}

	return nil
}
