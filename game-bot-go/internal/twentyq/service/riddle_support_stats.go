package service

import (
	"context"
	"strings"
	"time"

	qmodel "github.com/park285/llm-kakao-bots/game-bot-go/internal/twentyq/model"
)

func (s *RiddleService) recordGameCompletionIfEnabled(
	ctx context.Context,
	chatID string,
	secret qmodel.RiddleSecret,
	result GameResult,
	answererID *string,
	history []qmodel.QuestionHistory,
	hintCount int,
	totalQuestionCount int,
	completedAt time.Time,
) {
	if s.statsRecorder == nil {
		return
	}

	answerer := ""
	if answererID != nil {
		answerer = strings.TrimSpace(*answererID)
	}

	players, err := s.playerStore.GetAll(ctx, chatID)
	if err != nil {
		s.logger.Warn("player_get_failed", "chat_id", chatID, "err", err)
		players = nil
	}

	senderByUser := make(map[string]string, len(players))
	userIDs := make([]string, 0, len(players))
	for _, p := range players {
		uid := strings.TrimSpace(p.UserID)
		if uid == "" {
			continue
		}

		if _, ok := senderByUser[uid]; !ok {
			userIDs = append(userIDs, uid)
		}

		sender := strings.TrimSpace(p.Sender)
		if sender != "" {
			senderByUser[uid] = sender
		}
	}

	if len(userIDs) == 0 && answerer != "" {
		userIDs = append(userIDs, answerer)
	}

	questionCounts := make(map[string]int, len(userIDs))
	for _, h := range history {
		if h.QuestionNumber <= 0 || h.UserID == nil {
			continue
		}

		uid := strings.TrimSpace(*h.UserID)
		if uid == "" {
			continue
		}

		questionCounts[uid]++
	}

	playerRecords := make([]PlayerCompletionRecord, 0, len(userIDs))

	// 배치로 모든 유저의 오답 수를 한 번에 조회 (N개 Redis 호출 → 1개 호출)
	wgCounts, err := s.wrongGuessStore.GetUserWrongGuessCountBatch(ctx, chatID, userIDs)
	if err != nil {
		s.logger.Warn("wrong_guess_count_batch_failed", "chat_id", chatID, "err", err)
		wgCounts = make(map[string]int)
	}

	for _, uid := range userIDs {
		var target *string
		if result == GameResultCorrect && answerer != "" && uid == answerer {
			t := secret.Target
			target = &t
		}

		playerRecords = append(playerRecords, PlayerCompletionRecord{
			UserID:          uid,
			Sender:          senderByUser[uid],
			QuestionCount:   questionCounts[uid],
			WrongGuessCount: wgCounts[uid],
			Target:          target,
		})
	}

	s.statsRecorder.RecordGameCompletion(ctx, GameCompletionRecord{
		SessionID:          "",
		ChatID:             chatID,
		Category:           strings.TrimSpace(secret.Category),
		Result:             result,
		Players:            playerRecords,
		TotalQuestionCount: totalQuestionCount,
		HintCount:          hintCount,
		CompletedAt:        completedAt,
	})
}
