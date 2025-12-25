package service

import (
	"context"
	"fmt"
	"strings"

	cerrors "github.com/park285/llm-kakao-bots/game-bot-go/internal/common/errors"
)

func (s *RiddleService) normalizeAndGuard(ctx context.Context, _ string, question string) (string, error) {
	question = strings.TrimSpace(question)
	if question == "" {
		return "", cerrors.InvalidQuestionError{Message: "empty question"}
	}

	malicious, err := s.restClient.GuardIsMalicious(ctx, question)
	if err != nil {
		return "", fmt.Errorf("guard check failed: %w", err)
	}
	if malicious {
		return "", cerrors.InvalidQuestionError{Message: "guard blocked"}
	}
	return question, nil
}
