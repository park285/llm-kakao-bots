package health

import (
	"context"
	"fmt"
	"net"
	"net/url"
	"runtime"
	"sync"
	"time"

	"github.com/park285/llm-kakao-bots/mcp-llm-server-go/internal/config"
	"github.com/park285/llm-kakao-bots/mcp-llm-server-go/internal/session"
)

var (
	startTime = time.Now()
	version   = "dev"
	initOnce  sync.Once
)

// Init: 서비스 시작 시 호출 (버전 정보 설정)
func Init(v string) {
	initOnce.Do(func() {
		if v != "" {
			version = v
		}
	})
}

// Component: 상태 구성 요소입니다.
type Component struct {
	Status string         `json:"status"`
	Detail map[string]any `json:"detail"`
}

// Response: 상태 응답 본문입니다.
type Response struct {
	Status       string               `json:"status"`
	Version      string               `json:"version"`
	Uptime       string               `json:"uptime"`
	Goroutines   int                  `json:"goroutines"`
	Components   map[string]Component `json:"components"`
	SessionStore map[string]any       `json:"session_store"`
}

// Collect: 헬스 상태를 수집합니다.
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
		Version:      version,
		Uptime:       formatDuration(time.Since(startTime)),
		Goroutines:   runtime.NumGoroutine(),
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
			"uptime":         formatDuration(time.Since(startTime)),
			"version":        version,
			"goroutines":     runtime.NumGoroutine(),
		},
	}
}

// formatDuration: Duration을 사람이 읽기 쉬운 형식으로 변환
func formatDuration(d time.Duration) string {
	d = d.Round(time.Second)
	h := d / time.Hour
	d -= h * time.Hour
	m := d / time.Minute
	d -= m * time.Minute
	s := d / time.Second

	if h > 0 {
		return time.Duration(h*time.Hour + m*time.Minute + s*time.Second).String()
	}
	if m > 0 {
		return time.Duration(m*time.Minute + s*time.Second).String()
	}
	return time.Duration(s * time.Second).String()
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
	storeEnabled := false
	sessionTTL := 0
	storeAddr := ""
	storeAddrErr := ""
	backend := "memory"
	reachability := false
	pingErr := ""
	checked := false

	if cfg != nil {
		storeEnabled = cfg.SessionStore.Enabled
		sessionTTL = cfg.Session.SessionTTLMinutes
		if cfg.SessionStore.URL != "" {
			addr, err := storeAddress(cfg.SessionStore.URL)
			if err != nil {
				storeAddrErr = err.Error()
			} else {
				storeAddr = addr
			}
		}
	}
	if ctx == nil {
		ctx = context.Background()
	}
	if storeEnabled {
		backend = "valkey"
	}
	if storeEnabled && deepChecks {
		checked = true
		checkCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), 2*time.Second)
		defer cancel()

		store, err := session.NewStore(cfg)
		if err != nil {
			pingErr = err.Error()
		} else {
			defer store.Close()
			if err := store.Ping(checkCtx); err != nil {
				pingErr = err.Error()
			} else {
				reachability = true
			}
		}
	}

	status := "ok"
	if storeEnabled && deepChecks && !reachability {
		status = "degraded"
	}

	var storeConnected any
	if checked {
		storeConnected = reachability
	} else {
		storeConnected = nil
	}

	detail := map[string]any{
		"store_enabled":       storeEnabled,
		"store_connected":     storeConnected,
		"backend":             backend,
		"store_address":       storeAddr,
		"session_ttl_minutes": sessionTTL,
		"deep_checked":        deepChecks,
	}
	if storeAddrErr != "" {
		detail["store_address_error"] = storeAddrErr
	}
	if pingErr != "" {
		detail["store_ping_error"] = pingErr
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
