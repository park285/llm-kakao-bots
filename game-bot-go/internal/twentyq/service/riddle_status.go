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

// Status: 현재 게임 진행 상황을 문자열 하나로 합쳐서 반환합니다.
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

// StatusSeparated: 게임 진행 상황을 메인 정보와 힌트 정보로 분리하여 반환합니다.
func (s *RiddleService) StatusSeparated(ctx context.Context, chatID string) (string, string, error) {
	main, hint, _, err := s.StatusSeparatedWithCount(ctx, chatID)
	return main, hint, err
}

// StatusSeparatedWithCount: 게임 진행 상황을 메인 정보, 힌트 정보, 질문 횟수로 분리하여 반환합니다.
// questionCount는 힌트를 제외한 실제 질문 수 (체인질문 포함).
func (s *RiddleService) StatusSeparatedWithCount(ctx context.Context, chatID string) (string, string, int, error) {
	chatID = strings.TrimSpace(chatID)
	if chatID == "" {
		return "", "", 0, fmt.Errorf("chat id is empty")
	}

	secret, err := s.sessionStore.GetSecret(ctx, chatID)
	if err != nil {
		return "", "", 0, fmt.Errorf("secret get failed: %w", err)
	}
	if secret == nil {
		return "", "", 0, qerrors.SessionNotFoundError{ChatID: chatID}
	}

	history, err := s.historyStore.Get(ctx, chatID)
	if err != nil {
		return "", "", 0, fmt.Errorf("history get failed: %w", err)
	}

	hintCount, err := s.hintCountStore.Get(ctx, chatID)
	if err != nil {
		return "", "", 0, fmt.Errorf("hint count get failed: %w", err)
	}

	remaining := (qconfig.MaxHintsTotal - hintCount)
	if remaining < 0 {
		remaining = 0
	}

	header := s.buildStatusHeader(secret.Category, remaining)

	wrongGuesses, err := s.wrongGuessStore.GetSessionWrongGuesses(ctx, chatID)
	if err != nil {
		return "", "", 0, fmt.Errorf("wrong guess get failed: %w", err)
	}
	wrongLine := s.buildStatusWrongLine(wrongGuesses)

	hintLine := s.buildStatusHintLine(history)
	qnaLines := s.buildStatusQnALines(history)

	// 마지막 힌트 이후의 질문 횟수 계산
	// 힌트가 없으면 0 반환 (힌트 라인도 없으므로 표시 안됨)
	questionsSinceHint := countQuestionsSinceLastHint(history)

	main := s.buildStatusMain(header, wrongLine, qnaLines)
	return main, hintLine, questionsSinceHint, nil
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

// countQuestionsSinceLastHint: 히스토리에서 마지막 힌트 이후에 추가된 질문 수를 계산합니다.
// 힌트가 없으면 0을 반환합니다 (힌트 라인도 없으므로 표시되지 않음).
func countQuestionsSinceLastHint(history []qmodel.QuestionHistory) int {
	// 마지막 힌트의 인덱스를 찾는다
	lastHintIndex := -1
	for i, h := range history {
		if h.QuestionNumber < 0 {
			lastHintIndex = i
		}
	}

	// 힌트가 없으면 0 반환
	if lastHintIndex < 0 {
		return 0
	}

	// 마지막 힌트 이후의 질문 수를 계산
	count := 0
	for i := lastHintIndex + 1; i < len(history); i++ {
		if history[i].QuestionNumber > 0 {
			count++
		}
	}
	return count
}
