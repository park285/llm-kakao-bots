package command

import (
	"context"
	"fmt"

	"github.com/kapu/hololive-kakao-bot-go/internal/adapter"
	"github.com/kapu/hololive-kakao-bot-go/internal/domain"
)

// UpcomingCommand: 예정된 방송 목록을 조회하는 커맨드 핸들러
type UpcomingCommand struct {
	BaseCommand
}

// NewUpcomingCommand: 예정 방송 조회 커맨드 핸들러를 생성합니다.
func NewUpcomingCommand(deps *Dependencies) *UpcomingCommand {
	return &UpcomingCommand{BaseCommand: NewBaseCommand(deps)}
}

// Name: 커맨드의 이름("upcoming")을 반환합니다.
func (c *UpcomingCommand) Name() string {
	return "upcoming"
}

// Description: 커맨드에 대한 설명을 반환합니다.
func (c *UpcomingCommand) Description() string {
	return "예정된 방송 목록"
}

// Execute: 예정된 방송 목록을 Holodex API로부터 조회하여 출력한다. (멤버 필터링 가능)
func (c *UpcomingCommand) Execute(ctx context.Context, cmdCtx *domain.CommandContext, params map[string]any) error {
	if err := c.ensureDeps(); err != nil {
		return err
	}
	hours := 24 // \uae30\ubcf8\uac12

	if h, ok := params["hours"]; ok {
		switch v := h.(type) {
		case float64:
			hours = int(v)
		case int:
			hours = v
		}
	}

	if hours < 1 {
		hours = 24
	}
	if hours > 168 {
		hours = 168
	}

	memberName, hasMember := params["member"].(string)

	// 멤버 필터링이 지정된 경우
	if hasMember && memberName != "" {
		channel, err := FindActiveMemberOrError(ctx, c.Deps(), cmdCtx.Room, memberName)
		if err != nil {
			return err
		}

		streams, err := c.Deps().Holodex.GetUpcomingStreams(ctx, hours)
		if err != nil {
			return c.Deps().SendError(ctx, cmdCtx.Room, adapter.ErrUpcomingStreamQueryFailed)
		}

		memberStreams := make([]*domain.Stream, 0, len(streams))
		for _, stream := range streams {
			if stream.ChannelID == channel.ID {
				memberStreams = append(memberStreams, stream)
			}
		}

		if len(memberStreams) == 0 {
			return c.Deps().SendMessage(ctx, cmdCtx.Room, fmt.Sprintf(adapter.MsgMemberNoUpcoming, channel.Name, hours))
		}

		message := c.Deps().Formatter.UpcomingStreams(memberStreams, hours)
		return c.Deps().SendMessage(ctx, cmdCtx.Room, message)
	}

	// 전체 예정 방송 조회
	streams, err := c.Deps().Holodex.GetUpcomingStreams(ctx, hours)
	if err != nil {
		return c.Deps().SendError(ctx, cmdCtx.Room, adapter.ErrUpcomingStreamQueryFailed)
	}

	message := c.Deps().Formatter.UpcomingStreams(streams, hours)
	return c.Deps().SendMessage(ctx, cmdCtx.Room, message)
}

func (c *UpcomingCommand) ensureDeps() error {
	if err := c.EnsureBaseDeps(); err != nil {
		return err
	}

	if c.Deps().Holodex == nil || c.Deps().Formatter == nil {
		return fmt.Errorf("upcoming command services not configured")
	}

	return nil
}
