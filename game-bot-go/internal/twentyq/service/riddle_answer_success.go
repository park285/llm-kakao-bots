package service

import (
	"context"
	"strings"
	"time"

	"github.com/park285/llm-kakao-bots/game-bot-go/internal/common/messageprovider"
	qconfig "github.com/park285/llm-kakao-bots/game-bot-go/internal/twentyq/config"
	qmessages "github.com/park285/llm-kakao-bots/game-bot-go/internal/twentyq/messages"
	qmodel "github.com/park285/llm-kakao-bots/game-bot-go/internal/twentyq/model"
)

func (s *RiddleService) handleSuccess(ctx context.Context, chatID string, answererID string, secret qmodel.RiddleSecret) string {
	history, err := s.historyStore.Get(ctx, chatID)
	if err != nil {
		s.logger.Warn("history_get_failed", "chat_id", chatID, "err", err)
		history = nil
	}

	questionCount := 0
	hints := make([]qmodel.QuestionHistory, 0, 1)
	for _, h := range history {
		if h.QuestionNumber > 0 {
			questionCount++
		}
		if h.QuestionNumber < 0 {
			hints = append(hints, h)
		}
	}

	hintCount, err := s.hintCountStore.Get(ctx, chatID)
	if err != nil {
		s.logger.Warn("hint_count_get_failed", "chat_id", chatID, "err", err)
		hintCount = 0
	}

	wrongGuesses, err := s.wrongGuessStore.GetSessionWrongGuesses(ctx, chatID)
	if err != nil {
		s.logger.Warn("wrong_guess_get_failed", "chat_id", chatID, "err", err)
		wrongGuesses = nil
	}

	wrongGuessBlock := ""
	if len(wrongGuesses) > 0 {
		wrongGuessBlock = s.msgProvider.Get(qmessages.AnswerWrongGuessSection, messageprovider.P("wrongGuesses", strings.Join(wrongGuesses, ", ")))
	}

	var hintBlock string
	if len(hints) > 0 {
		hintLines := make([]string, 0, len(hints))
		for _, h := range hints {
			hintLines = append(hintLines, s.msgProvider.Get(qmessages.AnswerHintItem, messageprovider.P("question", h.Question), messageprovider.P("answer", h.Answer)))
		}
		hintBlock = s.msgProvider.Get(
			qmessages.AnswerHintSectionUsed,
			messageprovider.P("hintCount", len(hints)),
			messageprovider.P("hintList", strings.Join(hintLines, "\n")),
		)
	} else {
		hintBlock = s.msgProvider.Get(qmessages.AnswerHintSectionNone)
	}

	successMessage := s.msgProvider.Get(
		qmessages.AnswerSuccess,
		messageprovider.P("target", secret.Target),
		messageprovider.P("questionCount", questionCount),
		messageprovider.P("hintCount", hintCount),
		messageprovider.P("maxHints", qconfig.MaxHintsTotal),
		messageprovider.P("wrongGuessBlock", wrongGuessBlock),
		messageprovider.P("hintBlock", hintBlock),
	)

	completedAt := time.Now()
	s.recordGameCompletionIfEnabled(ctx, chatID, secret, GameResultCorrect, &answererID, history, hintCount, questionCount, completedAt)

	categoryKey := strings.TrimSpace(secret.Category)
	_ = s.topicHistoryStore.AddCompletedTopic(ctx, chatID, categoryKey, secret.Target, 20)
	s.cleanupSession(ctx, chatID)

	if _, err := s.restClient.EndSessionByChat(ctx, qconfig.LlmNamespace, chatID); err != nil {
		s.logger.Warn("llm_session_end_failed", "chat_id", chatID, "err", err)
	}

	return successMessage
}
