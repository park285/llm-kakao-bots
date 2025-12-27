package service

import (
	"context"
	"fmt"
	"strings"

	json "github.com/goccy/go-json"

	"github.com/park285/llm-kakao-bots/game-bot-go/internal/common/messageprovider"
	qconfig "github.com/park285/llm-kakao-bots/game-bot-go/internal/twentyq/config"
	qmessages "github.com/park285/llm-kakao-bots/game-bot-go/internal/twentyq/messages"
	qmodel "github.com/park285/llm-kakao-bots/game-bot-go/internal/twentyq/model"
)

// Start: 새로운 스무고개 게임을 시작한다. (이전 세션 있으면 재개)
func (s *RiddleService) Start(ctx context.Context, chatID string, userID string, categories []string) (string, error) {
	chatID = strings.TrimSpace(chatID)
	if chatID == "" {
		return "", fmt.Errorf("chat id is empty")
	}

	holderName := userID
	returnText := ""

	err := s.lockManager.WithLock(ctx, chatID, &holderName, func(ctx context.Context) error {
		exists, err := s.sessionStore.Exists(ctx, chatID)
		if err != nil {
			return fmt.Errorf("session exists check failed: %w", err)
		}
		if exists {
			s.logger.Info("start_session_resumed", "chat_id", chatID)
			statusText, resumeErr := s.buildResumeMessage(ctx, chatID)
			if resumeErr != nil {
				return resumeErr
			}
			returnText = statusText
			return nil
		}

		selectedKey, invalidInput := selectCategory(categories)
		s.logger.Info("start_category_selection", "chat_id", chatID, "input", categories, "selectedKey", selectedKey, "invalidInput", invalidInput)

		allCategories := s.topicSelector.Categories()
		banned, err := s.topicHistoryStore.GetBannedTopics(ctx, chatID, optionalString(selectedKey), 20, allCategories)
		if err != nil {
			return fmt.Errorf("get banned topics failed: %w", err)
		}

		var excludedCategories []string
		if len(categories) == 0 {
			excludedCategories = []string{categoryMovie}
		}

		topic, err := s.topicSelector.SelectTopic(selectedKey, banned, excludedCategories)
		if err != nil {
			return err
		}
		s.logger.Info("start_topic_selected", "chat_id", chatID, "topic_name", topic.Name, "topic_category", topic.Category)

		descriptionJSON, err := json.Marshal(topic.Details)
		if err != nil {
			return fmt.Errorf("marshal topic details failed: %w", err)
		}

		secret := qmodel.RiddleSecret{
			Target:      topic.Name,
			Category:    topic.Category,
			Intro:       s.msgProvider.Get("start.intro"),
			Description: string(descriptionJSON),
		}

		if err := s.sessionStore.SaveSecret(ctx, chatID, secret); err != nil {
			return fmt.Errorf("save secret failed: %w", err)
		}
		if err := s.categoryStore.Save(ctx, chatID, optionalString(topic.Category)); err != nil {
			return fmt.Errorf("save category failed: %w", err)
		}

		if _, err := s.restClient.CreateSession(ctx, chatID, qconfig.LlmNamespace); err != nil {
			s.logger.Warn("llm_session_create_failed", "chat_id", chatID, "err", err)
		}

		returnText = s.buildStartMessage(categoryToKorean(topic.Category), invalidInput)
		return nil
	})
	if err != nil {
		return "", fmt.Errorf("start failed: %w", err)
	}

	return returnText, nil
}

func (s *RiddleService) buildResumeMessage(ctx context.Context, chatID string) (string, error) {
	// 세션 정보 조회
	secret, err := s.sessionStore.GetSecret(ctx, chatID)
	if err != nil {
		return "", fmt.Errorf("secret get failed: %w", err)
	}

	history, err := s.historyStore.Get(ctx, chatID)
	if err != nil {
		return "", fmt.Errorf("history get failed: %w", err)
	}

	wrongGuesses, err := s.wrongGuessStore.GetSessionWrongGuesses(ctx, chatID)
	if err != nil {
		return "", fmt.Errorf("wrong guess get failed: %w", err)
	}

	// 카테고리 라인 생성
	categoryLine := ""
	if secret != nil && secret.Category != "" {
		if categoryKo := categoryToKorean(secret.Category); categoryKo != nil {
			categoryLine = s.msgProvider.Get(qmessages.StartResumeCategoryLine, messageprovider.P("category", *categoryKo))
		}
	}

	// 헤더
	header := s.msgProvider.Get(qmessages.StartResumeHeader, messageprovider.P("categoryLine", categoryLine))

	// Q&A 기록
	qnaLines := s.buildStatusQnALines(history)
	qnaSection := ""
	if len(qnaLines) > 0 {
		qnaSection = s.msgProvider.Get(qmessages.StartResumeQnAHeader) + "\n" + strings.Join(qnaLines, "\n")
	}

	// 힌트 기록
	hintSection := s.buildResumeHintSection(history)

	// 틀린 정답
	wrongSection := ""
	if len(wrongGuesses) > 0 {
		wrongSection = s.msgProvider.Get(qmessages.StatusWrongGuesses, messageprovider.P("guesses", strings.Join(wrongGuesses, ", ")))
	}

	// 조합: 헤더 > 힌트 > 틀린정답 > Q&A (최하단)
	// 500자 초과 시 Q&A가 잘리더라도 중요 정보(헤더)는 유지됨
	parts := []string{header}
	if hintSection != "" {
		parts = append(parts, hintSection)
	}
	if wrongSection != "" {
		parts = append(parts, wrongSection)
	}
	if qnaSection != "" {
		parts = append(parts, qnaSection)
	}

	return strings.Join(parts, "\n\n"), nil
}

func (s *RiddleService) buildResumeHintSection(history []qmodel.QuestionHistory) string {
	var hints []string
	for _, h := range history {
		if h.QuestionNumber < 0 {
			hintNumber := -h.QuestionNumber
			hints = append(hints, s.msgProvider.Get(
				qmessages.StatusHintLine,
				messageprovider.P("number", hintNumber),
				messageprovider.P("content", h.Answer),
			))
		}
	}
	if len(hints) == 0 {
		return ""
	}
	return s.msgProvider.Get(qmessages.StartResumeHintHeader) + "\n" + strings.Join(hints, "\n")
}

func (s *RiddleService) buildStartMessage(selectedCategoryKo *string, invalidInput bool) string {
	ready := s.msgProvider.Get(qmessages.StartReady)
	if selectedCategoryKo != nil {
		categoryText := s.msgProvider.Get(qmessages.StartCategoryPrefix, messageprovider.P("category", *selectedCategoryKo))
		ready = s.msgProvider.Get(qmessages.StartReadyWithCategory, messageprovider.P("category", categoryText))
	}
	if invalidInput {
		return s.msgProvider.Get(qmessages.StartInvalidCategoryWarning) + ready
	}
	return ready
}
