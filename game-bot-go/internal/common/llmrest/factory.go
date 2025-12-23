package llmrest

import (
	commonconfig "github.com/park285/llm-kakao-bots/game-bot-go/internal/common/config"
)

// NewFromConfig 는 동작을 수행한다.
func NewFromConfig(cfg commonconfig.LlmRestConfig) (*Client, error) {
	return New(Config{
		BaseURL:          cfg.BaseURL,
		APIKey:           cfg.APIKey,
		Timeout:          cfg.Timeout,
		ConnectTimeout:   cfg.ConnectTimeout,
		HTTP2Enabled:     cfg.HTTP2Enabled,
		RetryMaxAttempts: cfg.RetryMaxAttempts,
		RetryDelay:       cfg.RetryDelay,
	})
}
