// Holodex API Integration
//
// 본 코드는 Holodex API (https://holodex.net)를 사용하며, Holodex API Terms of Service를 준수합니다.
//
// Attribution (Holodex API Terms Section 6):
//   - API Provider: Holodex (https://holodex.net)
//   - License: https://holodex.net/api/terms
//   - Disclaimer: THE HOLODEX API IS PROVIDED "AS IS" WITHOUT WARRANTY OF ANY KIND.
//
// See: https://holodex.net/api/terms for full terms.

package holodex

import (
	"context"
	stdErrors "errors"
	"fmt"
	"io"
	"log/slog"
	"math"
	"math/rand/v2"
	"net/http"
	"net/url"
	"sync"
	"time"

	"golang.org/x/time/rate"

	"github.com/kapu/hololive-kakao-bot-go/internal/constants"
	"github.com/kapu/hololive-kakao-bot-go/internal/util"
	"github.com/kapu/hololive-kakao-bot-go/pkg/errors"
)

// Requester: HTTP 요청 수행 및 서킷 브레이커 상태 확인을 위한 인터페이스
type Requester interface {
	DoRequest(ctx context.Context, method, path string, params url.Values) ([]byte, error)
	IsCircuitOpen() bool
}

// APIClient: Holodex API 요청을 처리하는 클라이언트
// API 키 로테이션, 서킷 브레이커, 속도 제한(Rate Limiting) 기능을 포함합니다.
type APIClient struct {
	httpClient       *http.Client
	apiKeys          []string
	currentKeyIndex  int
	keyMu            sync.Mutex
	logger           *slog.Logger
	failureCount     int
	failureMu        sync.Mutex
	circuitOpenUntil *time.Time
	circuitMu        sync.RWMutex
	rateLimiter      *rate.Limiter // Rate limiter: 초당 10 요청
}

var errNoAPIKeys = stdErrors.New("no Holodex API keys configured")

// NewHolodexAPIClient: 새로운 Holodex API 클라이언트를 생성하고 초기화합니다.
// 초당 10회 요청 제한(Rate Limit)이 기본 설정된다.
func NewHolodexAPIClient(httpClient *http.Client, apiKeys []string, logger *slog.Logger) *APIClient {
	return &APIClient{
		httpClient:  httpClient,
		apiKeys:     apiKeys,
		logger:      logger,
		rateLimiter: rate.NewLimiter(rate.Every(100*time.Millisecond), 1), // 초당 10 요청
	}
}

// DoRequest: Holodex API에 요청을 보낸다.
// Rate Limit 준수, 서킷 브레이커 확인, API 키 로테이션 및 재시도 로직을 수행합니다.
func (c *APIClient) DoRequest(ctx context.Context, method, path string, params url.Values) ([]byte, error) {
	// Rate limit 체크
	limiter := c.rateLimiter
	if limiter == nil {
		limiter = rate.NewLimiter(rate.Every(100*time.Millisecond), 1) // 초당 10 요청
		c.rateLimiter = limiter
	}
	if err := limiter.Wait(ctx); err != nil {
		return nil, fmt.Errorf("rate limiter wait failed: %w", err)
	}

	if err := c.rejectIfCircuitOpen(); err != nil {
		return nil, err
	}

	totalKeys := len(c.apiKeys)
	if totalKeys == 0 {
		return nil, errNoAPIKeys
	}

	maxAttempts := util.Min(totalKeys+constants.RetryConfig.MaxAttempts, 10)
	var lastErr error

	for attempt := 0; attempt < maxAttempts; attempt++ {
		body, done, err := c.tryHolodexRequest(ctx, method, path, params, attempt, maxAttempts)
		if !done {
			if err != nil {
				lastErr = err
			}
			continue
		}

		if err != nil {
			return nil, err
		}

		c.resetCircuit()
		return body, nil
	}

	if lastErr != nil {
		return nil, lastErr
	}

	return nil, fmt.Errorf("holodex request failed")
}

func (c *APIClient) rejectIfCircuitOpen() error {
	if !c.IsCircuitOpen() {
		return nil
	}

	c.circuitMu.RLock()
	var remainingMs int64
	if c.circuitOpenUntil != nil {
		remainingMs = time.Until(*c.circuitOpenUntil).Milliseconds()
	}
	c.circuitMu.RUnlock()

	c.logger.Warn("Circuit breaker is open", slog.Int64("retry_after_ms", remainingMs))
	return errors.NewAPIError("Circuit breaker open", 503, map[string]any{
		"retry_after_ms": remainingMs,
	})
}

func (c *APIClient) tryHolodexRequest(ctx context.Context, method, path string, params url.Values, attempt, maxAttempts int) ([]byte, bool, error) {
	reqURL := c.buildRequestURL(path, params)
	req, err := c.newRequest(ctx, method, reqURL, c.getNextAPIKey())
	if err != nil {
		return nil, true, err
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		if c.retryAfterNetworkFailure(ctx, err, attempt, maxAttempts) {
			return nil, false, fmt.Errorf("HTTP request failed (retrying): %w", err)
		}
		return nil, true, fmt.Errorf("HTTP request failed: %w", err)
	}

	body, readErr := io.ReadAll(resp.Body)
	_ = resp.Body.Close()
	if readErr != nil {
		return nil, false, fmt.Errorf("failed to read response: %w", readErr)
	}

	return c.processHolodexResponse(ctx, resp.StatusCode, body, reqURL, attempt, maxAttempts)
}

func (c *APIClient) buildRequestURL(path string, params url.Values) string {
	reqURL := constants.APIConfig.HolodexBaseURL + path
	if params != nil {
		reqURL += "?" + params.Encode()
	}
	return reqURL
}

func (c *APIClient) newRequest(ctx context.Context, method, reqURL string, apiKey string) (*http.Request, error) {
	req, err := http.NewRequestWithContext(ctx, method, reqURL, http.NoBody)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-APIKEY", apiKey)
	// Holodex API Terms 준수를 위해 정직한 User-Agent 사용 (Section 6: Attribution)
	req.Header.Set("User-Agent", "api.capu.blog/hololive-bot (Linux; +https://api.capu.blog; Holodex API client)")
	return req, nil
}

func (c *APIClient) retryAfterNetworkFailure(ctx context.Context, err error, attempt, maxAttempts int) bool {
	count := c.incrementFailureCount()
	if count >= constants.CircuitBreakerConfig.FailureThreshold {
		c.openCircuit()
		return false
	}

	if attempt < maxAttempts-1 {
		delay := c.computeDelay(attempt)
		c.logger.Debug("Request failed, retrying",
			slog.Any("error", err),
			slog.Int("attempt", attempt+1),
			slog.Duration("delay", delay),
		)
		if !sleepWithContext(ctx, delay) {
			return false // context 취소 시 재시도 중단
		}
		return true
	}

	return false
}

func (c *APIClient) processHolodexResponse(ctx context.Context, status int, body []byte, reqURL string, attempt, maxAttempts int) ([]byte, bool, error) {
	switch {
	case status == 429 || status == 403:
		c.logger.Warn("Rate limited, rotating key",
			slog.Int("status", status),
			slog.Int("attempt", attempt+1),
		)
		if attempt < maxAttempts-1 {
			return nil, false, nil
		}
		return nil, true, errors.NewKeyRotationError("All API keys rate limited", status, map[string]any{
			"url": reqURL,
		})
	case status >= 500:
		return c.handleServerError(ctx, status, attempt, maxAttempts)
	case status >= 400:
		return nil, true, errors.NewAPIError(fmt.Sprintf("Client error: %d", status), status, map[string]any{
			"url":  reqURL,
			"body": string(body),
		})
	default:
		return body, true, nil
	}
}

func (c *APIClient) handleServerError(ctx context.Context, status, attempt, maxAttempts int) ([]byte, bool, error) {
	count := c.incrementFailureCount()
	c.logger.Warn("Server error",
		slog.Int("status", status),
		slog.Int("failure_count", count),
	)

	if count >= constants.CircuitBreakerConfig.FailureThreshold {
		c.openCircuit()
		return nil, true, errors.NewAPIError(fmt.Sprintf("Server error: %d", status), status, nil)
	}

	if attempt < maxAttempts-1 {
		delay := c.computeDelay(attempt)
		if !sleepWithContext(ctx, delay) {
			return nil, true, fmt.Errorf("context canceled during backoff: %w", ctx.Err()) // context 취소 시 재시도 중단
		}
		return nil, false, errors.NewAPIError(fmt.Sprintf("Server error: %d", status), status, nil)
	}

	return nil, true, errors.NewAPIError(fmt.Sprintf("Server error: %d", status), status, nil)
}

// IsCircuitOpen: 현재 서킷 브레이커가 열려있는지(요청 차단 상태인지) 확인합니다.
func (c *APIClient) IsCircuitOpen() bool {
	c.circuitMu.RLock()
	defer c.circuitMu.RUnlock()

	if c.circuitOpenUntil == nil {
		return false
	}

	if time.Now().After(*c.circuitOpenUntil) {
		return false
	}

	return true
}

func (c *APIClient) getNextAPIKey() string {
	c.keyMu.Lock()
	defer c.keyMu.Unlock()

	if len(c.apiKeys) == 0 {
		return ""
	}

	index := c.currentKeyIndex
	key := c.apiKeys[index]
	c.currentKeyIndex = (c.currentKeyIndex + 1) % len(c.apiKeys)

	c.logger.Debug("Holodex API key selected",
		slog.Int("index", index),
		slog.Int("pool_size", len(c.apiKeys)),
	)

	return key
}

func (c *APIClient) openCircuit() {
	c.circuitMu.Lock()
	defer c.circuitMu.Unlock()

	resetTime := time.Now().Add(constants.CircuitBreakerConfig.ResetTimeout)
	c.circuitOpenUntil = &resetTime
	c.failureCount = 0

	c.logger.Error("Holodex circuit breaker opened",
		slog.Duration("reset_timeout", constants.CircuitBreakerConfig.ResetTimeout),
	)
}

func (c *APIClient) resetCircuit() {
	c.circuitMu.Lock()
	defer c.circuitMu.Unlock()

	c.failureCount = 0
	c.circuitOpenUntil = nil
}

func (c *APIClient) incrementFailureCount() int {
	c.failureMu.Lock()
	defer c.failureMu.Unlock()

	c.failureCount++
	return c.failureCount
}

func (c *APIClient) computeDelay(attempt int) time.Duration {
	base := constants.RetryConfig.BaseDelay * time.Duration(math.Pow(2, float64(attempt)))
	jitter := time.Duration(rand.Float64() * float64(constants.RetryConfig.Jitter))
	return base + jitter
}

// sleepWithContext: context 취소를 지원하는 sleep
// 정상 대기 완료 시 true, context 취소 시 false 반환
func sleepWithContext(ctx context.Context, delay time.Duration) bool {
	timer := time.NewTimer(delay)
	defer timer.Stop()
	select {
	case <-timer.C:
		return true
	case <-ctx.Done():
		return false
	}
}
