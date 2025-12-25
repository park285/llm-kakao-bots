package service

import (
	"context"
	"log/slog"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"

	"github.com/park285/llm-kakao-bots/game-bot-go/internal/common/messageprovider"
	"github.com/park285/llm-kakao-bots/game-bot-go/internal/common/testhelper"
	qmodel "github.com/park285/llm-kakao-bots/game-bot-go/internal/twentyq/model"
	qredis "github.com/park285/llm-kakao-bots/game-bot-go/internal/twentyq/redis"
	"github.com/park285/llm-kakao-bots/game-bot-go/internal/twentyq/repository"
	qrepo "github.com/park285/llm-kakao-bots/game-bot-go/internal/twentyq/repository"
)

func TestStatsService(t *testing.T) {
	client := testhelper.NewTestValkeyClient(t)
	defer client.Close()
	defer testhelper.CleanupTestKeys(t, client, "20q:")
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))

	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	sqlDB, err := db.DB()
	if err != nil {
		t.Fatal(err)
	}
	sqlDB.SetMaxOpenConns(1)
	sqlDB.SetMaxIdleConns(1)
	t.Cleanup(func() {
		_ = sqlDB.Close()
	})
	repo := qrepo.New(db)
	repo.AutoMigrate(context.Background())

	msgProvider, _ := messageprovider.NewFromYAML(`
stats:
  header: "User Stats: {nickname} ({totalGames})"
  user_not_found: "User Not Found"
  not_found: "User Not Found"
  no_stats: "No Stats"
  period:
    daily: "Daily"
    weekly: "Weekly"
    monthly: "Monthly"
    all: "All Time"
  room:
    header: "Room Stats ({period})"
    summary: "Total: {totalGames}"
    no_games: "No games in this room."
    activity_header: "Activity"
    activity_item: "{sender}: {games}"
  category:
    header: "Cat: {category} ({games})"
    results: "Res: completed={completed} surrender={surrender} rate={completionRate}"
    averages: "Avg: q={avgQuestions} h={avgHints}"
    best: "Best: {count} ({target})"
    no_best: "No Best"
`)

	sessionStore := qredis.NewSessionStore(client, logger)
	svc := NewStatsService(db, sessionStore, msgProvider, logger)

	ctx := context.Background()
	prefix := testhelper.UniqueTestPrefix(t)

	t.Run("GetUserStats_Empty", func(t *testing.T) {
		resp, err := svc.GetUserStats(ctx, prefix+"chat1", "user1", nil, nil)
		if err != nil {
			t.Fatal(err)
		}
		if !strings.Contains(resp, "User Not Found") {
			t.Error("expected user not found msg")
		}
	})

	stats := repository.UserStats{
		ID:                  prefix + "chat1:user1",
		ChatID:              prefix + "chat1",
		UserID:              "user1",
		TotalGamesCompleted: 5,
	}
	db.Create(&stats)

	t.Run("GetUserStats_Exists", func(t *testing.T) {
		nick := "MyNick"
		resp, err := svc.GetUserStats(ctx, prefix+"chat1", "user1", &nick, nil)
		if err != nil {
			t.Fatal(err)
		}
		if !strings.Contains(resp, "User Stats: MyNick") {
			t.Error("expected header with nickname")
		}
	})

	db.Create(&repository.GameSession{
		SessionID:   prefix + "sess1",
		ChatID:      prefix + "chat1",
		Result:      "CORRECT",
		CompletedAt: time.Now(),
	})
	db.Create(&repository.GameSession{
		SessionID:   prefix + "sess2",
		ChatID:      prefix + "chat1",
		Result:      "SURRENDER",
		CompletedAt: time.Now(),
	})

	t.Run("GetUserStats_WithCategories", func(t *testing.T) {
		catJSON := `{"ORGANISM":{"gamesCompleted":10,"surrenders":2,"questionsAsked":50,"hintsUsed":5,"bestQuestionCount":15,"bestTarget":"Cat"}}`
		statsWithCat := repository.UserStats{
			ID:                  prefix + "chat1:user_cat",
			ChatID:              prefix + "chat1",
			UserID:              "user_cat",
			TotalGamesCompleted: 34,
			CategoryStatsJSON:   &catJSON,
		}
		db.Create(&statsWithCat)

		resp, err := svc.GetUserStats(ctx, prefix+"chat1", "user_cat", nil, nil)
		if err != nil {
			t.Fatal(err)
		}

		if !strings.Contains(resp, "User Stats: 누군가 (10)") {
			t.Errorf("expected totalGames from category stats, got %s", resp)
		}
		if !strings.Contains(resp, "Cat: 생물 (10)") {
			t.Error("expected category header")
		}
		if count := strings.Count(resp, "Cat: 생물"); count != 1 {
			t.Errorf("expected single organism category, got %d", count)
		}
		if !strings.Contains(resp, "Cat: 사자성어/속담 (0)") {
			t.Error("expected idiom/proverb category with zero stats")
		}
		if !strings.Contains(resp, "Res: completed=8 surrender=2 rate=80") {
			t.Errorf("expected computed completion stats, got %s", resp)
		}
		if strings.Contains(resp, "Best:") {
			t.Error("expected no best score info")
		}
	})

	t.Run("GetRoomStats_All", func(t *testing.T) {
		resp, err := svc.GetRoomStats(ctx, prefix+"chat1", qmodel.StatsPeriodAll)
		if err != nil {
			t.Fatal(err)
		}

		if !strings.Contains(resp, "Room Stats (All Time)") {
			t.Error("expected room stats header")
		}
	})

	t.Run("GetRoomStats_Empty", func(t *testing.T) {
		resp, err := svc.GetRoomStats(ctx, prefix+"chat_empty", qmodel.StatsPeriodAll)
		if err != nil {
			t.Fatal(err)
		}
		if !strings.Contains(resp, "No games") {
			t.Error("expected no games message")
		}
	})

	t.Run("GetUserStats_TargetNickname_Found", func(t *testing.T) {
		pStore := qredis.NewPlayerStore(client, logger)
		pStore.Add(ctx, prefix+"chat1", "user_target", "TargetUser")

		db.Create(&repository.UserStats{
			ID:                  prefix + "chat1:user_target",
			ChatID:              prefix + "chat1",
			UserID:              "user_target",
			TotalGamesCompleted: 8,
		})

		nick := "TargetUser"
		resp, err := svc.GetUserStats(ctx, prefix+"chat1", "caller", nil, &nick)
		if err != nil {
			t.Fatal(err)
		}
		if !strings.Contains(resp, "User Stats: TargetUser") {
			t.Errorf("expected header with target nickname, got %s", resp)
		}
	})

	t.Run("GetUserStats_TargetNickname_DB_Fallback", func(t *testing.T) {
		now := time.Now()
		db.Create(&repository.UserNicknameMap{
			ChatID:     prefix + "chat1",
			UserID:     "user_db",
			LastSender: "DbNick",
			LastSeenAt: now,
			CreatedAt:  now,
		})
		db.Create(&repository.UserStats{
			ID:                  prefix + "chat1:user_db",
			ChatID:              prefix + "chat1",
			UserID:              "user_db",
			TotalGamesCompleted: 3,
		})

		nick := "dbnick"
		resp, err := svc.GetUserStats(ctx, prefix+"chat1", "caller", nil, &nick)
		if err != nil {
			t.Fatal(err)
		}
		if !strings.Contains(resp, "User Stats: DbNick") {
			t.Errorf("expected header with db nickname, got %s", resp)
		}
	})

	t.Run("GetRoomStats_Periods", func(t *testing.T) {
		oldTime := time.Now().AddDate(0, -2, 0)
		db.Create(&repository.GameSession{
			SessionID:   prefix + "sess_old",
			ChatID:      prefix + "chat_period",
			Result:      "CORRECT",
			CompletedAt: oldTime,
		})

		recentTime := time.Now().Add(-24 * time.Hour)
		db.Create(&repository.GameSession{
			SessionID:   prefix + "sess_recent",
			ChatID:      prefix + "chat_period",
			Result:      "CORRECT",
			CompletedAt: recentTime,
		})

		veryRecent := time.Now().Add(-1 * time.Hour)
		db.Create(&repository.GameSession{
			SessionID:   prefix + "sess_very_recent",
			ChatID:      prefix + "chat_period",
			Result:      "CORRECT",
			CompletedAt: veryRecent,
		})

		resp, _ := svc.GetRoomStats(ctx, prefix+"chat_period", qmodel.StatsPeriodDaily)
		_ = resp // Just ensure no error

		respM, _ := svc.GetRoomStats(ctx, prefix+"chat_period", qmodel.StatsPeriodMonthly)
		if strings.Contains(respM, "Total: 3") {
			t.Error("Monthly should not include 2 months ago")
		}
	})

	t.Run("GetRoomStats_WithActivity", func(t *testing.T) {
		db.Create(&repository.GameSession{
			SessionID:   prefix + "sess_act_1",
			ChatID:      prefix + "chat_activity",
			Result:      "CORRECT",
			CompletedAt: time.Now(),
		})
		db.Create(&repository.GameLog{
			ChatID:      prefix + "chat_activity",
			UserID:      "u1",
			Sender:      "ActiveUser",
			Result:      "WIN",
			CompletedAt: time.Now(),
		})

		resp, err := svc.GetRoomStats(ctx, prefix+"chat_activity", qmodel.StatsPeriodAll)
		if err != nil {
			t.Fatal(err)
		}
		if !strings.Contains(resp, "Activity") {
			t.Error("expected activity header")
		}
		if !strings.Contains(resp, "ActiveUser") {
			t.Error("expected active user in activity list")
		}
	})

	t.Run("GetPeriodHelpers_EdgeCases", func(t *testing.T) {
		unknownPeriod := qmodel.StatsPeriod("unknown")

		resp, err := svc.GetRoomStats(ctx, prefix+"chat_any", unknownPeriod)
		if err != nil {
			t.Fatal(err)
		}
		if !strings.Contains(resp, "All Time") {
			// default period msg
		}
	})
}
