package textutil

import "github.com/park285/llm-kakao-bots/game-bot-go/internal/common/mqmsg"

// EmitChunkedText: 긴 텍스트를 maxLength 기준으로 분할하여 emit 함수로 전송합니다.
// 마지막 청크는 Final 메시지로, 나머지는 Waiting 메시지로 전송된다.
func EmitChunkedText(chatID string, threadID *string, text string, maxLength int, emit func(mqmsg.OutboundMessage) error) error {
	chunks := ChunkByLines(text, maxLength)
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
