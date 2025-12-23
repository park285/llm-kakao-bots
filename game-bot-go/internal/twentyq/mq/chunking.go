package mq

import (
	"github.com/park285/llm-kakao-bots/game-bot-go/internal/common/mqmsg"
	"github.com/park285/llm-kakao-bots/game-bot-go/internal/common/textutil"
	qconfig "github.com/park285/llm-kakao-bots/game-bot-go/internal/twentyq/config"
)

func emitChunkedText(chatID string, threadID *string, text string, emit func(mqmsg.OutboundMessage) error) error {
	chunks := textutil.ChunkByLines(text, qconfig.KakaoMessageMaxLength)
	if len(chunks) == 0 {
		return emit(mqmsg.NewFinal(chatID, "", threadID))
	}

	for idx, chunk := range chunks {
		isLast := idx == len(chunks)-1
		if isLast {
			if err := emit(mqmsg.NewFinal(chatID, chunk, threadID)); err != nil {
				return err
			}
			continue
		}
		if err := emit(mqmsg.NewWaiting(chatID, chunk, threadID)); err != nil {
			return err
		}
	}
	return nil
}
