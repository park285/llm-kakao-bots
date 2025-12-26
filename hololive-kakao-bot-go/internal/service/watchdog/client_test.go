package watchdog

import (
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
)

func TestClientRequests(t *testing.T) {
	var (
		mu             sync.Mutex
		tokens         []string
		restartPayload map[string]any
	)
	expectedToken := "token-123"

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/health" {
			mu.Lock()
			tokens = append(tokens, r.Header.Get("X-Internal-Service-Token"))
			mu.Unlock()
		}

		switch {
		case r.URL.Path == "/health":
			w.WriteHeader(http.StatusOK)
		case r.URL.Path == "/admin/api/v1/docker/containers":
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(ContainersResponse{
				Containers:  []ContainerInfo{{Name: "c1"}},
				GeneratedAt: "now",
			})
		case r.URL.Path == "/admin/api/v1/targets":
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(map[string]any{
				"targets": []ContainerInfo{{Name: "c2"}},
			})
		case strings.HasPrefix(r.URL.Path, "/admin/api/v1/targets/") && strings.HasSuffix(r.URL.Path, "/restart"):
			if r.Method != http.MethodPost {
				w.WriteHeader(http.StatusMethodNotAllowed)
				return
			}
			body, _ := io.ReadAll(r.Body)
			_ = json.Unmarshal(body, &restartPayload)
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(RestartResponse{Status: "ok"})
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	client := NewClientWithToken(server.URL, expectedToken, logger)

	ctx := context.Background()

	containers, err := client.GetContainers(ctx)
	if err != nil {
		t.Fatalf("GetContainers failed: %v", err)
	}
	if len(containers.Containers) != 1 || containers.Containers[0].Name != "c1" {
		t.Fatalf("unexpected containers response: %+v", containers)
	}

	targets, err := client.GetManagedTargets(ctx)
	if err != nil {
		t.Fatalf("GetManagedTargets failed: %v", err)
	}
	if len(targets) != 1 || targets[0].Name != "c2" {
		t.Fatalf("unexpected targets response: %+v", targets)
	}

	resp, err := client.RestartContainer(ctx, "c1", "reason", true)
	if err != nil {
		t.Fatalf("RestartContainer failed: %v", err)
	}
	if resp.Status != "ok" {
		t.Fatalf("unexpected restart response: %+v", resp)
	}
	if restartPayload["reason"] != "reason" || restartPayload["force"] != true {
		t.Fatalf("unexpected restart payload: %+v", restartPayload)
	}

	if !client.IsAvailable(ctx) {
		t.Fatalf("expected watchdog to be available")
	}

	mu.Lock()
	defer mu.Unlock()
	if len(tokens) != 3 {
		t.Fatalf("expected tokens for requests, got %d", len(tokens))
	}
	for _, token := range tokens {
		if token != expectedToken {
			t.Fatalf("unexpected token: %s", token)
		}
	}
}
