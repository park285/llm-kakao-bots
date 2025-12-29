package llmrest

import (
	"bytes"
	"compress/gzip"
	"context"
	"errors"
	"fmt"
	"io"
	"math/rand/v2"
	"net"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/goccy/go-json"

	"github.com/park285/llm-kakao-bots/game-bot-go/internal/common/httpclient"
	"github.com/park285/llm-kakao-bots/game-bot-go/internal/common/httputil"
)

// Config: REST 클라이언트 설정
type Config struct {
	BaseURL          string
	APIKey           string
	Timeout          time.Duration
	ConnectTimeout   time.Duration
	HTTP2Enabled     bool
	RetryMaxAttempts int
	RetryDelay       time.Duration
}

// Client: LLM 서버와 통신하기 위한 HTTP 클라이언트
type Client struct {
	baseURL          *url.URL
	httpClient       *http.Client
	apiKey           string
	retryMaxAttempts int
	retryDelay       time.Duration
}

// New: 새로운 Client 인스턴스를 생성하고 초기화합니다.
func New(cfg Config) (*Client, error) {
	parsedBaseURL, err := url.Parse(strings.TrimSpace(cfg.BaseURL))
	if err != nil {
		return nil, fmt.Errorf("parse base url failed: %w", err)
	}
	if parsedBaseURL.Scheme == "" || parsedBaseURL.Host == "" {
		return nil, fmt.Errorf("invalid base url: %q", cfg.BaseURL)
	}

	retryMaxAttempts := cfg.RetryMaxAttempts
	if retryMaxAttempts < 1 {
		retryMaxAttempts = 1
	}
	retryDelay := cfg.RetryDelay
	if retryDelay < 0 {
		retryDelay = 0
	}

	return &Client{
		baseURL: parsedBaseURL,
		httpClient: httpclient.New(httpclient.Config{
			Timeout:        cfg.Timeout,
			ConnectTimeout: cfg.ConnectTimeout,
			HTTP2Enabled:   cfg.HTTP2Enabled,
		}),
		apiKey:           strings.TrimSpace(cfg.APIKey),
		retryMaxAttempts: retryMaxAttempts,
		retryDelay:       retryDelay,
	}, nil
}

type httpError struct {
	StatusCode int
	Body       string
}

func (e httpError) Error() string {
	if e.Body == "" {
		return fmt.Sprintf("http error status=%d", e.StatusCode)
	}
	return fmt.Sprintf("http error status=%d body=%s", e.StatusCode, e.Body)
}

func shouldRetry(err error) bool {
	if err == nil {
		return false
	}
	if errors.Is(err, context.Canceled) {
		return false
	}
	if errors.Is(err, context.DeadlineExceeded) {
		return true
	}

	var netErr net.Error
	if errors.As(err, &netErr) {
		if netErr.Timeout() {
			return true
		}
	}

	var httpErr httpError
	if errors.As(err, &httpErr) {
		return httpErr.StatusCode >= http.StatusInternalServerError
	}

	return false
}

func responseBodyReader(resp *http.Response) (io.Reader, func(), error) {
	contentEncoding := strings.ToLower(strings.TrimSpace(resp.Header.Get("Content-Encoding")))
	if !strings.Contains(contentEncoding, "gzip") {
		return resp.Body, func() {}, nil
	}

	gzipReader, err := gzip.NewReader(resp.Body)
	if err != nil {
		return nil, func() {}, fmt.Errorf("create gzip reader failed: %w", err)
	}

	return gzipReader, func() { _ = gzipReader.Close() }, nil
}

// Get: HTTP GET 요청을 전송합니다.
func (c *Client) Get(ctx context.Context, path string, out any) error {
	return c.doJSON(ctx, http.MethodGet, path, nil, nil, out)
}

// Post: HTTP POST 요청을 전송합니다.
func (c *Client) Post(ctx context.Context, path string, in any, out any) error {
	return c.doJSON(ctx, http.MethodPost, path, in, nil, out)
}

// Delete: HTTP DELETE 요청을 전송합니다.
func (c *Client) Delete(ctx context.Context, path string, out any) error {
	return c.doJSON(ctx, http.MethodDelete, path, nil, nil, out)
}

// GetWithHeaders: 헤더를 포함하여 HTTP GET 요청을 전송합니다.
func (c *Client) GetWithHeaders(ctx context.Context, path string, headers map[string]string, out any) error {
	return c.doJSON(ctx, http.MethodGet, path, nil, headers, out)
}

func (c *Client) doJSON(
	ctx context.Context,
	method string,
	path string,
	in any,
	headers map[string]string,
	out any,
) error {
	var payload []byte
	if in != nil {
		encoded, marshalErr := json.Marshal(in)
		if marshalErr != nil {
			return fmt.Errorf("marshal request body failed: %w", marshalErr)
		}
		payload = encoded
	}

	attempts := c.retryMaxAttempts
	if attempts < 1 {
		attempts = 1
	}

	var lastErr error
	for attempt := 1; attempt <= attempts; attempt++ {
		if ctx.Err() != nil {
			return fmt.Errorf("request context done: %w", ctx.Err())
		}

		err := c.doJSONOnce(ctx, method, path, payload, headers, out)
		if err == nil {
			return nil
		}
		lastErr = err

		if !shouldRetry(err) || attempt == attempts {
			return err
		}

		if c.retryDelay > 0 {
			// Jitter: Thundering Herd 문제를 방지하기 위해 0.8~1.2 배수의 무작위 지연 시간을 적용합니다.
			jitter := time.Duration(float64(c.retryDelay) * (0.8 + rand.Float64()*0.4))
			timer := time.NewTimer(jitter)
			select {
			case <-ctx.Done():
				timer.Stop()
				return fmt.Errorf("request context done: %w", ctx.Err())
			case <-timer.C:
			}
		}
	}

	return lastErr
}

func (c *Client) doJSONOnce(
	ctx context.Context,
	method string,
	path string,
	payload []byte,
	headers map[string]string,
	out any,
) error {
	// JoinPath: URL 경로를 안전하게 결합하고 슬래시를 정리합니다.
	fullURL := c.baseURL.JoinPath(path)

	var body io.Reader
	if payload != nil {
		body = bytes.NewReader(payload)
	}

	req, err := http.NewRequestWithContext(ctx, method, fullURL.String(), body)
	if err != nil {
		return fmt.Errorf("create request failed: %w", err)
	}
	req.Header.Set("Accept", httputil.ContentTypeJSON)
	req.Header.Set("Accept-Encoding", "gzip")
	if payload != nil {
		req.Header.Set("Content-Type", httputil.ContentTypeJSON)
	}
	if c.apiKey != "" {
		req.Header.Set(httputil.HeaderAPIKey, c.apiKey)
	}
	for k, v := range headers {
		req.Header.Set(k, v)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("request failed: %w", err)
	}
	defer func() {
		_ = resp.Body.Close()
	}()

	// 상태 코드 우선 확인
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		// 오류 발생 시에만 본문 읽기
		decodeReader, closeReader, bodyReaderErr := responseBodyReader(resp)
		if bodyReaderErr != nil {
			return bodyReaderErr
		}
		defer closeReader()

		respBytes, readBodyErr := io.ReadAll(io.LimitReader(decodeReader, 2*1024*1024))
		if readBodyErr != nil {
			return fmt.Errorf("read error response body failed: %w", readBodyErr)
		}
		return httpError{StatusCode: resp.StatusCode, Body: strings.TrimSpace(string(respBytes))}
	}

	if out == nil {
		// 결과가 필요 없으면 drain 하지 않고 종료 (defer에서 닫힘)
		return nil
	}

	// Stream Decoder를 사용하여 Zero-Copy 파싱 수행
	// 전체 본문을 메모리에 올리지 않아 효율적임
	decodeReader, closeReader, bodyReaderErr := responseBodyReader(resp)
	if bodyReaderErr != nil {
		return bodyReaderErr
	}
	defer closeReader()

	if err := json.NewDecoder(decodeReader).Decode(out); err != nil {
		if errors.Is(err, io.EOF) {
			return errors.New("empty response body")
		}
		return fmt.Errorf("decode response failed: %w", err)
	}
	return nil
}
