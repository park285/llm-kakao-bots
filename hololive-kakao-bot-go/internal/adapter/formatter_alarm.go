package adapter

import (
	"fmt"
	"slices"
	"strings"
	"time"

	"github.com/kapu/hololive-kakao-bot-go/internal/constants"
	"github.com/kapu/hololive-kakao-bot-go/internal/domain"
	"github.com/kapu/hololive-kakao-bot-go/internal/util"
)

// AlarmListEntry: 알림 목록 조회를 위한 개별 항목 (멤버 이름 및 다음 방송 정보 포함)
type AlarmListEntry struct {
	MemberName string
	NextStream *domain.NextStreamInfo
}

type alarmAddedTemplateData struct {
	Emoji      UIEmoji
	MemberName string
	Added      bool
	NextStream *nextStreamInfoView
	Prefix     string
}

type alarmRemovedTemplateData struct {
	Emoji      UIEmoji
	MemberName string
	Removed    bool
}

type alarmListTemplateData struct {
	Emoji  UIEmoji
	Count  int
	Prefix string
	Alarms []alarmListEntryView
}

type alarmListEntryView struct {
	MemberName string
	NextStream *nextStreamInfoView
}

type nextStreamInfoView struct {
	Status       string
	Title        string
	URL          string
	ScheduledKST string
	TimeDetail   string
	StartingSoon bool
}

type alarmClearedTemplateData struct {
	Emoji UIEmoji
	Count int
}

type alarmNotificationTemplateData struct {
	Emoji           UIEmoji
	ChannelName     string
	MinutesUntil    int
	Title           string
	URL             string
	ScheduleMessage string
}

func alarmChannelName(notification *domain.AlarmNotification) string {
	if notification == nil {
		return ""
	}

	if notification.Channel != nil {
		if name := util.TrimSpace(notification.Channel.GetDisplayName()); name != "" {
			return name
		}
	}

	if notification.Stream != nil && util.TrimSpace(notification.Stream.ChannelName) != "" {
		return notification.Stream.ChannelName
	}

	return ""
}

// FormatAlarmAdded: 알림 추가 성공 메시지를 생성한다. (다음 정규 방송 정보 포함)
func (f *ResponseFormatter) FormatAlarmAdded(memberName string, added bool, nextStreamInfo *domain.NextStreamInfo) string {
	data := alarmAddedTemplateData{
		Emoji:      DefaultEmoji,
		MemberName: memberName,
		Added:      added,
		NextStream: buildNextStreamInfoView(nextStreamInfo),
		Prefix:     f.prefix,
	}

	rendered, err := executeFormatterTemplate("alarm_added.tmpl", data)
	if err != nil {
		return ErrorMessage(ErrDisplayAlarmAddFailed)
	}

	return rendered
}

// FormatAlarmRemoved: 알림 삭제 성공 메시지를 생성한다.
func (f *ResponseFormatter) FormatAlarmRemoved(memberName string, removed bool) string {
	data := alarmRemovedTemplateData{
		Emoji:      DefaultEmoji,
		MemberName: memberName,
		Removed:    removed,
	}

	rendered, err := executeFormatterTemplate("alarm_removed.tmpl", data)
	if err != nil {
		return ErrorMessage(ErrDisplayAlarmRemoveFailed)
	}

	return rendered
}

const youtubeWatchURLPrefix = "https://youtube.com/watch?v="

func summarizeNextStreamInfo(info *domain.NextStreamInfo) *domain.NextStreamInfo {
	if info == nil || !info.Status.IsLive() {
		return nil
	}
	return info
}

func buildNextStreamInfoView(info *domain.NextStreamInfo) *nextStreamInfoView {
	if info == nil || !info.Status.IsValid() {
		return nil
	}

	view := &nextStreamInfoView{
		Status: info.Status.String(),
	}

	if title := util.TrimSpace(info.Title); title != "" {
		view.Title = util.TruncateString(title, constants.StringLimits.NextStreamTitle)
	}

	if videoID := util.TrimSpace(info.VideoID); videoID != "" {
		view.URL = youtubeWatchURLPrefix + videoID
	}

	if info.Status.IsUpcoming() {
		if info.StartScheduled == nil || view.URL == "" {
			return nil
		}

		scheduled := *info.StartScheduled
		view.ScheduledKST = util.FormatKST(scheduled, "01/02 15:04")

		timeLeft := time.Until(scheduled)
		if timeLeft <= 0 {
			view.StartingSoon = true
		} else {
			view.TimeDetail = formatUpcomingTimeDetail(timeLeft)
		}
	}

	return view
}

func formatUpcomingTimeDetail(timeLeft time.Duration) string {
	if timeLeft <= 0 {
		return ""
	}

	hoursLeft := int(timeLeft.Hours())
	minutesLeft := int(timeLeft.Minutes()) % 60

	switch {
	case hoursLeft >= 24:
		return fmt.Sprintf("%d일 후", hoursLeft/24)
	case hoursLeft > 0:
		return fmt.Sprintf("%d시간 %d분 후", hoursLeft, minutesLeft)
	default:
		return fmt.Sprintf("%d분 후", int(timeLeft.Minutes()))
	}
}

// FormatAlarmList: 사용자의 현재 알림 목록을 포맷팅하여 메시지 문자열을 생성한다.
func (f *ResponseFormatter) FormatAlarmList(alarms []AlarmListEntry) string {
	processed := make([]alarmListEntryView, len(alarms))
	for idx, alarm := range alarms {
		processed[idx] = alarmListEntryView{
			MemberName: alarm.MemberName,
			NextStream: buildNextStreamInfoView(summarizeNextStreamInfo(alarm.NextStream)),
		}
	}

	data := alarmListTemplateData{
		Emoji:  DefaultEmoji,
		Count:  len(processed),
		Prefix: f.prefix,
		Alarms: processed,
	}

	rendered, err := executeFormatterTemplate("alarm_list.tmpl", data)
	if err != nil {
		return ErrorMessage(ErrDisplayAlarmListFailed)
	}

	if data.Count == 0 {
		return rendered
	}
	instruction, body := splitTemplateInstruction(rendered)
	if instruction == "" || body == "" {
		return rendered
	}
	return util.ApplyKakaoSeeMorePadding(body, instruction)
}

// FormatAlarmCleared: 알림 전체 삭제 완료 메시지를 생성한다.
func (f *ResponseFormatter) FormatAlarmCleared(count int) string {
	data := alarmClearedTemplateData{Emoji: DefaultEmoji, Count: count}
	rendered, err := executeFormatterTemplate("alarm_cleared.tmpl", data)
	if err != nil {
		return ErrorMessage(ErrDisplayAlarmClearFailed)
	}

	return rendered
}

// InvalidAlarmUsage: 알림 명령어의 잘못된 사용법에 대한 안내 메시지를 반환한다.
func (f *ResponseFormatter) InvalidAlarmUsage() string {
	return ErrInvalidAlarmUsage
}

// AlarmNotification: 단일 방송 알림 메시지를 생성한다.
func (f *ResponseFormatter) AlarmNotification(notification *domain.AlarmNotification) string {
	if notification == nil || notification.Stream == nil {
		return ""
	}

	channelName := alarmChannelName(notification)

	data := alarmNotificationTemplateData{
		Emoji:           DefaultEmoji,
		ChannelName:     channelName,
		MinutesUntil:    notification.MinutesUntil,
		Title:           util.TruncateString(notification.Stream.Title, constants.StringLimits.StreamTitle),
		URL:             notification.Stream.GetYouTubeURL(),
		ScheduleMessage: notification.ScheduleChangeMessage,
	}

	rendered, err := executeFormatterTemplate("alarm_notification.tmpl", data)
	if err != nil {
		return ErrorMessage(ErrDisplayAlarmNotifyFailed)
	}

	instruction, body := splitTemplateInstruction(rendered)
	if instruction == "" || body == "" {
		return rendered
	}
	return util.ApplyKakaoSeeMorePadding(body, instruction)
}

// AlarmNotificationGroup: 여러 방송의 알림을 하나로 묶어 그룹 메시지를 생성한다. (알림 폭탄 방지)
func (f *ResponseFormatter) AlarmNotificationGroup(minutesUntil int, notifications []*domain.AlarmNotification) string {
	if len(notifications) == 0 {
		return ""
	}

	type entry struct {
		ChannelName string
		Title       string
		URL         string
	}

	entries := make([]entry, 0, len(notifications))
	for _, notification := range notifications {
		if notification == nil || notification.Stream == nil {
			continue
		}

		entries = append(entries, entry{
			ChannelName: alarmChannelName(notification),
			Title:       util.TruncateString(util.TrimSpace(notification.Stream.Title), constants.StringLimits.StreamTitle),
			URL:         util.TrimSpace(notification.Stream.GetYouTubeURL()),
		})
	}

	if len(entries) == 0 {
		return ""
	}

	slices.SortStableFunc(entries, func(a, b entry) int {
		if a.ChannelName != b.ChannelName {
			if a.ChannelName < b.ChannelName {
				return -1
			}
			return 1
		}
		if a.Title < b.Title {
			return -1
		}
		if a.Title > b.Title {
			return 1
		}
		return 0
	})

	instruction := DefaultEmoji.Alarm + " 방송 알림"

	var sb strings.Builder
	sb.WriteString(CountedHeader(DefaultEmoji.Alarm, "방송 알림", len(entries)))
	sb.WriteString("\n\n")
	sb.WriteString(DefaultEmoji.Time + " 여러 방송이 곧 시작됩니다.\n\n")

	for idx, entry := range entries {
		name := util.TrimSpace(entry.ChannelName)
		if name == "" {
			name = "알 수 없는 채널"
		}

		sb.WriteString(fmt.Sprintf("%d. %s\n", idx+1, name))

		if entry.Title != "" {
			sb.WriteString(fmt.Sprintf("   %s\n", entry.Title))
		}

		if entry.URL != "" {
			sb.WriteString(fmt.Sprintf("   %s\n", entry.URL))
		}

		if idx < len(entries)-1 {
			sb.WriteString("\n")
		}
	}

	content := util.TrimSpace(sb.String())
	if content == "" {
		return ""
	}

	return util.ApplyKakaoSeeMorePadding(content, instruction)
}
