package service

import (
	"context"
	"fmt"
	"strings"
	"time"

	qconfig "github.com/park285/llm-kakao-bots/game-bot-go/internal/twentyq/config"
	qerrors "github.com/park285/llm-kakao-bots/game-bot-go/internal/twentyq/errors"
	qmodel "github.com/park285/llm-kakao-bots/game-bot-go/internal/twentyq/model"
)

func (s *RiddleService) handleRegularQuestion(
	ctx context.Context,
	chatID string,
	userID string,
	secret qmodel.RiddleSecret,
	question string,
) (string, qmodel.FiveScaleKo, error) {
	return s.handleRegularQuestionWithFlags(ctx, chatID, userID, secret, question, false)
}

func (s *RiddleService) handleRegularQuestionWithFlags(
	ctx context.Context,
	chatID string,
	userID string,
	secret qmodel.RiddleSecret,
	question string,
	isChain bool,
) (string, qmodel.FiveScaleKo, error) {
	history, err := s.historyStore.Get(ctx, chatID)
	if err != nil {
		return "", qmodel.FiveScaleAlwaysNo, fmt.Errorf("history get failed: %w", err)
	}

	eq := normalizeForEquality(question)
	for _, h := range history {
		if h.QuestionNumber > 0 && normalizeForEquality(h.Question) == eq {
			return "", qmodel.FiveScaleAlwaysNo, qerrors.DuplicateQuestionError{}
		}
	}

	questionNumber := 0
	for _, h := range history {
		if h.QuestionNumber > 0 {
			questionNumber++
		}
	}
	questionNumber++

	details := parseDetailsOrNil(secret.Description)

	timeoutCtx, cancel := context.WithTimeout(ctx, time.Duration(qconfig.AITimeoutSeconds)*time.Second)
	defer cancel()

	resp, err := s.restClient.TwentyQAnswerQuestion(timeoutCtx, chatID, qconfig.LlmNamespace, secret.Target, secret.Category, question, details)
	if err != nil {
		return "", qmodel.FiveScaleAlwaysNo, fmt.Errorf("answer question failed: %w", err)
	}

	scale := qmodel.FiveScaleAlwaysNo
	answerToken := ""
	if resp.Scale != nil {
		if parsed, ok := qmodel.ParseFiveScaleKo(*resp.Scale); ok {
			if *parsed == qmodel.FiveScaleInvalid {
				return "", qmodel.FiveScaleAlwaysNo, qerrors.InvalidQuestionError{Message: "invalid question"}
			}
			scale = *parsed
			answerToken = qmodel.FiveScaleToken(*parsed)
		}
	}
	if strings.TrimSpace(answerToken) == "" {
		answerToken = qmodel.FiveScaleToken(qmodel.FiveScaleAlwaysNo)
		scale = qmodel.FiveScaleAlwaysNo
	}

	userIDTrimmed := strings.TrimSpace(userID)
	if userIDTrimmed == "" {
		userIDTrimmed = chatID
	}

	hItem := qmodel.QuestionHistory{
		QuestionNumber:   questionNumber,
		Question:         question,
		Answer:           answerToken,
		IsChain:          isChain,
		ThoughtSignature: resp.ThoughtSignature,
		UserID:           &userIDTrimmed,
	}
	if err := s.historyStore.Add(ctx, chatID, hItem); err != nil {
		return "", qmodel.FiveScaleAlwaysNo, fmt.Errorf("history add failed: %w", err)
	}

	return answerToken, scale, nil
}
