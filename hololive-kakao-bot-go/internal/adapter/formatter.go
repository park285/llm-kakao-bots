package adapter

import (
	"fmt"
	"sort"
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

// ResponseFormatter: 봇의 응답 메시지를 생성하는 포맷터 (카카오톡 UI 템플릿 적용)
type ResponseFormatter struct {
	prefix string
}

func splitTemplateInstruction(rendered string) (instruction string, body string) {
	trimmed := strings.TrimLeft(rendered, "\r\n")
	if trimmed == "" {
		return "", ""
	}

	parts := strings.SplitN(trimmed, "\n", 2)
	instruction = util.TrimSpace(strings.TrimSuffix(parts[0], "\r"))
	if len(parts) < 2 {
		return instruction, ""
	}

	body = strings.TrimLeft(parts[1], "\r\n")
	return instruction, body
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

type liveStreamView struct {
	ChannelName string
	Title       string
	URL         string
}

type liveStreamsTemplateData struct {
	Emoji   UIEmoji
	Count   int
	Streams []liveStreamView
}

type upcomingStreamView struct {
	ChannelName string
	Title       string
	TimeInfo    string
	URL         string
}

type upcomingStreamsTemplateData struct {
	Emoji   UIEmoji
	Count   int
	Hours   int
	Streams []upcomingStreamView
}

type scheduleEntryView struct {
	IsLive   bool
	Title    string
	TimeInfo string
	URL      string
}

type channelScheduleTemplateData struct {
	Emoji       UIEmoji
	ChannelName string
	Days        int
	Count       int
	Streams     []scheduleEntryView
}

// MemberDirectoryGroup: 멤버 목록 표시를 위한 그룹 (예: 'JP 3기생', 'EN Promise')
type MemberDirectoryGroup struct {
	GroupName string
	Members   []MemberDirectoryEntry
}

// MemberDirectoryEntry: 멤버 목록의 개별 항목 (주 이름 및 보조 이름 포함)
type MemberDirectoryEntry struct {
	PrimaryName   string
	SecondaryName string
}

type memberDirectoryTemplateData struct {
	Emoji  UIEmoji
	Total  int
	Groups []memberDirectoryGroupView
}

type memberDirectoryGroupView struct {
	GroupName string
	Members   []memberDirectoryEntryView
}

type memberDirectoryEntryView struct {
	Primary   string
	Secondary string
	ShowBoth  bool
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

type helpTemplateData struct {
	Emoji  UIEmoji
	Prefix string
}

// NewResponseFormatter: 새로운 ResponseFormatter 인스턴스를 생성한다.
func NewResponseFormatter(prefix string) *ResponseFormatter {
	if util.TrimSpace(prefix) == "" {
		prefix = "!"
	}
	return &ResponseFormatter{prefix: prefix}
}

// Prefix: 현재 설정된 명령어 접두사를 반환한다.
func (f *ResponseFormatter) Prefix() string {
	if f == nil {
		return "!"
	}
	if trimmed := util.TrimSpace(f.prefix); trimmed != "" {
		return trimmed
	}
	return "!"
}

// FormatLiveStreams: 라이브 스트림 목록을 포맷팅하여 메시지 문자열을 생성한다.
func (f *ResponseFormatter) FormatLiveStreams(streams []*domain.Stream) string {
	data := liveStreamsTemplateData{Emoji: DefaultEmoji, Count: len(streams)}
	if len(streams) > 0 {
		data.Streams = make([]liveStreamView, len(streams))
		for i, stream := range streams {
			data.Streams[i] = liveStreamView{
				ChannelName: stream.ChannelName,
				Title:       f.truncateTitle(stream.Title),
				URL:         stream.GetYouTubeURL(),
			}
		}
	}

	rendered, err := executeFormatterTemplate("live_streams.tmpl", data)
	if err != nil {
		return ErrorMessage(ErrDisplayLiveStreamsFailed)
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

// UpcomingStreams: 예정된 방송 목록을 포맷팅하여 메시지 문자열을 생성한다.
func (f *ResponseFormatter) UpcomingStreams(streams []*domain.Stream, hours int) string {
	data := upcomingStreamsTemplateData{Emoji: DefaultEmoji, Count: len(streams), Hours: hours}
	if len(streams) > 0 {
		data.Streams = make([]upcomingStreamView, len(streams))
		for i, stream := range streams {
			data.Streams[i] = upcomingStreamView{
				ChannelName: stream.ChannelName,
				Title:       f.truncateTitle(stream.Title),
				TimeInfo:    f.streamTimeInfo(stream),
				URL:         stream.GetYouTubeURL(),
			}
		}
	}

	rendered, err := executeFormatterTemplate("upcoming_streams.tmpl", data)
	if err != nil {
		return ErrorMessage(ErrDisplayUpcomingFailed)
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

// ChannelSchedule: 특정 채널의 방송 일정을 포맷팅하여 메시지 문자열을 생성한다.
func (f *ResponseFormatter) ChannelSchedule(channel *domain.Channel, streams []*domain.Stream, days int) string {
	data := channelScheduleTemplateData{Emoji: DefaultEmoji, Days: days, Count: len(streams)}
	if channel != nil {
		data.ChannelName = channel.GetDisplayName()
	}
	if len(streams) > 0 {
		data.Streams = make([]scheduleEntryView, len(streams))
		for i, stream := range streams {
			entry := scheduleEntryView{
				Title: f.truncateTitle(stream.Title),
				URL:   stream.GetYouTubeURL(),
			}

			if stream.IsLive() {
				entry.IsLive = true
			} else {
				entry.TimeInfo = f.streamTimeInfo(stream)
			}

			data.Streams[i] = entry
		}
	}

	rendered, err := executeFormatterTemplate("channel_schedule.tmpl", data)
	if err != nil {
		return ErrorMessage(ErrDisplayScheduleFailed)
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

	sort.SliceStable(entries, func(i, j int) bool {
		if entries[i].ChannelName == entries[j].ChannelName {
			return entries[i].Title < entries[j].Title
		}
		return entries[i].ChannelName < entries[j].ChannelName
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

// MemberDirectory: 전체 멤버 디렉토리 목록을 포맷팅하여 메시지 문자열을 생성한다.
func (f *ResponseFormatter) MemberDirectory(groups []MemberDirectoryGroup, total int) string {
	viewGroups := prepareMemberDirectoryGroups(groups)

	if total <= 0 {
		for _, group := range viewGroups {
			total += len(group.Members)
		}
	}

	data := memberDirectoryTemplateData{
		Emoji:  DefaultEmoji,
		Total:  total,
		Groups: viewGroups,
	}

	rendered, err := executeFormatterTemplate("member_directory.tmpl", data)
	if err != nil {
		return ErrorMessage(ErrDisplayMemberListFailed)
	}

	if len(viewGroups) == 0 {
		return rendered
	}

	instruction, body := splitTemplateInstruction(rendered)
	if instruction == "" || body == "" {
		return rendered
	}
	return util.ApplyKakaoSeeMorePadding(body, instruction)
}

func prepareMemberDirectoryGroups(groups []MemberDirectoryGroup) []memberDirectoryGroupView {
	if len(groups) == 0 {
		return nil
	}

	views := make([]memberDirectoryGroupView, 0, len(groups))
	for _, group := range groups {
		name := util.TrimSpace(group.GroupName)
		if name == "" {
			name = "기타"
		}

		members := make([]memberDirectoryEntryView, 0, len(group.Members))
		for _, member := range group.Members {
			primary := util.TrimSpace(member.PrimaryName)
			secondary := util.TrimSpace(member.SecondaryName)
			if primary == "" && secondary == "" {
				continue
			}

			entry := memberDirectoryEntryView{
				Primary:   primary,
				Secondary: secondary,
				ShowBoth:  primary != "" && secondary != "" && !strings.EqualFold(primary, secondary),
			}
			members = append(members, entry)
		}

		if len(members) == 0 {
			continue
		}

		views = append(views, memberDirectoryGroupView{
			GroupName: name,
			Members:   members,
		})
	}

	return views
}

// FormatHelp: 도움말 메시지를 생성한다.
func (f *ResponseFormatter) FormatHelp() string {
	data := helpTemplateData{Emoji: DefaultEmoji, Prefix: f.prefix}
	rendered, err := executeFormatterTemplate("help.tmpl", data)
	if err != nil {
		return ErrorMessage(ErrDisplayHelpFailed)
	}

	instruction, body := splitTemplateInstruction(rendered)
	if instruction == "" || body == "" {
		return rendered
	}
	return util.ApplyKakaoSeeMorePadding(body, instruction)
}

// FormatError: 에러 메시지를 사용자 친화적인 포맷으로 변환한다.
func (f *ResponseFormatter) FormatError(message string) string {
	return ErrorMessage(message)
}

// MemberNotFound: 멤버를 찾을 수 없을 때의 에러 메시지를 생성한다.
func (f *ResponseFormatter) MemberNotFound(memberName string) string {
	return f.FormatError(fmt.Sprintf("'%s' 멤버를 찾을 수 없습니다.", memberName))
}

// 번역된 값 우선 사용, 없으면 원본 반환
func getTranslatedText(translatedVal, rawVal string) string {
	if trimmed := util.TrimSpace(translatedVal); trimmed != "" {
		return trimmed
	}
	return util.TrimSpace(rawVal)
}

// 캐치프레이즈 섹션 포맷팅
func formatProfileCatchphrase(raw *domain.TalentProfile, translated *domain.Translated) string {
	catchphrase := ""
	if translated != nil {
		catchphrase = getTranslatedText(translated.Catchphrase, raw.Catchphrase)
	} else if raw != nil {
		catchphrase = util.TrimSpace(raw.Catchphrase)
	}

	if catchphrase == "" {
		return ""
	}
	return fmt.Sprintf("%s %s\n", DefaultEmoji.Speech, catchphrase)
}

// 요약 섹션 포맷팅
func formatProfileSummary(raw *domain.TalentProfile, translated *domain.Translated) string {
	summary := ""
	if translated != nil {
		summary = getTranslatedText(translated.Summary, raw.Description)
	} else if raw != nil {
		summary = util.TrimSpace(raw.Description)
	}

	if summary == "" {
		return ""
	}
	return summary + "\n"
}

// 하이라이트 섹션 포맷팅
func formatProfileHighlights(translated *domain.Translated) string {
	if translated == nil || len(translated.Highlights) == 0 {
		return ""
	}

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("\n%s 하이라이트\n", DefaultEmoji.Highlight))
	for _, highlight := range translated.Highlights {
		if trimmed := util.TrimSpace(highlight); trimmed != "" {
			sb.WriteString(fmt.Sprintf("- %s\n", trimmed))
		}
	}
	return sb.String()
}

// 번역된 데이터 또는 원본 데이터 반환
func getProfileDataEntries(raw *domain.TalentProfile, translated *domain.Translated) []domain.TranslatedProfileDataRow {
	if translated != nil && len(translated.Data) > 0 {
		return translated.Data
	}

	if raw == nil || len(raw.DataEntries) == 0 {
		return nil
	}

	entries := make([]domain.TranslatedProfileDataRow, 0)
	for _, entry := range raw.DataEntries {
		if util.TrimSpace(entry.Label) == "" || util.TrimSpace(entry.Value) == "" {
			continue
		}
		entries = append(entries, domain.TranslatedProfileDataRow(entry))
	}
	return entries
}

// 프로필 데이터 섹션 포맷팅 (최대 8개)
func formatProfileDataEntries(raw *domain.TalentProfile, translated *domain.Translated) string {
	dataEntries := getProfileDataEntries(raw, translated)
	if len(dataEntries) == 0 {
		return ""
	}

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("\n%s 프로필 데이터\n", DefaultEmoji.Data))

	maxRows := len(dataEntries)
	if maxRows > 8 {
		maxRows = 8
	}

	for i := 0; i < maxRows; i++ {
		row := dataEntries[i]
		label := util.TrimSpace(row.Label)
		value := util.TrimSpace(row.Value)
		if label == "" || value == "" {
			continue
		}

		if strings.Contains(value, "\n") {
			indented := "  " + strings.ReplaceAll(value, "\n", "\n  ")
			sb.WriteString(fmt.Sprintf("- %s:\n%s\n", label, indented))
		} else {
			sb.WriteString(fmt.Sprintf("- %s: %s\n", label, value))
		}
	}

	return sb.String()
}

// 소셜 링크 섹션 포맷팅 (최대 4개)
func formatProfileSocialLinks(raw *domain.TalentProfile) string {
	if raw == nil || len(raw.SocialLinks) == 0 {
		return ""
	}

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("\n%s 링크\n", DefaultEmoji.Link))

	maxLinks := len(raw.SocialLinks)
	if maxLinks > 4 {
		maxLinks = 4
	}

	for i := 0; i < maxLinks; i++ {
		link := raw.SocialLinks[i]
		if util.TrimSpace(link.Label) == "" || util.TrimSpace(link.URL) == "" {
			continue
		}
		translatedLabel := socialLinkLabel(link.Label)
		sb.WriteString(fmt.Sprintf("- %s: %s\n", translatedLabel, util.TrimSpace(link.URL)))
	}

	return sb.String()
}

// 공식 URL 섹션 포맷팅
func formatProfileOfficialURL(raw *domain.TalentProfile) string {
	if raw == nil || util.TrimSpace(raw.OfficialURL) == "" {
		return ""
	}
	return fmt.Sprintf("\n%s 공식 프로필: %s", DefaultEmoji.Web, util.TrimSpace(raw.OfficialURL))
}

// FormatTalentProfile: 탤런트 프로필 정보를 포맷팅하여 메시지 문자열을 생성한다.
func (f *ResponseFormatter) FormatTalentProfile(raw *domain.TalentProfile, translated *domain.Translated) string {
	if raw == nil {
		return ErrorMessage(ErrDisplayProfileDataFailed)
	}

	var sb strings.Builder
	header := buildTalentHeader(raw, translated)
	sb.WriteString(header)
	sb.WriteString("\n")

	sb.WriteString(formatProfileCatchphrase(raw, translated))
	sb.WriteString(formatProfileSummary(raw, translated))
	sb.WriteString(formatProfileHighlights(translated))
	sb.WriteString(formatProfileDataEntries(raw, translated))
	sb.WriteString(formatProfileSocialLinks(raw))
	sb.WriteString(formatProfileOfficialURL(raw))

	content := util.TrimSpace(sb.String())
	if content == "" {
		return content
	}

	body := util.StripLeadingHeader(content, header)
	body = util.TrimSpace(body)
	if body == "" {
		return content
	}

	instructionBase := util.TrimSpace(header)
	if instructionBase == "" {
		instructionBase = DefaultEmoji.Member + " 멤버 정보"
	}

	return util.ApplyKakaoSeeMorePadding(body, instructionBase)
}

func socialLinkLabel(label string) string {
	translations := map[string]string{
		"歌の再生リスト":   "음악 플레이리스트",
		"公式グッズ":     "공식 굿즈",
		"オフィシャルグッズ": "공식 굿즈",
	}

	if korean, ok := translations[label]; ok {
		return korean
	}
	return label
}

func buildTalentHeader(raw *domain.TalentProfile, translated *domain.Translated) string {
	names := talentDisplayNames(raw, translated)
	return MemberHeader(names)
}

func talentDisplayNames(raw *domain.TalentProfile, translated *domain.Translated) []string {
	var names []string

	english := ""
	japanese := ""
	if raw != nil {
		english = util.TrimSpace(raw.EnglishName)
		japanese = util.TrimSpace(raw.JapaneseName)
	}

	display := ""
	if translated != nil {
		display = util.TrimSpace(translated.DisplayName)
	}

	if english != "" {
		addUniqueName(&names, english)
	}

	for _, candidate := range parseDisplayNameComponents(display) {
		addUniqueName(&names, candidate)
	}

	if japanese != "" {
		addUniqueName(&names, japanese)
	}

	return names
}

func parseDisplayNameComponents(display string) []string {
	display = util.TrimSpace(display)
	if display == "" {
		return nil
	}

	var rawParts []string

	openIdx := strings.Index(display, "(")
	closeIdx := strings.LastIndex(display, ")")
	if openIdx != -1 && closeIdx != -1 && closeIdx > openIdx {
		before := util.TrimSpace(display[:openIdx])
		inside := util.TrimSpace(display[openIdx+1 : closeIdx])
		after := util.TrimSpace(display[closeIdx+1:])

		if before != "" {
			rawParts = append(rawParts, before)
		}
		if inside != "" {
			rawParts = append(rawParts, inside)
		}
		if after != "" {
			rawParts = append(rawParts, after)
		}
	} else {
		rawParts = append(rawParts, display)
	}

	var result []string
	for _, part := range rawParts {
		segments := strings.Split(part, "/")
		for _, segment := range segments {
			candidate := util.TrimSpace(segment)
			if candidate != "" {
				result = append(result, candidate)
			}
		}
	}

	return result
}

func addUniqueName(names *[]string, candidate string) {
	candidate = util.TrimSpace(candidate)
	if candidate == "" {
		return
	}

	for _, existing := range *names {
		if strings.EqualFold(existing, candidate) {
			return
		}
	}

	*names = append(*names, candidate)
}

func (f *ResponseFormatter) truncateTitle(title string) string {
	return util.TruncateString(title, constants.StringLimits.StreamTitle)
}

func (f *ResponseFormatter) streamTimeInfo(stream *domain.Stream) string {
	if stream == nil || stream.StartScheduled == nil {
		return MsgTimeUnknown
	}

	kstTime := util.FormatKST(*stream.StartScheduled, "01/02 15:04")
	minutesUntil := stream.MinutesUntilStart()

	if minutesUntil <= 0 {
		return kstTime
	}

	hoursUntil := minutesUntil / 60
	minutesRem := minutesUntil % 60

	if hoursUntil > 24 {
		daysUntil := hoursUntil / 24
		return fmt.Sprintf("%s (%d일 후)", kstTime, daysUntil)
	} else if hoursUntil > 0 {
		return fmt.Sprintf("%s (%d시간 %d분 후)", kstTime, hoursUntil, minutesRem)
	} else {
		return fmt.Sprintf("%s (%d분 후)", kstTime, minutesRem)
	}
}
