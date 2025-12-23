package holodex

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"
	"go.uber.org/zap"

	"github.com/kapu/hololive-kakao-bot-go/internal/constants"
	"github.com/kapu/hololive-kakao-bot-go/internal/domain"
	"github.com/kapu/hololive-kakao-bot-go/internal/service/cache"
	"github.com/kapu/hololive-kakao-bot-go/internal/util"
)

// ScraperService 는 타입이다.
type ScraperService struct {
	httpClient    *http.Client
	cache         *cache.Service
	membersData   domain.MemberDataProvider
	memberNameMap map[string]string // memberName -> channelID
	logger        *zap.Logger
	baseURL       string
}

const (
	scraperChannelCacheKeyPrefix = "scraper:channel:"
)

// NewScraperService 는 동작을 수행한다.
func NewScraperService(cache *cache.Service, membersData domain.MemberDataProvider, logger *zap.Logger) *ScraperService {
	nameMap := make(map[string]string)

	for _, member := range membersData.GetAllMembers() {
		nameMap[util.Normalize(member.Name)] = member.ChannelID

		if member.NameJa != "" {
			nameMap[util.Normalize(member.NameJa)] = member.ChannelID
		}

		if member.Aliases != nil {
			for _, alias := range member.Aliases.Ko {
				nameMap[util.Normalize(alias)] = member.ChannelID
			}
			for _, alias := range member.Aliases.Ja {
				nameMap[util.Normalize(alias)] = member.ChannelID
			}
		}
	}

	logger.Info("Scraper initialized with member matching",
		zap.Int("members", len(membersData.GetAllMembers())),
		zap.Int("name_mappings", len(nameMap)))

	return &ScraperService{
		httpClient: &http.Client{
			Timeout: constants.OfficialScheduleConfig.Timeout,
		},
		cache:         cache,
		membersData:   membersData,
		memberNameMap: nameMap,
		logger:        logger,
		baseURL:       constants.OfficialScheduleConfig.BaseURL,
	}
}

// FetchChannel 는 동작을 수행한다.
func (s *ScraperService) FetchChannel(ctx context.Context, channelID string) ([]*domain.Stream, error) {
	cacheKey := scraperChannelCacheKeyPrefix + channelID
	if cached, found := s.cache.GetStreams(ctx, cacheKey); found {
		s.logger.Debug("Scraper cache hit", zap.String("channel", channelID))
		return cached, nil
	}

	s.logger.Info("Fetching from official schedule (FALLBACK MODE)",
		zap.String("channel", channelID),
		zap.String("url", s.baseURL))

	allStreams, err := s.fetchAllStreams(ctx)
	if err != nil {
		return nil, fmt.Errorf("scraper failed: %w", err)
	}

	channelStreams := make([]*domain.Stream, 0)
	for _, stream := range allStreams {
		if stream.ChannelID == channelID {
			channelStreams = append(channelStreams, stream)
		}
	}

	s.cache.SetStreams(ctx, cacheKey, channelStreams, constants.OfficialScheduleConfig.CacheExpiry)

	s.logger.Info("Scraper completed",
		zap.String("channel", channelID),
		zap.Int("streams", len(channelStreams)))

	return channelStreams, nil
}

// FetchAllStreams 는 동작을 수행한다.
func (s *ScraperService) FetchAllStreams(ctx context.Context) ([]*domain.Stream, error) {
	return s.fetchAllStreams(ctx)
}

func (s *ScraperService) fetchAllStreams(ctx context.Context) ([]*domain.Stream, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", s.baseURL+"/lives/hololive", http.NoBody)
	if err != nil {
		return nil, fmt.Errorf("failed to create scraper request: %w", err)
	}

	req.Header.Set("User-Agent", "Mozilla/5.0 (compatible; HololiveBot/1.0)")

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("HTTP request failed: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	doc, err := goquery.NewDocumentFromReader(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("HTML parse failed: %w", err)
	}

	streams := make([]*domain.Stream, 0)
	parseErrors := 0
	currentDate := ""

	doc.Find(".container .col-12").Each(func(i int, container *goquery.Selection) {
		dateHeader := container.Find(".navbar-inverse .holodule.navbar-text")
		if dateHeader.Length() > 0 {
			dateText := util.TrimSpace(dateHeader.Text())
			dateText = strings.Split(dateText, "(")[0]
			currentDate = util.TrimSpace(dateText)
			s.logger.Debug("Found date section", zap.String("date", currentDate))
			return
		}

		container.Find("a.thumbnail").Each(func(j int, sel *goquery.Selection) {
			stream, err := s.parseStreamElement(sel, currentDate)
			if err != nil {
				parseErrors++
				s.logger.Debug("Failed to parse stream element",
					zap.String("date", currentDate),
					zap.Error(err))
				return
			}

			if stream != nil {
				streams = append(streams, stream)
			}
		})
	})

	if len(streams) == 0 {
		return nil, &StructureChangedError{
			Message:     "No streams found - HTML structure may have changed",
			ParseErrors: parseErrors,
		}
	}

	if parseErrors > len(streams)/2 {
		s.logger.Warn("High parse error rate detected",
			zap.Int("successes", len(streams)),
			zap.Int("errors", parseErrors))
	}

	s.logger.Info("Scraper fetched all streams",
		zap.Int("total", len(streams)),
		zap.Int("parse_errors", parseErrors))

	return streams, nil
}

func (s *ScraperService) parseStreamElement(sel *goquery.Selection, currentDate string) (*domain.Stream, error) {
	videoURL, exists := sel.Attr("href")
	if !exists || !strings.Contains(videoURL, "youtube.com/watch?v=") {
		return nil, fmt.Errorf("invalid video URL")
	}

	videoID := s.extractVideoID(videoURL)
	if videoID == "" {
		return nil, fmt.Errorf("could not extract video ID from %s", videoURL)
	}

	timeText := util.TrimSpace(sel.Find(".datetime").Text())
	memberName := util.TrimSpace(sel.Find(".name").Text())
	if memberName == "" {
		memberName = util.TrimSpace(sel.Find(".text").Text())
	}

	if onclickStr, exists := sel.Attr("onclick"); exists {
		if extractedName := s.extractMemberFromOnClick(onclickStr); extractedName != "" {
			memberName = extractedName
		}
	}

	startTime, err := s.parseDatetimeWithContext(currentDate, timeText)
	if err != nil {
		s.logger.Debug("Failed to parse datetime",
			zap.String("date", currentDate),
			zap.String("time", timeText),
			zap.Error(err))
	}

	thumbnailURL := fmt.Sprintf("https://img.youtube.com/vi/%s/mqdefault.jpg", videoID)

	channelID := s.matchMemberToChannel(memberName)
	if channelID == "" {
		s.logger.Debug("Could not match member name to channel ID",
			zap.String("member_name", memberName),
			zap.String("video_id", videoID))
	}

	stream := &domain.Stream{
		ID:             videoID,
		Title:          memberName,
		ChannelID:      channelID,
		ChannelName:    memberName,
		Status:         domain.StreamStatusUpcoming,
		StartScheduled: startTime,
		Link:           &videoURL,
		Thumbnail:      &thumbnailURL,
	}

	return stream, nil
}

func (s *ScraperService) matchMemberToChannel(memberName string) string {
	if memberName == "" {
		return ""
	}

	normalized := util.Normalize(memberName)
	if channelID, found := s.memberNameMap[normalized]; found {
		return channelID
	}

	for name, channelID := range s.memberNameMap {
		if strings.Contains(name, normalized) || strings.Contains(normalized, name) {
			s.logger.Debug("Matched member via partial match",
				zap.String("scraped", memberName),
				zap.String("matched", name),
				zap.String("channel_id", channelID))
			return channelID
		}
	}

	return ""
}

func (s *ScraperService) extractVideoID(videoURL string) string {
	parts := strings.Split(videoURL, "?v=")
	if len(parts) < 2 {
		return ""
	}

	videoID := parts[1]
	if idx := strings.Index(videoID, "&"); idx != -1 {
		videoID = videoID[:idx]
	}

	return videoID
}

func (s *ScraperService) parseDatetimeWithContext(date, timeStr string) (*time.Time, error) {
	date = util.TrimSpace(date)
	timeStr = util.TrimSpace(timeStr)

	if date == "" || timeStr == "" {
		return nil, fmt.Errorf("empty date or time")
	}

	combined := fmt.Sprintf("%s %s", date, timeStr)

	jst, _ := time.LoadLocation("Asia/Tokyo")
	now := time.Now().In(jst)

	t, err := time.ParseInLocation("01/02 15:04", combined, jst)
	if err != nil {
		return nil, fmt.Errorf("failed to parse %s: %w", combined, err)
	}

	result := time.Date(now.Year(), t.Month(), t.Day(), t.Hour(), t.Minute(), 0, 0, jst)
	if result.Before(now.Add(-90 * 24 * time.Hour)) {
		result = result.AddDate(1, 0, 0)
	}

	return &result, nil
}

func (s *ScraperService) extractMemberFromOnClick(onclick string) string {
	startMarker := "event_category':'"
	startIdx := strings.Index(onclick, startMarker)
	if startIdx == -1 {
		startMarker = `event_category":"`
		startIdx = strings.Index(onclick, startMarker)
	}

	if startIdx == -1 {
		return ""
	}

	startIdx += len(startMarker)
	endIdx := strings.Index(onclick[startIdx:], "'")
	if endIdx == -1 {
		endIdx = strings.Index(onclick[startIdx:], `"`)
	}

	if endIdx == -1 {
		return ""
	}

	return onclick[startIdx : startIdx+endIdx]
}

// ValidateStructure 는 동작을 수행한다.
func (s *ScraperService) ValidateStructure(ctx context.Context) error {
	_, err := s.fetchAllStreams(ctx)
	return err
}

// StructureChangedError 는 타입이다.
type StructureChangedError struct {
	Message     string
	ParseErrors int
}

func (e *StructureChangedError) Error() string {
	return fmt.Sprintf("%s (parse errors: %d)", e.Message, e.ParseErrors)
}

// IsStructureError 는 동작을 수행한다.
func IsStructureError(err error) bool {
	structureChangedError := &StructureChangedError{}
	ok := errors.As(err, &structureChangedError)
	return ok
}
