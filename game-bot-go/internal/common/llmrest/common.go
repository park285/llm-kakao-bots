package llmrest

import (
	"context"
	"fmt"
	"net/url"
	"strings"
)

// GetModelConfig: LLM 모델 설정 정보를 조회합니다.
func (c *Client) GetModelConfig(ctx context.Context) (*ModelConfigResponse, error) {
	var out ModelConfigResponse
	if err := c.Get(ctx, "/health/models", &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// CreateSession: 새로운 세션을 생성합니다.
func (c *Client) CreateSession(ctx context.Context) (*SessionCreateResponse, error) {
	var out SessionCreateResponse
	if err := c.Post(ctx, "/api/sessions", SessionCreateRequest{}, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// EndSession: 세션 ID로 세션을 종료합니다.
func (c *Client) EndSession(ctx context.Context, sessionID string) (*SessionEndResponse, error) {
	escaped := url.PathEscape(strings.TrimSpace(sessionID))
	if escaped == "" {
		return nil, fmt.Errorf("invalid session id: %q", sessionID)
	}

	var out SessionEndResponse
	if err := c.Delete(ctx, "/api/sessions/"+escaped, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// EndSessionByChat: 채팅 ID로 세션을 종료합니다.
func (c *Client) EndSessionByChat(ctx context.Context, namespace string, chatID string) (*SessionEndResponse, error) {
	sessionID := strings.TrimSpace(namespace) + ":" + strings.TrimSpace(chatID)
	return c.EndSession(ctx, sessionID)
}

// GuardIsMalicious: 입력 텍스트의 악의적 체크 여부를 확인합니다.
func (c *Client) GuardIsMalicious(ctx context.Context, text string) (bool, error) {
	var out GuardMaliciousResponse
	if err := c.Post(ctx, "/api/guard/checks", GuardRequest{InputText: text}, &out); err != nil {
		return false, err
	}
	return out.Malicious, nil
}

// GetUsage: 메모리 기반 사용량을 조회합니다.
func (c *Client) GetUsage(ctx context.Context, headers map[string]string) (*UsageResponse, error) {
	var out UsageResponse
	if err := c.GetWithHeaders(ctx, "/api/llm/usage", headers, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// GetTotalUsage: 전체 누적 사용량을 조회합니다.
func (c *Client) GetTotalUsage(ctx context.Context, headers map[string]string) (*UsageResponse, error) {
	var out UsageResponse
	if err := c.GetWithHeaders(ctx, "/api/llm/usage/total", headers, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// GetDailyUsage: 일별 사용량을 조회합니다.
func (c *Client) GetDailyUsage(ctx context.Context, headers map[string]string) (*DailyUsageResponse, error) {
	var out DailyUsageResponse
	if err := c.GetWithHeaders(ctx, "/api/usage/daily", headers, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// GetRecentUsage: 지정된 일수 내 사용량을 조회합니다.
func (c *Client) GetRecentUsage(ctx context.Context, days int, headers map[string]string) (*UsageListResponse, error) {
	var out UsageListResponse
	if err := c.GetWithHeaders(ctx, fmt.Sprintf("/api/usage/recent?days=%d", days), headers, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// GetUsageTotalFromDB: DB에서 지정된 일수 내 누적 사용량을 조회합니다.
func (c *Client) GetUsageTotalFromDB(ctx context.Context, days int, headers map[string]string) (*UsageResponse, error) {
	var out UsageResponse
	if err := c.GetWithHeaders(ctx, fmt.Sprintf("/api/usage/total?days=%d", days), headers, &out); err != nil {
		return nil, err
	}
	return &out, nil
}
