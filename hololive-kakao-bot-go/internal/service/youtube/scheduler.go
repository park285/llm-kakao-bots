package youtube

import (
	"context"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"github.com/kapu/hololive-kakao-bot-go/internal/domain"
	"github.com/kapu/hololive-kakao-bot-go/internal/service/cache"
	"github.com/kapu/hololive-kakao-bot-go/internal/util"
)

// Scheduler: YouTube 데이터 수집(통계, 영상 등) 작업을 주기적으로 실행하는 스케줄러
type Scheduler struct {
	youtube      *Service
	cache        *cache.Service
	statsRepo    *StatsRepository
	membersData  domain.MemberDataProvider
	logger       *slog.Logger
	ticker       *time.Ticker
	stopCh       chan struct{}
	currentBatch int
	batchMu      sync.Mutex
}

const (
	schedulerInterval = 12 * time.Hour

	channelsPerBatch = 30 // 30 channels × 100 units = 3,000 units per batch
	batchesPerDay    = 2  // 2 batches × 3,000 = 6,000 units
	totalDailyQuota  = 6000
)

// NewScheduler: YouTube 데이터 수집 스케줄러를 생성한다.
func NewScheduler(youtube *Service, cache *cache.Service, statsRepo *StatsRepository, membersData domain.MemberDataProvider, logger *slog.Logger) *Scheduler {
	return &Scheduler{
		youtube:      youtube,
		cache:        cache,
		statsRepo:    statsRepo,
		membersData:  membersData,
		logger:       logger,
		currentBatch: 0,
		stopCh:       make(chan struct{}),
	}
}

// Start: 스케줄러를 시작하여 주기적인 작업을 등록한다.
func (ys *Scheduler) Start(ctx context.Context) {
	ys.ticker = time.NewTicker(schedulerInterval)

	ys.logger.Info("YouTube quota building scheduler started",
		slog.Duration("interval", schedulerInterval),
		slog.Int("channels_per_batch", channelsPerBatch),
		slog.Int("daily_quota_target", totalDailyQuota))

	go func() {
		for {
			select {
			case <-ys.ticker.C:
				ys.runBatch(ctx)
			case <-ys.stopCh:
				ys.logger.Info("YouTube scheduler stopped")
				return
			case <-ctx.Done():
				ys.logger.Info("YouTube scheduler context canceled")
				return
			}
		}
	}()
}

// Stop: 스케줄러를 중지한다.
func (ys *Scheduler) Stop() {
	if ys.ticker != nil {
		ys.ticker.Stop()
	}
	close(ys.stopCh)
}

func (ys *Scheduler) runBatch(ctx context.Context) {
	ys.batchMu.Lock()
	batchNum := ys.currentBatch
	ys.currentBatch = (ys.currentBatch + 1) % batchesPerDay
	ys.batchMu.Unlock()

	ys.logger.Info("Running YouTube quota building batch",
		slog.Int("batch", batchNum),
		slog.Int("total_batches", batchesPerDay))

	go ys.trackAllSubscribers(ctx)

	go ys.fetchRecentVideosRotation(ctx, batchNum)
}

func (ys *Scheduler) trackAllSubscribers(ctx context.Context) {
	channelIDs, channelToMember := ys.buildChannelMaps()

	ys.logger.Info("Tracking all member subscribers",
		slog.Int("channels", len(channelIDs)),
		slog.Int("quota_cost", len(channelIDs)))

	stats, err := ys.youtube.GetChannelStatistics(ctx, channelIDs)
	if err != nil {
		ys.logger.Error("Failed to track subscribers", slog.Any("error", err))
		return
	}

	now := time.Now()
	totalChanges := 0
	totalMilestones := 0

	for channelID, currentStats := range stats {
		member := channelToMember[channelID]
		if member == nil {
			continue
		}

		changes, milestones := ys.processChannelStats(ctx, channelID, member, currentStats, now)
		totalChanges += changes
		totalMilestones += milestones
	}

	ys.logger.Info("Subscriber tracking completed",
		slog.Int("tracked", len(stats)),
		slog.Int("changes", totalChanges),
		slog.Int("milestones", totalMilestones))
}

// 멤버 데이터에서 채널 ID 리스트와 채널-멤버 맵 생성
func (ys *Scheduler) buildChannelMaps() ([]string, map[string]*domain.Member) {
	channelIDs := make([]string, 0, len(ys.membersData.GetAllMembers()))
	channelToMember := make(map[string]*domain.Member)

	for _, member := range ys.membersData.GetAllMembers() {
		channelIDs = append(channelIDs, member.ChannelID)
		channelToMember[member.ChannelID] = member
	}

	return channelIDs, channelToMember
}

// 현재 통계를 TimestampedStats 객체로 변환
func createTimestampedStats(channelID string, member *domain.Member, stats *ChannelStats, timestamp time.Time) *domain.TimestampedStats {
	return &domain.TimestampedStats{
		ChannelID:       channelID,
		MemberName:      member.Name,
		SubscriberCount: stats.SubscriberCount,
		VideoCount:      stats.VideoCount,
		ViewCount:       stats.ViewCount,
		Timestamp:       timestamp,
	}
}

// 이전 통계와 현재 통계를 비교하여 변경값 계산
func calculateStatsChanges(prev *domain.TimestampedStats, current *ChannelStats) (subChange, vidChange, viewChange int64) {
	subChange = int64(current.SubscriberCount) - int64(prev.SubscriberCount)
	vidChange = int64(current.VideoCount) - int64(prev.VideoCount)
	viewChange = int64(current.ViewCount) - int64(prev.ViewCount)
	return
}

// 달성된 마일스톤들을 저장하고 달성 개수 반환
func (ys *Scheduler) processMilestones(ctx context.Context, channelID string, member *domain.Member, milestones []uint64, now time.Time) int {
	achieved := 0
	for _, milestone := range milestones {
		milestoneRecord := &domain.Milestone{
			ChannelID:  channelID,
			MemberName: member.Name,
			Type:       domain.MilestoneSubscribers,
			Value:      milestone,
			AchievedAt: now,
			Notified:   false,
		}

		if err := ys.statsRepo.SaveMilestone(ctx, milestoneRecord); err != nil {
			ys.logger.Error("Failed to save milestone",
				slog.String("member", member.Name),
				slog.Any("value", milestone),
				slog.Any("error", err))
		} else {
			achieved++
			ys.logger.Info("Milestone achieved",
				slog.String("member", member.Name),
				slog.Any("subscribers", milestone))
		}
	}
	return achieved
}

// 단일 채널의 통계 처리 (저장, 변경 기록, 마일스톤)
func (ys *Scheduler) processChannelStats(ctx context.Context, channelID string, member *domain.Member, currentStats *ChannelStats, now time.Time) (changesDetected, milestonesAchieved int) {
	prevStats, err := ys.statsRepo.GetLatestStats(ctx, channelID)
	if err != nil {
		ys.logger.Warn("Failed to get previous stats",
			slog.String("channel", channelID),
			slog.Any("error", err))
	}

	timestampedStats := createTimestampedStats(channelID, member, currentStats, now)

	if err := ys.statsRepo.SaveStats(ctx, timestampedStats); err != nil {
		ys.logger.Error("Failed to save stats",
			slog.String("channel", channelID),
			slog.Any("error", err))
		return 0, 0
	}

	if prevStats != nil {
		subChange, vidChange, viewChange := calculateStatsChanges(prevStats, currentStats)

		if subChange != 0 || vidChange != 0 {
			change := &domain.StatsChange{
				ChannelID:        channelID,
				MemberName:       member.Name,
				SubscriberChange: subChange,
				VideoChange:      vidChange,
				ViewChange:       viewChange,
				PreviousStats:    prevStats,
				CurrentStats:     timestampedStats,
				DetectedAt:       now,
			}

			if err := ys.statsRepo.RecordChange(ctx, change); err != nil {
				ys.logger.Error("Failed to record change",
					slog.String("member", member.Name),
					slog.Any("error", err))
			} else {
				changesDetected = 1
			}

			milestones := ys.checkMilestones(prevStats.SubscriberCount, currentStats.SubscriberCount)
			milestonesAchieved = ys.processMilestones(ctx, channelID, member, milestones, now)
		}
	}

	return changesDetected, milestonesAchieved
}

func (ys *Scheduler) fetchRecentVideosRotation(ctx context.Context, batchNum int) {
	channels := ys.getRotatingBatch(batchNum, channelsPerBatch)

	if len(channels) == 0 {
		ys.logger.Info("Skipping recent videos batch: no channels configured",
			slog.Int("batch", batchNum))
		return
	}

	ys.logger.Info("Fetching recent videos for batch",
		slog.Int("batch", batchNum),
		slog.Int("channels", len(channels)),
		slog.Int("quota_cost", len(channels)*100))

	successCount := 0
	errorCount := 0

	for _, channelID := range channels {
		videos, err := ys.youtube.GetRecentVideos(ctx, channelID, 10)
		if err != nil {
			ys.logger.Warn("Failed to fetch recent videos",
				slog.String("channel", channelID),
				slog.Any("error", err))
			errorCount++
			continue
		}

		cacheKey := "youtube:recent_videos:" + channelID
		_ = ys.cache.Set(ctx, cacheKey, videos, 24*time.Hour)

		ys.logger.Debug("Recent videos fetched",
			slog.String("channel", channelID),
			slog.Int("videos", len(videos)))

		successCount++
	}

	ys.logger.Info("Recent videos batch completed",
		slog.Int("batch", batchNum),
		slog.Int("success", successCount),
		slog.Int("errors", errorCount))
}

func (ys *Scheduler) getRotatingBatch(batchNum int, size int) []string {
	allChannels := make([]string, 0, len(ys.membersData.GetAllMembers()))
	for _, member := range ys.membersData.GetAllMembers() {
		allChannels = append(allChannels, member.ChannelID)
	}

	total := len(allChannels)
	if total == 0 || size <= 0 {
		return []string{}
	}

	start := (batchNum * size) % total
	end := start + size

	if end <= total {
		return allChannels[start:end]
	}

	batch := make([]string, 0, size)
	batch = append(batch, allChannels[start:]...)
	batch = append(batch, allChannels[0:end-total]...)
	return batch
}

// CheckMilestones: (내부용) 구독자 수 마일스톤 달성 여부를 확인하고 기록한다.
func (ys *Scheduler) checkMilestones(prevCount, currentCount uint64) []uint64 {
	milestones := []uint64{
		100000,   // 10만
		250000,   // 25만
		500000,   // 50만
		750000,   // 75만
		1000000,  // 100만
		1500000,  // 150만
		2000000,  // 200만
		2500000,  // 250만
		3000000,  // 300만
		4000000,  // 400만
		5000000,  // 500만
		10000000, // 1000만
	}

	var achieved []uint64
	for _, milestone := range milestones {
		if prevCount < milestone && currentCount >= milestone {
			achieved = append(achieved, milestone)
		}
	}

	return achieved
}

// SendMilestoneAlerts: 감지된 중요 통계 변화(마일스톤 등)에 대해 채팅방에 알림 메시지를 전송한다.
func (ys *Scheduler) SendMilestoneAlerts(ctx context.Context, sendMessage func(room, message string) error, rooms []string) error {
	changes, err := ys.statsRepo.GetUnnotifiedChanges(ctx, 50)
	if err != nil {
		return fmt.Errorf("failed to get unnotified changes: %w", err)
	}

	if len(changes) == 0 {
		return nil
	}

	ys.logger.Debug("Processing stats changes for notifications",
		slog.Int("changes", len(changes)))

	sentCount := 0
	for _, change := range changes {
		if !ys.isSignificantChange(change) {
			if err := ys.statsRepo.MarkChangeNotified(ctx, change.ChannelID, change.DetectedAt); err != nil {
				ys.logger.Warn("Failed to mark change notified",
					slog.String("channel", change.ChannelID),
					slog.Any("error", err))
			}
			continue
		}

		message := ys.formatChangeMessage(change)
		if message == "" {
			continue
		}

		for _, room := range rooms {
			if err := sendMessage(room, message); err != nil {
				ys.logger.Error("Failed to send milestone notification",
					slog.String("room", room),
					slog.String("member", change.MemberName),
					slog.Any("error", err))
				continue
			}
		}

		if err := ys.statsRepo.MarkChangeNotified(ctx, change.ChannelID, change.DetectedAt); err != nil {
			ys.logger.Warn("Failed to mark change notified",
				slog.String("channel", change.ChannelID),
				slog.Any("error", err))
		} else {
			sentCount++
		}
	}

	if sentCount > 0 {
		ys.logger.Info("Milestone notifications sent",
			slog.Int("sent", sentCount))
	}

	return nil
}

func (ys *Scheduler) isSignificantChange(change *domain.StatsChange) bool {
	if change.SubscriberChange >= 10000 {
		return true
	}

	if change.PreviousStats != nil && change.CurrentStats != nil {
		milestones := ys.checkMilestones(change.PreviousStats.SubscriberCount, change.CurrentStats.SubscriberCount)
		if len(milestones) > 0 {
			return true
		}
	}

	return false
}

func (ys *Scheduler) formatChangeMessage(change *domain.StatsChange) string {
	if change.PreviousStats == nil || change.CurrentStats == nil {
		return ""
	}

	milestones := ys.checkMilestones(change.PreviousStats.SubscriberCount, change.CurrentStats.SubscriberCount)
	if len(milestones) > 0 {
		milestone := milestones[0] // Take first milestone
		return fmt.Sprintf("🎉 %s님이 구독자 %s명을 달성했습니다!\n축하합니다! 🎊",
			change.MemberName,
			util.FormatKoreanNumber(int64(milestone)))
	}

	if change.SubscriberChange >= 10000 {
		return fmt.Sprintf("📈 %s님의 구독자가 %s명 증가했습니다!\n현재 구독자: %s명",
			change.MemberName,
			util.FormatKoreanNumber(change.SubscriberChange),
			util.FormatKoreanNumber(int64(change.CurrentStats.SubscriberCount)))
	}

	return ""
}
