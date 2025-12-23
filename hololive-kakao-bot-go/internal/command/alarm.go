package command

import (
	"context"
	"fmt"

	"go.uber.org/zap"

	"github.com/kapu/hololive-kakao-bot-go/internal/adapter"
	"github.com/kapu/hololive-kakao-bot-go/internal/domain"
)

// AlarmCommand 는 타입이다.
type AlarmCommand struct {
	BaseCommand
}

// NewAlarmCommand 는 동작을 수행한다.
func NewAlarmCommand(deps *Dependencies) *AlarmCommand {
	return &AlarmCommand{BaseCommand: NewBaseCommand(deps)}
}

// Name 는 동작을 수행한다.
func (c *AlarmCommand) Name() string {
	return "alarm"
}

// Description 는 동작을 수행한다.
func (c *AlarmCommand) Description() string {
	return "방송 알람 관리"
}

// Execute 는 동작을 수행한다.
func (c *AlarmCommand) Execute(ctx context.Context, cmdCtx *domain.CommandContext, params map[string]any) error {
	if err := c.ensureDeps(); err != nil {
		return err
	}

	if c.Deps().Alarm == nil {
		return c.Deps().SendError(ctx, cmdCtx.Room, adapter.ErrAlarmServiceNotInitialized)
	}

	action, hasAction := params["action"].(string)
	if !hasAction {
		action = "list"
	}

	switch action {
	case "set", "add":
		return c.handleAdd(ctx, cmdCtx, params)
	case "remove", "delete":
		return c.handleRemove(ctx, cmdCtx, params)
	case "list":
		c.Deps().Logger.Info("Alarm list requested")
		return c.handleList(ctx, cmdCtx)
	case "clear":
		return c.handleClear(ctx, cmdCtx)
	case "invalid":
		subCmd, _ := params["sub_command"].(string)
		memberName, _ := params["member"].(string)
		c.Deps().Logger.Info("Invalid alarm command received",
			zap.String("room", cmdCtx.Room),
			zap.String("sender", cmdCtx.Sender),
			zap.String("sub_command", subCmd),
			zap.String("member", memberName),
		)
		return c.Deps().SendError(ctx, cmdCtx.Room, c.Deps().Formatter.InvalidAlarmUsage())
	default:
		return c.Deps().SendMessage(ctx, cmdCtx.Room, c.Deps().Formatter.FormatHelp())
	}
}

func (c *AlarmCommand) ensureDeps() error {
	if err := c.EnsureBaseDeps(); err != nil {
		return err
	}

	if c.Deps().Matcher == nil || c.Deps().Formatter == nil {
		return fmt.Errorf("alarm command services not configured")
	}

	return nil
}

func (c *AlarmCommand) handleAdd(ctx context.Context, cmdCtx *domain.CommandContext, params map[string]any) error {
	memberName, hasMember := params["member"].(string)
	if !hasMember || memberName == "" {
		return c.Deps().SendError(ctx, cmdCtx.Room, adapter.ErrAlarmNeedMemberNameAdd)
	}

	c.Deps().Logger.Info("Alarm add requested", zap.String("member", memberName))

	channel, err := FindActiveMemberOrError(ctx, c.Deps(), cmdCtx.Room, memberName)
	if err != nil {
		return err
	}

	added, err := c.Deps().Alarm.AddAlarm(
		ctx,
		cmdCtx.Room,   // roomID (숫자)
		cmdCtx.UserID, // userID (숫자)
		channel.ID,
		channel.Name,
		cmdCtx.RoomName, // roomName (한글)
		cmdCtx.UserName, // userName (한글)
	)
	if err != nil {
		c.Deps().Logger.Error("Failed to add alarm",
			zap.String("channel", channel.Name),
			zap.Error(err),
		)
		return c.Deps().SendError(ctx, cmdCtx.Room, adapter.ErrAlarmAddFailed)
	}

	nextStreamInfo, _ := c.Deps().Alarm.GetNextStreamInfo(ctx, channel.ID)

	message := c.Deps().Formatter.FormatAlarmAdded(channel.Name, added, nextStreamInfo)
	return c.Deps().SendMessage(ctx, cmdCtx.Room, message)
}

func (c *AlarmCommand) handleRemove(ctx context.Context, cmdCtx *domain.CommandContext, params map[string]any) error {
	memberName, hasMember := params["member"].(string)
	if !hasMember || memberName == "" {
		return c.Deps().SendError(ctx, cmdCtx.Room, adapter.ErrAlarmNeedMemberNameRemove)
	}

	c.Deps().Logger.Info("Alarm remove requested", zap.String("member", memberName))

	channel, err := FindActiveMemberOrError(ctx, c.Deps(), cmdCtx.Room, memberName)
	if err != nil {
		return err
	}

	removed, err := c.Deps().Alarm.RemoveAlarm(ctx, cmdCtx.Room, cmdCtx.UserID, channel.ID)
	if err != nil {
		c.Deps().Logger.Error("Failed to remove alarm",
			zap.String("channel", channel.Name),
			zap.Error(err),
		)
		return c.Deps().SendError(ctx, cmdCtx.Room, adapter.ErrAlarmRemoveFailed)
	}

	message := c.Deps().Formatter.FormatAlarmRemoved(channel.Name, removed)
	return c.Deps().SendMessage(ctx, cmdCtx.Room, message)
}

func (c *AlarmCommand) handleList(ctx context.Context, cmdCtx *domain.CommandContext) error {
	channelIDs, err := c.Deps().Alarm.GetUserAlarms(ctx, cmdCtx.Room, cmdCtx.UserID)
	if err != nil {
		return c.Deps().SendError(ctx, cmdCtx.Room, adapter.ErrAlarmListFailed)
	}

	alarmInfos := make([]adapter.AlarmListEntry, 0, len(channelIDs))
	for _, channelID := range channelIDs {
		memberName, err := c.Deps().Alarm.GetMemberName(ctx, channelID)
		if err != nil || memberName == "" {
			memberName = channelID
		}
		nextStreamInfo, _ := c.Deps().Alarm.GetNextStreamInfo(ctx, channelID)
		alarmInfos = append(alarmInfos, adapter.AlarmListEntry{
			MemberName: memberName,
			NextStream: nextStreamInfo,
		})
	}

	message := c.Deps().Formatter.FormatAlarmList(alarmInfos)
	return c.Deps().SendMessage(ctx, cmdCtx.Room, message)
}

func (c *AlarmCommand) handleClear(ctx context.Context, cmdCtx *domain.CommandContext) error {
	count, err := c.Deps().Alarm.ClearUserAlarms(ctx, cmdCtx.Room, cmdCtx.UserID)
	if err != nil {
		c.Deps().Logger.Error("Failed to clear alarms", zap.Error(err))
		return c.Deps().SendError(ctx, cmdCtx.Room, adapter.ErrAlarmClearFailed)
	}

	message := c.Deps().Formatter.FormatAlarmCleared(count)
	return c.Deps().SendMessage(ctx, cmdCtx.Room, message)
}
