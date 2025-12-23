package service

import (
	"strings"

	qerrors "github.com/park285/llm-kakao-bots/game-bot-go/internal/twentyq/errors"
)

func pickHintText(hints []string) (string, error) {
	if len(hints) == 0 {
		return "", qerrors.HintNotAvailableError{}
	}
	hintText := strings.TrimSpace(hints[0])
	if hintText == "" {
		return "", qerrors.HintNotAvailableError{}
	}
	return hintText, nil
}
