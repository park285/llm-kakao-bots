package health

import (
	"context"
	"fmt"
	"net"
	"net/url"
	"time"

	"github.com/park285/llm-kakao-bots/mcp-llm-server-go/internal/config"
	"github.com/park285/llm-kakao-bots/mcp-llm-server-go/internal/session"
)

var startTime = time.Now()

// Component 는 상태 구성 요소다.
type Component struct {
	Status string         `json:"status"`
	Detail map[string]any `json:"detail"`
}

// Response 는 상태 응답 본문이다.
type Response struct {
	Status       string               `json:"status"`
	Components   map[string]Component `json:"components"`
	SessionStore map[string]any       `json:"session_store"`
}

// Collect 는 헬스 상태를 수집한다.
func Collect(ctx context.Context, cfg *config.Config, deepChecks bool) Response {
	components := make(map[string]Component)

	appStatus := buildAppStatus()
	components["app"] = appStatus

	sessionStoreStatus := buildSessionStoreStatus(ctx, cfg, deepChecks)
	components["session_store"] = sessionStoreStatus

	geminiStatus := buildGeminiStatus(cfg)
	components["gemini"] = geminiStatus

	overall := "ok"
	for _, component := range components {
		if component.Status != "ok" {
			overall = "degraded"
			break
		}
	}

	return Response{
		Status:       overall,
		Components:   components,
		SessionStore: sessionStoreStatus.Detail,
	}
}

func buildAppStatus() Component {
	uptimeSeconds := int(time.Since(startTime).Seconds())
	return Component{
		Status: "ok",
		Detail: map[string]any{
			"uptime_seconds": uptimeSeconds,
		},
	}
}

func buildGeminiStatus(cfg *config.Config) Component {
	apiKeyPresent := false
	defaultModel := ""
	timeoutSeconds := 0
	maxRetries := 0

	if cfg != nil {
		apiKeyPresent = cfg.Gemini.PrimaryKey() != ""
		defaultModel = cfg.Gemini.DefaultModel
		timeoutSeconds = cfg.Gemini.TimeoutSeconds
		maxRetries = cfg.Gemini.MaxRetries
	}
	status := "ok"
	if !apiKeyPresent {
		status = "degraded"
	}

	detail := map[string]any{
		"api_key_present": apiKeyPresent,
		"default_model":   defaultModel,
		"timeout_seconds": timeoutSeconds,
		"max_retries":     maxRetries,
	}

	return Component{
		Status: status,
		Detail: detail,
	}
}

func buildSessionStoreStatus(ctx context.Context, cfg *config.Config, deepChecks bool) Component {
	reachability := false
	backend := "memory"
	storeEnabled := false
	storeURL := ""
	sessionTTL := 0
	sessionCount := 0
	sessionCountErr := ""

	if cfg != nil {
		storeEnabled = cfg.SessionStore.Enabled
		storeURL = cfg.SessionStore.URL
		sessionTTL = cfg.Session.SessionTTLMinutes
	}
	if ctx == nil {
		ctx = context.Background()
	}
	if storeEnabled && deepChecks {
		checkCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), 2*time.Second)
		defer cancel()

		store, err := session.NewStore(cfg)
		if err != nil {
			sessionCountErr = err.Error()
		} else {
			defer store.Close()
			if err := store.Ping(checkCtx); err != nil {
				sessionCountErr = err.Error()
			} else {
				reachability = true
				backend = "valkey"
				count, err := store.SessionCount(checkCtx)
				if err != nil {
					sessionCountErr = err.Error()
				} else {
					sessionCount = count
				}
			}
		}
	}

	status := "ok"
	if storeEnabled && !reachability {
		status = "degraded"
	}

	detail := map[string]any{
		"store_enabled":       storeEnabled,
		"store_connected":     reachability,
		"backend":             backend,
		"session_count":       sessionCount,
		"store_url":           storeURL,
		"session_ttl_minutes": sessionTTL,
		"deep_checked":        deepChecks,
	}
	if sessionCountErr != "" {
		detail["session_count_error"] = sessionCountErr
	}

	return Component{
		Status: status,
		Detail: detail,
	}
}

func storeAddress(storeURL string) (string, error) {
	parsed, err := url.Parse(storeURL)
	if err != nil {
		return "", fmt.Errorf("parse session store url: %w", err)
	}

	host := parsed.Hostname()
	if host == "" {
		return "", fmt.Errorf("session store host missing")
	}

	port := parsed.Port()
	if port == "" {
		port = "6379"
	}

	return net.JoinHostPort(host, port), nil
}
