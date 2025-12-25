package command

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/kapu/hololive-kakao-bot-go/internal/adapter"
	"github.com/kapu/hololive-kakao-bot-go/internal/domain"
	"github.com/kapu/hololive-kakao-bot-go/internal/util"
)

// StatsCommand 는 타입이다.
type StatsCommand struct {
	deps *Dependencies
}

// NewStatsCommand 는 동작을 수행한다.
func NewStatsCommand(deps *Dependencies) *StatsCommand {
	return &StatsCommand{deps: deps}
}

// Name 는 동작을 수행한다.
func (c *StatsCommand) Name() string {
	return "stats"
}

// Description 는 동작을 수행한다.
func (c *StatsCommand) Description() string {
	return "구독자 순위 및 통계 조회"
}

// Execute 는 동작을 수행한다.
func (c *StatsCommand) Execute(ctx context.Context, cmdCtx *domain.CommandContext, params map[string]any) error {
	if err := c.ensureDeps(cmdCtx); err != nil {
		return err
	}

	action, _ := params["action"].(string)
	if action == "" {
		action = "gainers"
	}

	switch util.Normalize(action) {
	case "gainers", "구독자순위":
		return c.showTopGainers(ctx, cmdCtx, params)
	default:
		return c.deps.SendError(ctx, cmdCtx.Room, adapter.ErrUnknownStatsPeriod)
	}
}

func (c *StatsCommand) showTopGainers(ctx context.Context, cmdCtx *domain.CommandContext, params map[string]any) error {
	periodStr, _ := params["period"].(string)
	now := time.Now()
	since, periodLabel := domain.ResolveStatsPeriod(now, periodStr)

	gainers, err := c.deps.StatsRepo.GetTopGainers(ctx, since, 10)
	if err != nil {
		c.deps.Logger.Error("Failed to get top gainers", slog.Any("error", err))
		return c.deps.SendError(ctx, cmdCtx.Room, adapter.ErrStatsQueryFailed)
	}

	if len(gainers) == 0 {
		return c.deps.SendMessage(ctx, cmdCtx.Room, adapter.MsgNoStatsData)
	}

	trimmedPeriod := util.TrimSpace(periodLabel)
	instruction := fmt.Sprintf("%s %s", adapter.DefaultEmoji.Stats, adapter.MsgStatsGainersHeader)
	if trimmedPeriod != "" {
		instruction = fmt.Sprintf("%s (%s)", instruction, trimmedPeriod)
	}

	var builder strings.Builder
	builder.WriteString(instruction)
	builder.WriteString("\n\n")

	for _, entry := range gainers {
		builder.WriteString(fmt.Sprintf("%d위. %s\n", entry.Rank, entry.MemberName))
		builder.WriteString(fmt.Sprintf("    +%s명", util.FormatKoreanNumber(entry.Value)))
		if entry.CurrentSubscribers > 0 {
			builder.WriteString(fmt.Sprintf(" (현재 %s명)", util.FormatKoreanNumber(int64(entry.CurrentSubscribers))))
		}
		builder.WriteString("\n\n")
	}

	content := util.TrimSpace(builder.String())
	message := util.ApplyKakaoSeeMorePadding(util.StripLeadingHeader(content, instruction), instruction)

	return c.deps.SendMessage(ctx, cmdCtx.Room, message)
}

func (c *StatsCommand) ensureDeps(cmdCtx *domain.CommandContext) error {
	if c == nil || c.deps == nil {
		return fmt.Errorf("stats command dependencies not configured")
	}

	if c.deps.SendMessage == nil || c.deps.SendError == nil {
		return fmt.Errorf("message callbacks not configured")
	}

	if c.deps.StatsRepo == nil {
		return fmt.Errorf("stats repository not configured")
	}

	if c.deps.Logger == nil {
		c.deps.Logger = slog.Default()
	}

	return nil
}
