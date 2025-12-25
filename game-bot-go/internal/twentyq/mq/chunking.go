package mq

import (
	"fmt"

	"github.com/park285/llm-kakao-bots/game-bot-go/internal/common/mqmsg"
	"github.com/park285/llm-kakao-bots/game-bot-go/internal/common/textutil"
	qconfig "github.com/park285/llm-kakao-bots/game-bot-go/internal/twentyq/config"
)

func emitChunkedText(chatID string, threadID *string, text string, emit func(mqmsg.OutboundMessage) error) error {
	if err := textutil.EmitChunkedText(chatID, threadID, text, qconfig.KakaoMessageMaxLength, emit); err != nil {
		return fmt.Errorf("emit chunked text failed: %w", err)
	}
	return nil
}
