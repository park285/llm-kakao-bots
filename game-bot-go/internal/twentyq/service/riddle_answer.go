package service

import (
	"context"
	"fmt"
	"strings"

	qerrors "github.com/park285/llm-kakao-bots/game-bot-go/internal/twentyq/errors"
	qmodel "github.com/park285/llm-kakao-bots/game-bot-go/internal/twentyq/model"
)

// Answer: 사용자의 질문에 대한 답변을 처리하고 결과를 문자열로 반환한다 (간편 호출용).
func (s *RiddleService) Answer(ctx context.Context, chatID string, userID string, sender *string, question string) (string, error) {
	outcome, err := s.AnswerWithOutcome(ctx, chatID, userID, sender, question, false)
	if err != nil {
		return "", err
	}
	return outcome.Message, nil
}

// AnswerOutcome: 질문 처리에 대한 상세 결과 (메시지, 긍정/부정 척도 등)
type AnswerOutcome struct {
	Message         string
	Scale           qmodel.FiveScaleKo
	IsAnswerAttempt bool
}

// AnswerWithOutcome: 질문 처리 결과와 함께 답변 타입(정답 시도 여부 등)을 반환한다.
func (s *RiddleService) AnswerWithOutcome(
	ctx context.Context,
	chatID string,
	userID string,
	sender *string,
	question string,
	isChain bool,
) (AnswerOutcome, error) {
	chatID = strings.TrimSpace(chatID)
	if chatID == "" {
		return AnswerOutcome{}, fmt.Errorf("chat id is empty")
	}

	holderName := userID
	out := AnswerOutcome{}

	err := s.lockManager.WithLock(ctx, chatID, &holderName, func(ctx context.Context) error {
		secret, err := s.sessionStore.GetSecret(ctx, chatID)
		if err != nil {
			return fmt.Errorf("secret get failed: %w", err)
		}
		if secret == nil {
			return qerrors.SessionNotFoundError{ChatID: chatID}
		}

		normalized, err := s.normalizeAndGuard(ctx, chatID, question)
		if err != nil {
			return err
		}

		if guessText, ok := matchExplicitAnswer(normalized); ok {
			outcome, scale, guessErr := s.handleGuess(ctx, chatID, userID, sender, *secret, guessText)
			if guessErr != nil {
				return guessErr
			}
			out = AnswerOutcome{
				Message:         outcome,
				Scale:           scale,
				IsAnswerAttempt: true,
			}
			return nil
		}

		outcome, scale, err := s.handleRegularQuestionWithFlags(ctx, chatID, userID, *secret, normalized, isChain)
		if err != nil {
			return err
		}
		out = AnswerOutcome{
			Message:         outcome,
			Scale:           scale,
			IsAnswerAttempt: false,
		}
		return nil
	})
	if err != nil {
		return AnswerOutcome{}, fmt.Errorf("answer failed: %w", err)
	}

	return out, nil
}
