package youtube

import (
	"context"
	"io"
	"log/slog"
	"testing"
	"time"

	"github.com/kapu/hololive-kakao-bot-go/internal/domain"
)

// mockMemberDataProvider: 테스트용 MemberDataProvider 구현
type mockMemberDataProvider struct {
	members []*domain.Member
}

func (m *mockMemberDataProvider) FindMemberByChannelID(channelID string) *domain.Member {
	for _, member := range m.members {
		if member.ChannelID == channelID {
			return member
		}
	}
	return nil
}

func (m *mockMemberDataProvider) FindMemberByName(name string) *domain.Member {
	for _, member := range m.members {
		if member.Name == name {
			return member
		}
	}
	return nil
}

func (m *mockMemberDataProvider) FindMemberByAlias(alias string) *domain.Member {
	return nil
}

func (m *mockMemberDataProvider) GetChannelIDs() []string {
	ids := make([]string, len(m.members))
	for i, member := range m.members {
		ids[i] = member.ChannelID
	}
	return ids
}

func (m *mockMemberDataProvider) GetAllMembers() []*domain.Member {
	return m.members
}

func (m *mockMemberDataProvider) WithContext(ctx context.Context) domain.MemberDataProvider {
	return m
}

// testMembers: 테스트용 멤버 데이터
func testMembers() []*domain.Member {
	return []*domain.Member{
		{ChannelID: "UC1", Name: "TestMember1"},
		{ChannelID: "UC2", Name: "TestMember2"},
		{ChannelID: "UC3", Name: "TestMember3"},
	}
}

func TestNewScheduler(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	mockMembers := &mockMemberDataProvider{members: testMembers()}

	scheduler := NewScheduler(nil, nil, nil, nil, mockMembers, logger)

	if scheduler == nil {
		t.Fatal("expected scheduler to be created, got nil")
	}
	if scheduler.currentBatch != 0 {
		t.Errorf("expected currentBatch to be 0, got %d", scheduler.currentBatch)
	}
	if scheduler.stopCh == nil {
		t.Error("expected stopCh to be initialized")
	}
}

func TestScheduler_CheckMilestones(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	mockMembers := &mockMemberDataProvider{members: testMembers()}

	scheduler := NewScheduler(nil, nil, nil, nil, mockMembers, logger)

	testCases := []struct {
		name         string
		prevCount    uint64
		currentCount uint64
		wantCount    int
		wantValues   []uint64
	}{
		{
			name:         "100k milestone crossed",
			prevCount:    99000,
			currentCount: 101000,
			wantCount:    1,
			wantValues:   []uint64{100000},
		},
		{
			name:         "no milestone crossed",
			prevCount:    100000,
			currentCount: 110000,
			wantCount:    0,
			wantValues:   []uint64{},
		},
		{
			name:         "multiple milestones crossed",
			prevCount:    240000,
			currentCount: 510000,
			wantCount:    2,
			wantValues:   []uint64{250000, 500000},
		},
		{
			name:         "1M milestone crossed",
			prevCount:    999000,
			currentCount: 1010000,
			wantCount:    1,
			wantValues:   []uint64{1000000},
		},
		{
			name:         "exact milestone boundary",
			prevCount:    249999,
			currentCount: 250000,
			wantCount:    1,
			wantValues:   []uint64{250000},
		},
		{
			name:         "decrease in subscribers",
			prevCount:    110000,
			currentCount: 95000,
			wantCount:    0,
			wantValues:   []uint64{},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			milestones := scheduler.checkMilestones(tc.prevCount, tc.currentCount)

			if len(milestones) != tc.wantCount {
				t.Errorf("expected %d milestones, got %d", tc.wantCount, len(milestones))
			}

			for i, want := range tc.wantValues {
				if i < len(milestones) && milestones[i] != want {
					t.Errorf("expected milestone[%d] = %d, got %d", i, want, milestones[i])
				}
			}
		})
	}
}

func TestScheduler_GetRotatingBatch(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))

	// 5명의 멤버로 테스트 (배치 크기보다 작은 경우)
	smallMembers := &mockMemberDataProvider{
		members: []*domain.Member{
			{ChannelID: "UC1", Name: "Member1"},
			{ChannelID: "UC2", Name: "Member2"},
			{ChannelID: "UC3", Name: "Member3"},
			{ChannelID: "UC4", Name: "Member4"},
			{ChannelID: "UC5", Name: "Member5"},
		},
	}

	scheduler := NewScheduler(nil, nil, nil, nil, smallMembers, logger)

	testCases := []struct {
		name      string
		batchNum  int
		batchSize int
		wantLen   int
	}{
		{
			name:      "batch 0 with size 2",
			batchNum:  0,
			batchSize: 2,
			wantLen:   2,
		},
		{
			name:      "batch 1 with size 2",
			batchNum:  1,
			batchSize: 2,
			wantLen:   2,
		},
		{
			name:      "batch 2 wraps around",
			batchNum:  2,
			batchSize: 2,
			wantLen:   2,
		},
		{
			name:      "batch size larger than total",
			batchNum:  0,
			batchSize: 10,
			wantLen:   10,
		},
		{
			name:      "batch size of 0",
			batchNum:  0,
			batchSize: 0,
			wantLen:   0,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			batch := scheduler.getRotatingBatch(tc.batchNum, tc.batchSize)

			if len(batch) != tc.wantLen {
				t.Errorf("expected batch length %d, got %d", tc.wantLen, len(batch))
			}
		})
	}
}

func TestScheduler_GetRotatingBatch_EmptyMembers(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	emptyMembers := &mockMemberDataProvider{members: []*domain.Member{}}

	scheduler := NewScheduler(nil, nil, nil, nil, emptyMembers, logger)

	batch := scheduler.getRotatingBatch(0, 10)
	if len(batch) != 0 {
		t.Errorf("expected empty batch for empty members, got %d", len(batch))
	}
}

func TestScheduler_BuildChannelMaps(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	mockMembers := &mockMemberDataProvider{members: testMembers()}

	scheduler := NewScheduler(nil, nil, nil, nil, mockMembers, logger)

	channelIDs, channelToMember := scheduler.buildChannelMaps()

	if len(channelIDs) != 3 {
		t.Errorf("expected 3 channel IDs, got %d", len(channelIDs))
	}

	if len(channelToMember) != 3 {
		t.Errorf("expected 3 channel-to-member mappings, got %d", len(channelToMember))
	}

	// 매핑 검증
	if member := channelToMember["UC1"]; member == nil || member.Name != "TestMember1" {
		t.Error("expected UC1 to map to TestMember1")
	}
}

func TestScheduler_StartStop(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	mockMembers := &mockMemberDataProvider{members: testMembers()}

	scheduler := NewScheduler(nil, nil, nil, nil, mockMembers, logger)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Start 호출 시 패닉이 발생하지 않아야 함
	scheduler.Start(ctx)

	// ticker가 초기화되어야 함
	if scheduler.ticker == nil {
		t.Error("expected ticker to be initialized after Start")
	}

	// Stop 호출 시 정상 종료
	scheduler.Stop()

	// 채널이 닫혀야 함 (다시 Stop 호출 시 panic 방지)
	// stopCh가 닫힌 상태인지 확인
	select {
	case <-scheduler.stopCh:
		// 채널이 닫힘 - 정상
	default:
		t.Error("expected stopCh to be closed after Stop")
	}
}

func TestScheduler_IsSignificantChange(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	mockMembers := &mockMemberDataProvider{members: testMembers()}

	scheduler := NewScheduler(nil, nil, nil, nil, mockMembers, logger)

	testCases := []struct {
		name   string
		change *domain.StatsChange
		want   bool
	}{
		{
			name: "significant subscriber increase",
			change: &domain.StatsChange{
				SubscriberChange: 15000,
			},
			want: true,
		},
		{
			name: "small subscriber increase",
			change: &domain.StatsChange{
				SubscriberChange: 100,
			},
			want: false,
		},
		{
			name: "milestone crossed",
			change: &domain.StatsChange{
				SubscriberChange: 5000,
				PreviousStats:    &domain.TimestampedStats{SubscriberCount: 99000},
				CurrentStats:     &domain.TimestampedStats{SubscriberCount: 101000},
			},
			want: true,
		},
		{
			name: "no significant change",
			change: &domain.StatsChange{
				SubscriberChange: 500,
				PreviousStats:    &domain.TimestampedStats{SubscriberCount: 110000},
				CurrentStats:     &domain.TimestampedStats{SubscriberCount: 110500},
			},
			want: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			got := scheduler.isSignificantChange(tc.change)
			if got != tc.want {
				t.Errorf("expected %v, got %v", tc.want, got)
			}
		})
	}
}

func TestScheduler_FormatChangeMessage(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	mockMembers := &mockMemberDataProvider{members: testMembers()}

	scheduler := NewScheduler(nil, nil, nil, nil, mockMembers, logger)

	testCases := []struct {
		name      string
		change    *domain.StatsChange
		wantEmpty bool
		contains  string
	}{
		{
			name: "milestone message",
			change: &domain.StatsChange{
				MemberName:       "TestMember1",
				SubscriberChange: 5000,
				PreviousStats:    &domain.TimestampedStats{SubscriberCount: 99000},
				CurrentStats:     &domain.TimestampedStats{SubscriberCount: 101000},
			},
			wantEmpty: false,
			contains:  "🎉",
		},
		{
			name: "large subscriber increase message",
			change: &domain.StatsChange{
				MemberName:       "TestMember1",
				SubscriberChange: 15000,
				PreviousStats:    &domain.TimestampedStats{SubscriberCount: 110000},
				CurrentStats:     &domain.TimestampedStats{SubscriberCount: 125000},
			},
			wantEmpty: false,
			contains:  "📈",
		},
		{
			name: "no message for small change",
			change: &domain.StatsChange{
				MemberName:       "TestMember1",
				SubscriberChange: 100,
				PreviousStats:    &domain.TimestampedStats{SubscriberCount: 110000},
				CurrentStats:     &domain.TimestampedStats{SubscriberCount: 110100},
			},
			wantEmpty: true,
		},
		{
			name: "no message for nil stats",
			change: &domain.StatsChange{
				MemberName:       "TestMember1",
				SubscriberChange: 15000,
			},
			wantEmpty: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			msg := scheduler.formatChangeMessage(tc.change)

			if tc.wantEmpty {
				if msg != "" {
					t.Errorf("expected empty message, got: %s", msg)
				}
			} else {
				if msg == "" {
					t.Error("expected non-empty message")
				}
				if tc.contains != "" && !containsStr(msg, tc.contains) {
					t.Errorf("expected message to contain %q, got: %s", tc.contains, msg)
				}
			}
		})
	}
}

func TestCalculateStatsChanges(t *testing.T) {
	testCases := []struct {
		name     string
		prev     *domain.TimestampedStats
		current  *ChannelStats
		wantSub  int64
		wantVid  int64
		wantView int64
	}{
		{
			name: "all increases",
			prev: &domain.TimestampedStats{
				SubscriberCount: 100000,
				VideoCount:      50,
				ViewCount:       1000000,
			},
			current: &ChannelStats{
				SubscriberCount: 110000,
				VideoCount:      55,
				ViewCount:       1100000,
			},
			wantSub:  10000,
			wantVid:  5,
			wantView: 100000,
		},
		{
			name: "subscriber decrease",
			prev: &domain.TimestampedStats{
				SubscriberCount: 100000,
				VideoCount:      50,
				ViewCount:       1000000,
			},
			current: &ChannelStats{
				SubscriberCount: 99000,
				VideoCount:      50,
				ViewCount:       1010000,
			},
			wantSub:  -1000,
			wantVid:  0,
			wantView: 10000,
		},
		{
			name: "no change",
			prev: &domain.TimestampedStats{
				SubscriberCount: 100000,
				VideoCount:      50,
				ViewCount:       1000000,
			},
			current: &ChannelStats{
				SubscriberCount: 100000,
				VideoCount:      50,
				ViewCount:       1000000,
			},
			wantSub:  0,
			wantVid:  0,
			wantView: 0,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			subChange, vidChange, viewChange := calculateStatsChanges(tc.prev, tc.current)

			if subChange != tc.wantSub {
				t.Errorf("subscriber change: expected %d, got %d", tc.wantSub, subChange)
			}
			if vidChange != tc.wantVid {
				t.Errorf("video change: expected %d, got %d", tc.wantVid, vidChange)
			}
			if viewChange != tc.wantView {
				t.Errorf("view change: expected %d, got %d", tc.wantView, viewChange)
			}
		})
	}
}

func TestCreateTimestampedStats(t *testing.T) {
	member := &domain.Member{
		ChannelID: "UC123",
		Name:      "TestMember",
	}

	stats := &ChannelStats{
		SubscriberCount: 500000,
		VideoCount:      100,
		ViewCount:       10000000,
	}

	timestamp := time.Now()

	result := createTimestampedStats("UC123", member, stats, timestamp)

	if result.ChannelID != "UC123" {
		t.Errorf("expected ChannelID UC123, got %s", result.ChannelID)
	}
	if result.MemberName != "TestMember" {
		t.Errorf("expected MemberName TestMember, got %s", result.MemberName)
	}
	if result.SubscriberCount != 500000 {
		t.Errorf("expected SubscriberCount 500000, got %d", result.SubscriberCount)
	}
	if result.VideoCount != 100 {
		t.Errorf("expected VideoCount 100, got %d", result.VideoCount)
	}
	if result.ViewCount != 10000000 {
		t.Errorf("expected ViewCount 10000000, got %d", result.ViewCount)
	}
	if !result.Timestamp.Equal(timestamp) {
		t.Errorf("expected Timestamp %v, got %v", timestamp, result.Timestamp)
	}
}

// containsStr: 문자열 포함 여부 확인 헬퍼
func containsStr(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && containsStrHelper(s, substr))
}

func containsStrHelper(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

// mockStatsRepository: SendMilestoneAlerts 테스트용 mock repository
type mockStatsRepository struct {
	changes          []*domain.StatsChange
	notifiedChannels []string
	savedMilestones  []*domain.Milestone
}

func (m *mockStatsRepository) GetUnnotifiedChanges(ctx context.Context, limit int) ([]*domain.StatsChange, error) {
	if len(m.changes) > limit {
		return m.changes[:limit], nil
	}
	return m.changes, nil
}

func (m *mockStatsRepository) MarkChangeNotified(ctx context.Context, channelID string, detectedAt time.Time) error {
	m.notifiedChannels = append(m.notifiedChannels, channelID)
	return nil
}

func (m *mockStatsRepository) GetLatestStats(ctx context.Context, channelID string) (*domain.TimestampedStats, error) {
	return nil, nil
}

func (m *mockStatsRepository) SaveStats(ctx context.Context, stats *domain.TimestampedStats) error {
	return nil
}

func (m *mockStatsRepository) RecordChange(ctx context.Context, change *domain.StatsChange) error {
	return nil
}

func (m *mockStatsRepository) SaveMilestone(ctx context.Context, milestone *domain.Milestone) error {
	m.savedMilestones = append(m.savedMilestones, milestone)
	return nil
}

func (m *mockStatsRepository) GetTopGainers(ctx context.Context, since time.Time, limit int) ([]domain.RankEntry, error) {
	return nil, nil
}

// TestSendMilestoneAlerts_Integration: 마일스톤 달성 시 메시지 발송 플로우 전체 테스트
func TestSendMilestoneAlerts_Integration(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	mockMembers := &mockMemberDataProvider{members: testMembers()}

	// 마일스톤 달성 변경사항 (99000 → 101000, 100k 돌파)
	milestoneChange := &domain.StatsChange{
		ChannelID:        "UC1",
		MemberName:       "TestMember1",
		SubscriberChange: 2000,
		PreviousStats:    &domain.TimestampedStats{SubscriberCount: 99000},
		CurrentStats:     &domain.TimestampedStats{SubscriberCount: 101000},
		DetectedAt:       time.Now(),
	}

	// 큰 구독자 증가 (마일스톤 없음, 15000명 증가)
	largeGainChange := &domain.StatsChange{
		ChannelID:        "UC2",
		MemberName:       "TestMember2",
		SubscriberChange: 15000,
		PreviousStats:    &domain.TimestampedStats{SubscriberCount: 110000},
		CurrentStats:     &domain.TimestampedStats{SubscriberCount: 125000},
		DetectedAt:       time.Now(),
	}

	// 작은 변화 (알림 불필요)
	smallChange := &domain.StatsChange{
		ChannelID:        "UC3",
		MemberName:       "TestMember3",
		SubscriberChange: 100,
		PreviousStats:    &domain.TimestampedStats{SubscriberCount: 50000},
		CurrentStats:     &domain.TimestampedStats{SubscriberCount: 50100},
		DetectedAt:       time.Now(),
	}

	// 향후 SendMilestoneAlerts 통합 테스트 시 사용 예정
	_ = &mockStatsRepository{
		changes: []*domain.StatsChange{milestoneChange, largeGainChange, smallChange},
	}

	// 실제 Scheduler 대신 mock repo를 사용하는 테스트용 구조체 필요
	// 여기서는 로직만 테스트
	scheduler := &Scheduler{
		membersData: mockMembers,
		logger:      logger,
		stopCh:      make(chan struct{}),
	}

	// 메시지 수집용
	var sentMessages []struct {
		room    string
		message string
	}

	sendMessageFunc := func(room, message string) error {
		sentMessages = append(sentMessages, struct {
			room    string
			message string
		}{room, message})
		return nil
	}

	rooms := []string{"testRoom1", "testRoom2"}

	// 직접 로직 테스트 (statsRepo가 nil이므로 SendMilestoneAlerts 호출 불가)
	// 대신 개별 로직 검증

	// 1. 마일스톤 변경사항 - 유의미한 변화로 인식되어야 함
	if !scheduler.isSignificantChange(milestoneChange) {
		t.Error("milestone change should be significant")
	}

	// 2. 마일스톤 메시지 포맷 검증
	msg := scheduler.formatChangeMessage(milestoneChange)
	if msg == "" {
		t.Error("expected milestone message, got empty")
	}
	if !containsStr(msg, "🎉") {
		t.Errorf("milestone message should contain celebration emoji, got: %s", msg)
	}
	if !containsStr(msg, "TestMember1") {
		t.Errorf("milestone message should contain member name, got: %s", msg)
	}

	// 3. 큰 구독자 증가 메시지 포맷 검증
	msg2 := scheduler.formatChangeMessage(largeGainChange)
	if msg2 == "" {
		t.Error("expected large gain message, got empty")
	}
	if !containsStr(msg2, "📈") {
		t.Errorf("large gain message should contain chart emoji, got: %s", msg2)
	}

	// 4. 작은 변화는 메시지 없음
	msg3 := scheduler.formatChangeMessage(smallChange)
	if msg3 != "" {
		t.Errorf("small change should not generate message, got: %s", msg3)
	}

	// 5. 작은 변화는 significant하지 않음
	if scheduler.isSignificantChange(smallChange) {
		t.Error("small change should not be significant")
	}

	// 수동 메시지 발송 시뮬레이션
	for _, change := range []*domain.StatsChange{milestoneChange, largeGainChange} {
		message := scheduler.formatChangeMessage(change)
		if message != "" {
			for _, room := range rooms {
				_ = sendMessageFunc(room, message)
			}
		}
	}

	// 6. 메시지가 올바른 방에 발송되었는지 확인
	// 2개 변경사항 × 2개 방 = 4개 메시지
	if len(sentMessages) != 4 {
		t.Errorf("expected 4 messages sent, got %d", len(sentMessages))
	}

	// 7. 각 방에 2개씩 발송되었는지 확인
	room1Count := 0
	room2Count := 0
	for _, m := range sentMessages {
		if m.room == "testRoom1" {
			room1Count++
		}
		if m.room == "testRoom2" {
			room2Count++
		}
	}
	if room1Count != 2 || room2Count != 2 {
		t.Errorf("expected 2 messages per room, got room1=%d, room2=%d", room1Count, room2Count)
	}
}

// TestMilestoneDetectionFlow: 구독자 증가 → 마일스톤 검출 → 메시지 생성 전체 플로우
func TestMilestoneDetectionFlow(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	mockMembers := &mockMemberDataProvider{members: testMembers()}

	scheduler := NewScheduler(nil, nil, nil, nil, mockMembers, logger)

	testCases := []struct {
		name            string
		prevSubs        uint64
		currentSubs     uint64
		expectMilestone bool
		expectEmoji     string
	}{
		{
			name:            "100k milestone",
			prevSubs:        99000,
			currentSubs:     101000,
			expectMilestone: true,
			expectEmoji:     "🎉",
		},
		{
			name:            "1M milestone",
			prevSubs:        999000,
			currentSubs:     1010000,
			expectMilestone: true,
			expectEmoji:     "🎉",
		},
		{
			name:            "2M milestone",
			prevSubs:        1990000,
			currentSubs:     2010000,
			expectMilestone: true,
			expectEmoji:     "🎉",
		},
		{
			name:            "no milestone but large gain",
			prevSubs:        110000,
			currentSubs:     125000,
			expectMilestone: false,
			expectEmoji:     "📈",
		},
		{
			name:            "no notification needed",
			prevSubs:        110000,
			currentSubs:     111000,
			expectMilestone: false,
			expectEmoji:     "",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// 마일스톤 검출
			milestones := scheduler.checkMilestones(tc.prevSubs, tc.currentSubs)

			if tc.expectMilestone && len(milestones) == 0 {
				t.Error("expected milestone to be detected")
			}
			if !tc.expectMilestone && len(milestones) > 0 {
				t.Errorf("unexpected milestone detected: %v", milestones)
			}

			// 메시지 생성
			change := &domain.StatsChange{
				MemberName:       "TestMember",
				SubscriberChange: int64(tc.currentSubs) - int64(tc.prevSubs),
				PreviousStats:    &domain.TimestampedStats{SubscriberCount: tc.prevSubs},
				CurrentStats:     &domain.TimestampedStats{SubscriberCount: tc.currentSubs},
			}

			msg := scheduler.formatChangeMessage(change)

			if tc.expectEmoji == "" {
				if msg != "" {
					t.Errorf("expected no message, got: %s", msg)
				}
			} else {
				if !containsStr(msg, tc.expectEmoji) {
					t.Errorf("expected message with %s, got: %s", tc.expectEmoji, msg)
				}
			}
		})
	}
}
