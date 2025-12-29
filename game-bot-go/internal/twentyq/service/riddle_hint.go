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

// GenerateHint: LLM을 통해 힌트를 생성하고 저장합니다. 힌트 횟수를 차감합니다.
func (s *RiddleService) GenerateHint(ctx context.Context, chatID string) (string, error) {
	chatID = strings.TrimSpace(chatID)
	if chatID == "" {
		return "", fmt.Errorf("chat id is empty")
	}

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

		hintCount, err := s.hintCountStore.Get(ctx, chatID)
		if err != nil {
			return fmt.Errorf("hint count get failed: %w", err)
		}
		if hintCount >= qconfig.MaxHintsTotal {
			return qerrors.HintLimitExceededError{MaxHints: qconfig.MaxHintsTotal, HintCount: hintCount, Remaining: 0}
		}

		details := parseDetailsOrNil(secret.Description)
		hintsResp, err := s.restClient.TwentyQGenerateHints(ctx, secret.Target, secret.Category, details)
		if err != nil {
			return fmt.Errorf("generate hints failed: %w", err)
		}

		nextHintCount, err := s.hintCountStore.Increment(ctx, chatID)
		if err != nil {
			return fmt.Errorf("hint count increment failed: %w", err)
		}

		hintNumber := nextHintCount
		hintText, err := pickHintText(hintsResp.Hints)
		if err != nil {
			return err
		}

		historyItem := qmodel.QuestionHistory{
			QuestionNumber:   -hintNumber,
			Question:         fmt.Sprintf("힌트 #%d", hintNumber),
			Answer:           hintText,
			IsChain:          false,
			ThoughtSignature: hintsResp.ThoughtSignature,
			UserID:           nil,
		}
		if err := s.historyStore.Add(ctx, chatID, historyItem); err != nil {
			return fmt.Errorf("history add failed: %w", err)
		}

		out = s.msgProvider.Get(
			qmessages.HintGenerated,
			messageprovider.P("hintNumber", hintNumber),
			messageprovider.P("content", hintText),
		)
		return nil
	})
	if err != nil {
		return "", fmt.Errorf("generate hint failed: %w", err)
	}

	return out, nil
}

// CanGenerateHint: 현재 힌트를 생성할 수 있는 상태인지(횟수 제한 등) 확인합니다.
func (s *RiddleService) CanGenerateHint(ctx context.Context, chatID string) (bool, error) {
	chatID = strings.TrimSpace(chatID)
	if chatID == "" {
		return false, fmt.Errorf("chat id is empty")
	}

	secret, err := s.sessionStore.GetSecret(ctx, chatID)
	if err != nil {
		return false, fmt.Errorf("secret get failed: %w", err)
	}
	if secret == nil {
		return false, qerrors.SessionNotFoundError{ChatID: chatID}
	}

	hintCount, err := s.hintCountStore.Get(ctx, chatID)
	if err != nil {
		return false, fmt.Errorf("hint count get failed: %w", err)
	}

	return hintCount < qconfig.MaxHintsTotal, nil
}
