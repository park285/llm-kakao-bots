package adapter

import (
	"fmt"

	"github.com/kapu/hololive-kakao-bot-go/internal/constants"
	"github.com/kapu/hololive-kakao-bot-go/internal/domain"
	"github.com/kapu/hololive-kakao-bot-go/internal/util"
)

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

// FormatLiveStreams: 라이브 스트림 목록을 포맷팅하여 메시지 문자열을 생성합니다.
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

// UpcomingStreams: 예정된 방송 목록을 포맷팅하여 메시지 문자열을 생성합니다.
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

// ChannelSchedule: 특정 채널의 방송 일정을 포맷팅하여 메시지 문자열을 생성합니다.
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
