package command

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/kapu/hololive-kakao-bot-go/internal/adapter"
	"github.com/kapu/hololive-kakao-bot-go/internal/domain"
)

// AlarmCommand: 알람 설정 및 관리를 담당하는 커맨드 핸들러
type AlarmCommand struct {
	BaseCommand
}

// NewAlarmCommand: 알람 관리 커맨드 핸들러를 생성합니다.
func NewAlarmCommand(deps *Dependencies) *AlarmCommand {
	return &AlarmCommand{BaseCommand: NewBaseCommand(deps)}
}

// Name: 커맨드의 이름("alarm")을 반환합니다.
func (c *AlarmCommand) Name() string {
	return "alarm"
}

// Description: 커맨드에 대한 설명을 반환합니다.
func (c *AlarmCommand) Description() string {
	return "방송 알람 관리"
}

// Execute: 알람 추가, 삭제, 목록 조회 등의 작업을 수행합니다.
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
			slog.String("room", cmdCtx.Room),
			slog.String("sender", cmdCtx.UserName),
			slog.String("sub_command", subCmd),
			slog.String("member", memberName),
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

	c.Deps().Logger.Info("Alarm add requested", slog.String("member", memberName))

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
			slog.String("channel", channel.Name),
			slog.Any("error", err),
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

	c.Deps().Logger.Info("Alarm remove requested", slog.String("member", memberName))

	channel, err := FindActiveMemberOrError(ctx, c.Deps(), cmdCtx.Room, memberName)
	if err != nil {
		return err
	}

	removed, err := c.Deps().Alarm.RemoveAlarm(ctx, cmdCtx.Room, cmdCtx.UserID, channel.ID)
	if err != nil {
		c.Deps().Logger.Error("Failed to remove alarm",
			slog.String("channel", channel.Name),
			slog.Any("error", err),
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
		memberName := c.Deps().Alarm.GetMemberNameWithFallback(ctx, channelID)
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
		c.Deps().Logger.Error("Failed to clear alarms", slog.Any("error", err))
		return c.Deps().SendError(ctx, cmdCtx.Room, adapter.ErrAlarmClearFailed)
	}

	message := c.Deps().Formatter.FormatAlarmCleared(count)
	return c.Deps().SendMessage(ctx, cmdCtx.Room, message)
}
