package service

import (
	"context"
	"fmt"
	"strings"

	cerrors "github.com/park285/llm-kakao-bots/game-bot-go/internal/common/errors"
	"github.com/park285/llm-kakao-bots/game-bot-go/internal/common/messageprovider"
	domainmodels "github.com/park285/llm-kakao-bots/game-bot-go/internal/domain/models"
	qerrors "github.com/park285/llm-kakao-bots/game-bot-go/internal/twentyq/errors"
	qmessages "github.com/park285/llm-kakao-bots/game-bot-go/internal/twentyq/messages"
	qmodel "github.com/park285/llm-kakao-bots/game-bot-go/internal/twentyq/model"
)

func (s *RiddleService) handleGuess(
	ctx context.Context,
	chatID string,
	userID string,
	sender *string,
	secret qmodel.RiddleSecret,
	guess string,
) (string, qmodel.FiveScaleKo, error) {
	guess = strings.TrimSpace(guess)
	if guess == "" {
		return "", qmodel.FiveScaleAlwaysNo, cerrors.InvalidQuestionError{Message: "empty guess"}
	}

	// 개인별 Rate Limit 체크 (1분에 1회)
	if s.guessRateLimiter != nil {
		allowed, remaining, err := s.guessRateLimiter.CheckAndSet(ctx, chatID, userID)
		if err != nil {
			s.logger.Warn("guess_rate_limit_check_failed", "chat_id", chatID, "user_id", userID, "err", err)
			// 에러 시 Rate Limit 무시하고 진행
		} else if !allowed {
			return "", qmodel.FiveScaleAlwaysNo, qerrors.GuessRateLimitError{RemainingSeconds: remaining, TotalSeconds: s.guessRateLimiter.GetLimitSeconds()}
		}
	}

	if normalizeForEquality(guess) == normalizeForEquality(secret.Target) {
		return s.handleSuccess(ctx, chatID, userID, secret), qmodel.FiveScaleAlwaysYes, nil
	}

	verifyResp, err := s.restClient.TwentyQVerifyGuess(ctx, secret.Target, guess)
	if err != nil {
		s.logger.Warn("verify_failed", "chat_id", chatID, "err", err)
		verifyResp = nil
	}

	if verifyResp != nil && verifyResp.Result != nil {
		switch strings.ToUpper(strings.TrimSpace(*verifyResp.Result)) {
		case "ACCEPT":
			return s.handleSuccess(ctx, chatID, userID, secret), qmodel.FiveScaleAlwaysYes, nil
		case "CLOSE":
			if err := s.wrongGuessStore.Add(ctx, chatID, userID, guess); err != nil {
				return "", qmodel.FiveScaleAlwaysNo, fmt.Errorf("wrong guess add failed: %w", err)
			}
			return s.msgProvider.Get(qmessages.AnswerCloseCall), qmodel.FiveScaleAlwaysNo, nil
		default:
		}
	}

	if err := s.wrongGuessStore.Add(ctx, chatID, userID, guess); err != nil {
		return "", qmodel.FiveScaleAlwaysNo, fmt.Errorf("wrong guess add failed: %w", err)
	}

	displayName := domainmodels.DisplayName(chatID, userID, sender, s.msgProvider.Get(qmessages.UserAnonymous))
	return s.msgProvider.Get(
		qmessages.AnswerWrongGuess,
		messageprovider.P("nickname", displayName),
		messageprovider.P("guess", guess),
	), qmodel.FiveScaleAlwaysNo, nil
}
