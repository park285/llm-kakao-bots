package llmrest

import (
	commonconfig "github.com/park285/llm-kakao-bots/game-bot-go/internal/common/config"
)

// NewFromConfig: 설정 객체로부터 새로운 Client 인스턴스를 생성합니다.
func NewFromConfig(cfg commonconfig.LlmConfig) (*Client, error) {
	return New(Config{
		BaseURL:          cfg.BaseURL,
		RequireGRPC:      cfg.RequireGRPC,
		APIKey:           cfg.APIKey,
		Timeout:          cfg.Timeout,
		ConnectTimeout:   cfg.ConnectTimeout,
		HTTP2Enabled:     cfg.HTTP2Enabled,
		RetryMaxAttempts: cfg.RetryMaxAttempts,
		RetryDelay:       cfg.RetryDelay,
	})
}
