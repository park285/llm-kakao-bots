package mq

import (
	"log/slog"

	"github.com/park285/llm-kakao-bots/game-bot-go/internal/common/messageprovider"
	commonmq "github.com/park285/llm-kakao-bots/game-bot-go/internal/common/mq"
	"github.com/park285/llm-kakao-bots/game-bot-go/internal/common/mqmsg"
	qmessages "github.com/park285/llm-kakao-bots/game-bot-go/internal/twentyq/messages"
)

// MessageQueueNotifier: twentyq 전용 큐 알림을 위한 공통 notifier alias 입니다.
type MessageQueueNotifier = commonmq.MessageQueueNotifier

// NewMessageQueueNotifier: 새로운 MessageQueueNotifier 인스턴스를 생성합니다.
func NewMessageQueueNotifier(provider *messageprovider.Provider, commandPrefix string, logger *slog.Logger) *MessageQueueNotifier {
	cfg := commonmq.QueueNotifierConfig{
		UserAnonymousKey: qmessages.UserAnonymous,
		DefaultErrorKey:  qmessages.ErrorGeneric,

		QueueProcessingKey:     qmessages.QueueProcessing,
		QueueRetryKey:          qmessages.QueueRetry,
		QueueRetryDuplicateKey: qmessages.QueueRetryDuplicate,
		QueueRetryFailedKey:    qmessages.QueueRetryFailed,

		ProcessingStartFactory: mqmsg.NewWaiting,
		RetryFactory:           mqmsg.NewWaiting,
		DuplicateFactory:       mqmsg.NewWaiting,
		FailedFactory:          mqmsg.NewFinal,
		ErrorFactory:           mqmsg.NewFinal,
	}
	mapper := func(err error) (string, []messageprovider.Param) {
		mapping := GetErrorMapping(err, commandPrefix)
		return mapping.Key, mapping.Params
	}
	return commonmq.NewMessageQueueNotifier(provider, logger, cfg, mapper, nil)
}
