package mq

import (
	"log/slog"
	"time"

	"github.com/valkey-io/valkey-go"
)

// NewBotStreamConsumer: 봇 애플리케이션용 스트림 소비자 인스턴스를 생성합니다.
func NewBotStreamConsumer(
	client valkey.Client,
	logger *slog.Logger,
	stream string,
	group string,
	name string,
	batchSize int64,
	block time.Duration,
	concurrency int,
	resetGroupOnStartup bool,
) *StreamConsumer {
	return NewStreamConsumer(client, logger, StreamConsumerConfig{
		Stream:              stream,
		Group:               group,
		Name:                name,
		BatchSize:           batchSize,
		Block:               block,
		Concurrency:         concurrency,
		AckOnError:          true,
		ResetGroupOnStartup: resetGroupOnStartup,
	})
}

// NewBotReplyPublisher: 봇 애플리케이션용 응답 발행자 인스턴스를 생성합니다.
func NewBotReplyPublisher(
	client valkey.Client,
	logger *slog.Logger,
	stream string,
	maxLen int64,
) *ReplyPublisher {
	streamPublisher := NewStreamPublisher(client, logger, StreamPublisherConfig{
		Stream: stream,
		MaxLen: maxLen,
	})
	return NewReplyPublisher(streamPublisher)
}
