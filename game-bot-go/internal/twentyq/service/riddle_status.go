package service

import (
	"context"
	"fmt"
	"strings"

	"github.com/park285/llm-kakao-bots/game-bot-go/internal/common/messageprovider"
	qconfig "github.com/park285/llm-kakao-bots/game-bot-go/internal/twentyq/config"
	qerrors "github.com/park285/llm-kakao-bots/game-bot-go/internal/twentyq/errors"
	qmessages "github.com/park285/llm-kakao-bots/game-bot-go/internal/twentyq/messages"
	qmodel "github.com/park285/llm-kakao-bots/game-bot-go/internal/twentyq/model"
)

// Status 는 동작을 수행한다.
func (s *RiddleService) Status(ctx context.Context, chatID string) (string, error) {
	main, hint, err := s.StatusSeparated(ctx, chatID)
	if err != nil {
		return "", err
	}
	if hint == "" {
		return main, nil
	}
	return main + "\n" + hint, nil
}

// StatusSeparated 는 동작을 수행한다.
func (s *RiddleService) StatusSeparated(ctx context.Context, chatID string) (string, string, error) {
	chatID = strings.TrimSpace(chatID)
	if chatID == "" {
		return "", "", fmt.Errorf("chat id is empty")
	}

	secret, err := s.sessionStore.GetSecret(ctx, chatID)
	if err != nil {
		return "", "", fmt.Errorf("secret get failed: %w", err)
	}
	if secret == nil {
		return "", "", qerrors.SessionNotFoundError{ChatID: chatID}
	}

	history, err := s.historyStore.Get(ctx, chatID)
	if err != nil {
		return "", "", fmt.Errorf("history get failed: %w", err)
	}

	hintCount, err := s.hintCountStore.Get(ctx, chatID)
	if err != nil {
		return "", "", fmt.Errorf("hint count get failed: %w", err)
	}

	remaining := (qconfig.MaxHintsTotal - hintCount)
	if remaining < 0 {
		remaining = 0
	}

	header := s.buildStatusHeader(secret.Category, remaining)

	wrongGuesses, err := s.wrongGuessStore.GetSessionWrongGuesses(ctx, chatID)
	if err != nil {
		return "", "", fmt.Errorf("wrong guess get failed: %w", err)
	}
	wrongLine := s.buildStatusWrongLine(wrongGuesses)

	hintLine := s.buildStatusHintLine(history)
	qnaLines := s.buildStatusQnALines(history)

	main := s.buildStatusMain(header, wrongLine, qnaLines)
	return main, hintLine, nil
}

func (s *RiddleService) buildStatusHeader(category string, remaining int) string {
	selectedCategoryKo := categoryToKorean(category)
	if selectedCategoryKo != nil {
		return s.msgProvider.Get(
			qmessages.StatusHeaderWithCategory,
			messageprovider.P("category", *selectedCategoryKo),
			messageprovider.P("remaining", remaining),
		)
	}
	return s.msgProvider.Get(qmessages.StatusHeaderNoCategory, messageprovider.P("remaining", remaining))
}

func (s *RiddleService) buildStatusHintLine(history []qmodel.QuestionHistory) string {
	for _, h := range history {
		if h.QuestionNumber < 0 {
			hintNumber := -h.QuestionNumber
			return s.msgProvider.Get(
				qmessages.StatusHintLine,
				messageprovider.P("number", hintNumber),
				messageprovider.P("content", h.Answer),
			)
		}
	}
	return ""
}

func (s *RiddleService) buildStatusQnALines(history []qmodel.QuestionHistory) []string {
	qnaLines := make([]string, 0, len(history))
	qIndex := 0
	for _, h := range history {
		if h.QuestionNumber <= 0 {
			continue
		}
		qIndex++
		numberText := fmt.Sprintf("%d", qIndex)
		if h.IsChain {
			numberText += s.msgProvider.Get(qmessages.StatusChainSuffix)
		}
		qnaLines = append(
			qnaLines,
			s.msgProvider.Get(
				qmessages.StatusQuestionAnswer,
				messageprovider.P("number", numberText),
				messageprovider.P("question", h.Question),
				messageprovider.P("answer", h.Answer),
			),
		)
	}
	return qnaLines
}

func (s *RiddleService) buildStatusWrongLine(wrongGuesses []string) string {
	if len(wrongGuesses) == 0 {
		return ""
	}
	return s.msgProvider.Get(qmessages.StatusWrongGuesses, messageprovider.P("guesses", strings.Join(wrongGuesses, ", ")))
}

func (s *RiddleService) buildStatusMain(header string, wrongLine string, qnaLines []string) string {
	parts := make([]string, 0, 3)
	if header != "" {
		parts = append(parts, header)
	}
	if wrongLine != "" {
		parts = append(parts, wrongLine)
	}
	if len(qnaLines) > 0 {
		parts = append(parts, strings.Join(qnaLines, "\n"))
	}
	return strings.Join(parts, "\n")
}
