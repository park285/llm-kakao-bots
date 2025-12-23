package bot

import (
	"testing"
	"time"

	"github.com/kapu/hololive-kakao-bot-go/internal/domain"
)

func TestGroupAlarmNotifications_GroupByScheduledTime(t *testing.T) {
	now := time.Now()
	scheduled := now.Add(5 * time.Minute).Truncate(time.Minute)
	scheduledVariant := scheduled.Add(30 * time.Second)

	notif1 := &domain.AlarmNotification{
		RoomID:       "room-1",
		MinutesUntil: 5,
		Stream: &domain.Stream{
			ID:             "stream-1",
			Title:          "Stream 1",
			Status:         domain.StreamStatusUpcoming,
			StartScheduled: &scheduled,
		},
	}

	notif2 := &domain.AlarmNotification{
		RoomID:       "room-1",
		MinutesUntil: 4,
		Stream: &domain.Stream{
			ID:             "stream-2",
			Title:          "Stream 2",
			Status:         domain.StreamStatusUpcoming,
			StartScheduled: &scheduledVariant,
		},
	}

	groups := groupAlarmNotifications([]*domain.AlarmNotification{notif1, notif2})
	if len(groups) != 1 {
		t.Fatalf("expected 1 group, got %d", len(groups))
	}

	group := groups[0]
	if len(group.notifications) != 2 {
		t.Fatalf("expected 2 notifications in group, got %d", len(group.notifications))
	}

	if group.minutesUntil != 4 {
		t.Fatalf("expected minutesUntil to be 4, got %d", group.minutesUntil)
	}
}

func TestGroupAlarmNotifications_FallbackByMinutes(t *testing.T) {
	notif1 := &domain.AlarmNotification{
		RoomID:       "room-1",
		MinutesUntil: 5,
	}

	notif2 := &domain.AlarmNotification{
		RoomID:       "room-1",
		MinutesUntil: 3,
	}

	groups := groupAlarmNotifications([]*domain.AlarmNotification{notif1, notif2})
	if len(groups) != 2 {
		t.Fatalf("expected 2 groups, got %d", len(groups))
	}

	if groups[0].minutesUntil == groups[1].minutesUntil {
		t.Fatalf("expected different minutesUntil values per group when schedules are missing")
	}
}
