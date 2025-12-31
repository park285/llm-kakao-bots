package mq

import (
	"context"

	"github.com/park285/llm-kakao-bots/game-bot-go/internal/common/messageprovider"
	"github.com/park285/llm-kakao-bots/game-bot-go/internal/common/mqmsg"
	"github.com/park285/llm-kakao-bots/game-bot-go/internal/common/textutil"
)

// SendFinalChunked: Kakao 메시지 길이 제한을 고려하여 긴 응답을 분할 전송합니다.
// 마지막 청크는 Final로, 그 외는 Waiting으로 전송하여 UX를 유지합니다.
func SendFinalChunked(
	ctx context.Context,
	publish func(ctx context.Context, msg mqmsg.OutboundMessage) error,
	chatID string,
	text string,
	threadID *string,
	maxLength int,
) error {
	chunks := textutil.ChunkByLines(text, maxLength)
	if len(chunks) == 0 {
		return publish(ctx, mqmsg.NewFinal(chatID, "", threadID))
	}

	for idx, chunk := range chunks {
		isLast := idx == len(chunks)-1
		if isLast {
			if err := publish(ctx, mqmsg.NewFinal(chatID, chunk, threadID)); err != nil {
				return err
			}
			continue
		}
		if err := publish(ctx, mqmsg.NewWaiting(chatID, chunk, threadID)); err != nil {
			return err
		}
	}
	return nil
}

// SendWaitingFromCommand: Command.WaitingMessageKey()에 따라 대기 메시지를 전송합니다.
// 반환값이 nil이면 별도의 대기 메시지를 보내지 않습니다.
func SendWaitingFromCommand(
	ctx context.Context,
	publish func(ctx context.Context, msg mqmsg.OutboundMessage) error,
	msgProvider *messageprovider.Provider,
	chatID string,
	threadID *string,
	command interface {
		WaitingMessageKey() *string
	},
) error {
	key := command.WaitingMessageKey()
	if key == nil {
		return nil
	}
	return publish(ctx, mqmsg.NewWaiting(chatID, msgProvider.Get(*key), threadID))
}
