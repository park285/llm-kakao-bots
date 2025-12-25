package service

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/park285/llm-kakao-bots/game-bot-go/internal/common/messageprovider"
	qconfig "github.com/park285/llm-kakao-bots/game-bot-go/internal/twentyq/config"
	qerrors "github.com/park285/llm-kakao-bots/game-bot-go/internal/twentyq/errors"
	qmessages "github.com/park285/llm-kakao-bots/game-bot-go/internal/twentyq/messages"
)

// Surrender: 게임 포기 처리를 수행하고 정답을 공개한다.
func (s *RiddleService) Surrender(ctx context.Context, chatID string) (string, error) {
	holderName := chatID
	out := ""

	err := s.lockManager.WithLock(ctx, chatID, &holderName, func(ctx context.Context) error {
		secret, err := s.sessionStore.GetSecret(ctx, chatID)
		if err != nil {
			return fmt.Errorf("secret get failed: %w", err)
		}
		if secret == nil {
			return qerrors.SessionNotFoundError{ChatID: chatID}
		}

		categoryLine := s.buildSurrenderCategoryLine(secret.Category)

		history, err := s.historyStore.Get(ctx, chatID)
		if err != nil {
			return fmt.Errorf("history get failed: %w", err)
		}
		hintBlock := s.buildSurrenderHintBlock(history)

		out = s.msgProvider.Get(
			qmessages.SurrenderResult,
			messageprovider.P("hintBlock", hintBlock),
			messageprovider.P("target", secret.Target),
			messageprovider.P("categoryLine", categoryLine),
		)

		questionCount, hintCount := countHistoryStats(history)

		completedAt := time.Now()
		s.recordGameCompletionIfEnabled(ctx, chatID, *secret, GameResultSurrender, nil, history, hintCount, questionCount, completedAt)

		_ = s.topicHistoryStore.AddCompletedTopic(ctx, chatID, strings.TrimSpace(secret.Category), secret.Target, 20)
		s.cleanupSession(ctx, chatID)

		if _, err := s.restClient.EndSessionByChat(ctx, qconfig.LlmNamespace, chatID); err != nil {
			s.logger.Warn("llm_session_end_failed", "chat_id", chatID, "err", err)
		}

		return nil
	})
	if err != nil {
		return "", fmt.Errorf("surrender failed: %w", err)
	}
	return out, nil
}
