package llmrest

import (
	"bytes"
	"compress/gzip"
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/goccy/go-json"

	"github.com/park285/llm-kakao-bots/game-bot-go/internal/common/httpclient"
)

// Config 는 타입이다.
type Config struct {
	BaseURL          string
	APIKey           string
	Timeout          time.Duration
	ConnectTimeout   time.Duration
	HTTP2Enabled     bool
	RetryMaxAttempts int
	RetryDelay       time.Duration
}

// Client 는 타입이다.
type Client struct {
	baseURL          *url.URL
	httpClient       *http.Client
	apiKey           string
	retryMaxAttempts int
	retryDelay       time.Duration
}

// New 는 동작을 수행한다.
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

// Get 는 동작을 수행한다.
func (c *Client) Get(ctx context.Context, path string, out any) error {
	return c.doJSON(ctx, http.MethodGet, path, nil, nil, out)
}

// Post 는 동작을 수행한다.
func (c *Client) Post(ctx context.Context, path string, in any, out any) error {
	return c.doJSON(ctx, http.MethodPost, path, in, nil, out)
}

// Delete 는 동작을 수행한다.
func (c *Client) Delete(ctx context.Context, path string, out any) error {
	return c.doJSON(ctx, http.MethodDelete, path, nil, nil, out)
}

// GetWithHeaders 는 동작을 수행한다.
func (c *Client) GetWithHeaders(ctx context.Context, path string, headers map[string]string, out any) error {
	return c.doJSON(ctx, http.MethodGet, path, nil, headers, out)
}

// PostWithHeaders 는 동작을 수행한다.
func (c *Client) PostWithHeaders(ctx context.Context, path string, headers map[string]string, in any, out any) error {
	return c.doJSON(ctx, http.MethodPost, path, in, headers, out)
}

// DeleteWithHeaders 는 동작을 수행한다.
func (c *Client) DeleteWithHeaders(ctx context.Context, path string, headers map[string]string, out any) error {
	return c.doJSON(ctx, http.MethodDelete, path, nil, headers, out)
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
			timer := time.NewTimer(c.retryDelay)
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
	fullURL, err := c.baseURL.Parse(path)
	if err != nil {
		return fmt.Errorf("build request url failed path=%s: %w", path, err)
	}

	var body io.Reader
	if payload != nil {
		body = bytes.NewReader(payload)
	}

	req, err := http.NewRequestWithContext(ctx, method, fullURL.String(), body)
	if err != nil {
		return fmt.Errorf("create request failed: %w", err)
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Accept-Encoding", "gzip")
	if payload != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	if c.apiKey != "" {
		req.Header.Set("X-API-Key", c.apiKey)
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

	// Check status code first
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		// Only read body for error cases
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
		// Eliminate drain body if not needed, close is enough (handled by defer)
		return nil
	}

	// Use Stream Decoder for Zero-Copy parsing
	// This is the most efficient way as it avoids allocating a large buffer for the entire body
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
