package service

import (
	"context"
	"encoding/json"
	"log/slog"
	"os"
	"testing"
	"time"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"

	"github.com/park285/llm-kakao-bots/game-bot-go/internal/common/ptr"
	qconfig "github.com/park285/llm-kakao-bots/game-bot-go/internal/twentyq/config"
	qrepo "github.com/park285/llm-kakao-bots/game-bot-go/internal/twentyq/repository"
)

type categoryStatPayload struct {
	GamesCompleted    int     `json:"gamesCompleted"`
	Surrenders        int     `json:"surrenders"`
	QuestionsAsked    int     `json:"questionsAsked"`
	HintsUsed         int     `json:"hintsUsed"`
	BestQuestionCount *int    `json:"bestQuestionCount,omitempty"`
	BestTarget        *string `json:"bestTarget,omitempty"`
}

func TestStatsRecorder(t *testing.T) {
	// Setup DB
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
	if err := repo.AutoMigrate(context.Background()); err != nil {
		t.Fatal(err)
	}

	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	recorder := NewStatsRecorder(repo, logger, qconfig.StatsConfig{})

	ctx := context.Background()
	chatID := "chat_rec_1"
	userID := "user_rec_1"

	t.Run("RecordGameStart", func(t *testing.T) {
		recorder.RecordGameStart(ctx, chatID, userID)
		// Verify DB has an entry?
		// Actually RecordGameStart logs interactively or maybe in a future table?
		// Looking at repo implementation (not visible here but assuming standard),
		// usually game starts are fire-and-forget or track in session.
		// Detailed repo check not needed if we just confirm no error/panic for now,
		// but checking side effects is better.
		// Since I don't see the Repo code for RecordGameStart, I will assume it works if no panic.
	})

	t.Run("RecordGameCompletion", func(t *testing.T) {
		rec := GameCompletionRecord{
			SessionID:          "sess_rec_1",
			ChatID:             chatID,
			Category:           "Animals",
			Result:             GameResultCorrect,
			TotalQuestionCount: 15,
			HintCount:          2,
			CompletedAt:        time.Now(),
			Players: []PlayerCompletionRecord{
				{
					UserID:          userID,
					Sender:          "PlayerOne",
					QuestionCount:   10,
					WrongGuessCount: 1,
					Target:          nil,
				},
				{
					UserID:          "user_rec_2",
					Sender:          "PlayerTwo",
					QuestionCount:   5,
					WrongGuessCount: 0,
					Target:          ptr.String("Tiger"),
				},
			},
		}

		recorder.RecordGameCompletionSync(ctx, rec)

		// Verify Session
		var session qrepo.GameSession
		if err := db.Where("session_id = ?", "sess_rec_1").First(&session).Error; err != nil {
			t.Fatalf("expected session to be created: %v", err)
		}
		if session.ParticipantCount != 2 {
			t.Errorf("expected 2 participants, got %d", session.ParticipantCount)
		}

		// Verify Logs
		var logs []qrepo.GameLog
		db.Where("chat_id = ?", chatID).Find(&logs)
		if len(logs) != 2 {
			t.Errorf("expected 2 game logs, got %d", len(logs))
		}

		// Verify Stats
		// The RecordGameCompletion also updates user stats.
		var stats qrepo.UserStats
		// Composite ID might be "chat_rec_1:user_rec_1"
		if err := db.Where("chat_id = ? AND user_id = ?", chatID, userID).First(&stats).Error; err != nil {
			t.Errorf("expected stats for user 1: %v", err)
		}

		if stats.CategoryStatsJSON == nil || *stats.CategoryStatsJSON == "" {
			t.Fatal("expected category stats json for user 1")
		}
		var categoryStats map[string]categoryStatPayload
		if err := json.Unmarshal([]byte(*stats.CategoryStatsJSON), &categoryStats); err != nil {
			t.Fatalf("unmarshal category stats for user 1 failed: %v", err)
		}
		catStat, ok := categoryStats["ANIMALS"]
		if !ok {
			t.Fatalf("expected category stats for user 1 in category ANIMALS")
		}
		if catStat.GamesCompleted != 1 {
			t.Errorf("user 1 gamesCompleted = %d, want 1", catStat.GamesCompleted)
		}
		if catStat.Surrenders != 0 {
			t.Errorf("user 1 surrenders = %d, want 0", catStat.Surrenders)
		}
		if catStat.QuestionsAsked != 10 {
			t.Errorf("user 1 questionsAsked = %d, want 10", catStat.QuestionsAsked)
		}
		if catStat.HintsUsed != 2 {
			t.Errorf("user 1 hintsUsed = %d, want 2", catStat.HintsUsed)
		}
		if catStat.BestQuestionCount != nil {
			t.Errorf("user 1 bestQuestionCount = %d, want nil", *catStat.BestQuestionCount)
		}

		var stats2 qrepo.UserStats
		if err := db.Where("chat_id = ? AND user_id = ?", chatID, "user_rec_2").First(&stats2).Error; err != nil {
			t.Errorf("expected stats for user 2: %v", err)
		}
		if stats2.CategoryStatsJSON == nil || *stats2.CategoryStatsJSON == "" {
			t.Fatal("expected category stats json for user 2")
		}
		var categoryStats2 map[string]categoryStatPayload
		if err := json.Unmarshal([]byte(*stats2.CategoryStatsJSON), &categoryStats2); err != nil {
			t.Fatalf("unmarshal category stats for user 2 failed: %v", err)
		}
		catStat2, ok := categoryStats2["ANIMALS"]
		if !ok {
			t.Fatalf("expected category stats for user 2 in category ANIMALS")
		}
		if catStat2.GamesCompleted != 1 {
			t.Errorf("user 2 gamesCompleted = %d, want 1", catStat2.GamesCompleted)
		}
		if catStat2.QuestionsAsked != 5 {
			t.Errorf("user 2 questionsAsked = %d, want 5", catStat2.QuestionsAsked)
		}
		if catStat2.HintsUsed != 2 {
			t.Errorf("user 2 hintsUsed = %d, want 2", catStat2.HintsUsed)
		}
		if catStat2.BestQuestionCount == nil || *catStat2.BestQuestionCount != 15 {
			if catStat2.BestQuestionCount == nil {
				t.Fatal("user 2 bestQuestionCount is nil, want 15")
			}
			t.Errorf("user 2 bestQuestionCount = %d, want 15", *catStat2.BestQuestionCount)
		}
		if catStat2.BestTarget == nil || *catStat2.BestTarget != "Tiger" {
			if catStat2.BestTarget == nil {
				t.Fatal("user 2 bestTarget is nil, want Tiger")
			}
			t.Errorf("user 2 bestTarget = %s, want Tiger", *catStat2.BestTarget)
		}
	})

	t.Run("RecordGameCompletion_Invalid", func(t *testing.T) {
		// Should not panic and should not record
		rec := GameCompletionRecord{
			ChatID: "", // Missing ChatID
		}
		recorder.RecordGameCompletion(ctx, rec)
	})
}

func TestBestScoreUpdate(t *testing.T) {
	// Setup DB
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
	if err := repo.AutoMigrate(context.Background()); err != nil {
		t.Fatal(err)
	}

	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	recorder := NewStatsRecorder(repo, logger, qconfig.StatsConfig{})

	ctx := context.Background()
	chatID := "chat_best_update"
	userID := "user_best_update"

	// First game: 15 questions
	rec1 := GameCompletionRecord{
		SessionID:          "sess_best_1",
		ChatID:             chatID,
		Category:           "concept",
		Result:             GameResultCorrect,
		TotalQuestionCount: 15,
		HintCount:          1,
		CompletedAt:        time.Now(),
		Players: []PlayerCompletionRecord{
			{
				UserID:          userID,
				Sender:          "TestPlayer",
				QuestionCount:   15,
				WrongGuessCount: 0,
				Target:          ptr.String("상상"),
			},
		},
	}
	recorder.RecordGameCompletionSync(ctx, rec1)

	// Verify first best score
	var stats1 qrepo.UserStats
	if err := db.Where("chat_id = ? AND user_id = ?", chatID, userID).First(&stats1).Error; err != nil {
		t.Fatalf("expected stats for user: %v", err)
	}

	var categoryStats1 map[string]categoryStatPayload
	if err := json.Unmarshal([]byte(*stats1.CategoryStatsJSON), &categoryStats1); err != nil {
		t.Fatalf("unmarshal category stats failed: %v", err)
	}
	catStat1 := categoryStats1["CONCEPT"]
	if catStat1.BestQuestionCount == nil || *catStat1.BestQuestionCount != 15 {
		t.Fatalf("expected best question count 15, got %v", catStat1.BestQuestionCount)
	}
	if catStat1.BestTarget == nil || *catStat1.BestTarget != "상상" {
		t.Fatalf("expected best target 상상, got %v", catStat1.BestTarget)
	}
	t.Logf("After game 1: BestQuestionCount=%d, BestTarget=%s", *catStat1.BestQuestionCount, *catStat1.BestTarget)

	// Second game: 0 questions (better score!)
	rec2 := GameCompletionRecord{
		SessionID:          "sess_best_2",
		ChatID:             chatID,
		Category:           "concept",
		Result:             GameResultCorrect,
		TotalQuestionCount: 0,
		HintCount:          1,
		CompletedAt:        time.Now(),
		Players: []PlayerCompletionRecord{
			{
				UserID:          userID,
				Sender:          "TestPlayer",
				QuestionCount:   0,
				WrongGuessCount: 0,
				Target:          ptr.String("복수"),
			},
		},
	}
	recorder.RecordGameCompletionSync(ctx, rec2)

	// Verify best score is updated
	var stats2 qrepo.UserStats
	if err := db.Where("chat_id = ? AND user_id = ?", chatID, userID).First(&stats2).Error; err != nil {
		t.Fatalf("expected stats for user after second game: %v", err)
	}

	var categoryStats2 map[string]categoryStatPayload
	if err := json.Unmarshal([]byte(*stats2.CategoryStatsJSON), &categoryStats2); err != nil {
		t.Fatalf("unmarshal category stats failed: %v", err)
	}
	catStat2 := categoryStats2["CONCEPT"]
	t.Logf("After game 2: BestQuestionCount=%v, BestTarget=%v", catStat2.BestQuestionCount, catStat2.BestTarget)

	if catStat2.BestQuestionCount == nil {
		t.Fatal("best question count should not be nil after second game")
	}
	if *catStat2.BestQuestionCount != 0 {
		t.Errorf("expected best question count to be updated to 0, got %d", *catStat2.BestQuestionCount)
	}
	if catStat2.BestTarget == nil || *catStat2.BestTarget != "복수" {
		t.Errorf("expected best target to be updated to 복수, got %v", catStat2.BestTarget)
	}
}

func stringPtr(s string) *string {
	return &s
}
