package service

import (
	"context"
	"fmt"
	"strings"

	qerrors "github.com/park285/llm-kakao-bots/game-bot-go/internal/twentyq/errors"
	qmodel "github.com/park285/llm-kakao-bots/game-bot-go/internal/twentyq/model"
)

// Answer 는 동작을 수행한다.
func (s *RiddleService) Answer(ctx context.Context, chatID string, userID string, sender *string, question string) (string, error) {
	outcome, err := s.AnswerWithOutcome(ctx, chatID, userID, sender, question, false)
	if err != nil {
		return "", err
	}
	return outcome.Message, nil
}

// AnswerOutcome 는 타입이다.
type AnswerOutcome struct {
	Message         string
	Scale           qmodel.FiveScaleKo
	IsAnswerAttempt bool
}

// AnswerWithOutcome 는 동작을 수행한다.
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
