package watchdog

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"go.uber.org/zap"
)

// ContainerInfo represents a container's status from watchdog.
type ContainerInfo struct {
	Name       string `json:"name"`
	ID         string `json:"id"`
	Image      string `json:"image"`
	State      string `json:"state"`
	Status     string `json:"status"`
	Health     string `json:"health"`
	Managed    bool   `json:"managed"`
	Paused     bool   `json:"paused"`
	StartedAt  string `json:"startedAt,omitempty"`
	FinishedAt string `json:"finishedAt,omitempty"`
}

// ContainersResponse is the response from watchdog containers endpoint.
type ContainersResponse struct {
	Containers  []ContainerInfo `json:"containers"`
	GeneratedAt string          `json:"generatedAt"`
}

// RestartResponse is the response from a restart request.
type RestartResponse struct {
	Status  string `json:"status"`
	Message string `json:"message,omitempty"`
	Error   *struct {
		Code    string `json:"code"`
		Message string `json:"message"`
	} `json:"error,omitempty"`
}

// Client is a client for the watchdog admin API.
type Client struct {
	baseURL    string
	httpClient *http.Client
	logger     *zap.Logger
}

// NewClient creates a new watchdog API client.
func NewClient(baseURL string, logger *zap.Logger) *Client {
	return &Client{
		baseURL: baseURL,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
		logger: logger,
	}
}

// GetContainers fetches the list of all containers.
func (c *Client) GetContainers(ctx context.Context) (*ContainersResponse, error) {
	url := c.baseURL + "/admin/api/v1/docker/containers?skip_auth=true"

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Accept", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("do request: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("unexpected status %d: %s", resp.StatusCode, string(body))
	}

	var result ContainersResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("decode response: %w", err)
	}

	return &result, nil
}

// GetManagedTargets fetches the list of managed targets.
func (c *Client) GetManagedTargets(ctx context.Context) ([]ContainerInfo, error) {
	url := c.baseURL + "/admin/api/v1/targets?skip_auth=true"

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Accept", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("do request: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("unexpected status %d: %s", resp.StatusCode, string(body))
	}

	var result struct {
		Targets []ContainerInfo `json:"targets"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("decode response: %w", err)
	}

	return result.Targets, nil
}

// RestartContainer sends a restart request to the watchdog.
func (c *Client) RestartContainer(ctx context.Context, name, reason string, force bool) (*RestartResponse, error) {
	url := c.baseURL + "/admin/api/v1/targets/" + name + "/restart?skip_auth=true"

	body := map[string]interface{}{
		"reason": reason,
		"force":  force,
	}
	bodyBytes, err := json.Marshal(body)
	if err != nil {
		return nil, fmt.Errorf("marshal body: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(bodyBytes))
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")

	c.logger.Info("Sending restart request to watchdog",
		zap.String("container", name),
		zap.String("reason", reason),
		zap.Bool("force", force),
	)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("do request: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	var result RestartResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		// non-JSON 응답 처리
		bodyBytes, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("decode response (status %d): %s", resp.StatusCode, string(bodyBytes))
	}

	if resp.StatusCode >= 400 {
		if result.Error != nil {
			return nil, fmt.Errorf("watchdog error: %s - %s", result.Error.Code, result.Error.Message)
		}
		return nil, fmt.Errorf("unexpected status %d", resp.StatusCode)
	}

	return &result, nil
}

// IsAvailable checks if the watchdog API is reachable.
func (c *Client) IsAvailable(ctx context.Context) bool {
	url := c.baseURL + "/health"

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return false
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return false
	}
	defer func() { _ = resp.Body.Close() }()

	return resp.StatusCode == http.StatusOK
}
