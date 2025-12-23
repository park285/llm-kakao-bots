package mq

import (
	"strings"

	"github.com/park285/llm-kakao-bots/game-bot-go/internal/common/messageprovider"
	tsconfig "github.com/park285/llm-kakao-bots/game-bot-go/internal/turtlesoup/config"
	tsmessages "github.com/park285/llm-kakao-bots/game-bot-go/internal/turtlesoup/messages"
	tsmodel "github.com/park285/llm-kakao-bots/game-bot-go/internal/turtlesoup/model"
)

// MessageBuilder 는 타입이다.
type MessageBuilder struct {
	provider *messageprovider.Provider
}

// NewMessageBuilder 는 동작을 수행한다.
func NewMessageBuilder(provider *messageprovider.Provider) *MessageBuilder {
	return &MessageBuilder{provider: provider}
}

// BuildStatusHeader 는 동작을 수행한다.
func (b *MessageBuilder) BuildStatusHeader(state tsmodel.GameState) string {
	return b.provider.Get(
		tsmessages.AnswerHistoryHeader,
		messageprovider.P("questionCount", state.QuestionCount),
		messageprovider.P("hintCount", state.HintsUsed),
		messageprovider.P("maxHints", tsconfig.GameMaxHints),
	)
}

// BuildSummary 는 동작을 수행한다.
func (b *MessageBuilder) BuildSummary(history []tsmodel.HistoryEntry) string {
	if len(history) == 0 {
		return b.provider.Get(tsmessages.SummaryEmpty)
	}

	header := b.provider.Get(tsmessages.SummaryHeader, messageprovider.P("count", len(history)))
	lines := make([]string, 0, len(history))
	for i, item := range history {
		lines = append(lines, b.provider.Get(
			tsmessages.SummaryItem,
			messageprovider.P("number", i+1),
			messageprovider.P("question", item.Question),
			messageprovider.P("answer", item.Answer),
		))
	}
	return header + "\n" + strings.Join(lines, "\n")
}

// BuildHintBlock 는 동작을 수행한다.
func (b *MessageBuilder) BuildHintBlock(hints []string) string {
	if len(hints) == 0 {
		return b.provider.Get(tsmessages.HintSectionNone)
	}

	lines := make([]string, 0, len(hints))
	for i, hint := range hints {
		lines = append(lines, b.provider.Get(
			tsmessages.HintItem,
			messageprovider.P("number", i+1),
			messageprovider.P("content", hint),
		))
	}

	return b.provider.Get(
		tsmessages.HintSectionUsed,
		messageprovider.P("hintCount", len(hints)),
		messageprovider.P("hintList", strings.Join(lines, "\n")),
	)
}
