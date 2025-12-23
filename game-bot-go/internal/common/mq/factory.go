package mq

import (
	"log/slog"
	"time"

	"github.com/valkey-io/valkey-go"
)

// NewBotStreamConsumer 는 동작을 수행한다.
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

// NewBotReplyPublisher 는 동작을 수행한다.
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
