package llmrest

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	json "github.com/goccy/go-json"
)

func TestNew(t *testing.T) {
	tests := []struct {
		name    string
		baseURL string
		wantErr bool
	}{
		{"valid http", "http://example.com", false},
		{"valid https", "https://example.com", false},
		{"empty", "", true},
		{"invalid url", "://invalid", true},
		{"no scheme", "example.com", true}, // scheme이 없으면 호스트 파싱이 다름, 하지만 Client 로직상 Scheme 체크함
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := New(Config{BaseURL: tt.baseURL})
			if (err != nil) != tt.wantErr {
				t.Errorf("New() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestClient_Methods(t *testing.T) {
	// 모의 서버 설정
	handleFunc := func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Accept") != "application/json" {
			t.Errorf("expected Accept header application/json, got %s", r.Header.Get("Accept"))
		}

		switch r.URL.Path {
		case "/get":
			if r.Method != http.MethodGet {
				t.Errorf("expected POST, got %s", r.Method)
			}
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{"result":"ok"}`))

		case "/post":
			if r.Method != http.MethodPost {
				t.Errorf("expected POST, got %s", r.Method)
			}
			if r.Header.Get("Content-Type") != "application/json" {
				t.Errorf("expected Content-Type application/json, got %s", r.Header.Get("Content-Type"))
			}
			var body map[string]string
			if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
				t.Errorf("decode request body failed: %v", err)
			}
			if body["key"] != "value" {
				t.Errorf("expected body key=value, got %v", body)
			}
			w.WriteHeader(http.StatusCreated)
			w.Write([]byte(`{"id":123}`))

		case "/delete":
			if r.Method != http.MethodDelete {
				t.Errorf("expected DELETE, got %s", r.Method)
			}
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{"deleted":true}`))

		case "/error":
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte(`{"error":"bad_request"}`))

		case "/empty":
			w.WriteHeader(http.StatusOK)
			// 빈 바디

		case "/invalid-json":
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{invalid json`))

		case "/headers":
			if r.Header.Get("X-Custom") != "foo" {
				t.Errorf("expected X-Custom header foo, got %s", r.Header.Get("X-Custom"))
			}
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{}`))
		}
	}

	server := httptest.NewServer(http.HandlerFunc(handleFunc))
	defer server.Close()

	client, err := New(Config{BaseURL: server.URL, Timeout: 1 * time.Second})
	if err != nil {
		t.Fatalf("failed to create client: %v", err)
	}
	ctx := context.Background()

	// 1. Get
	t.Run("Get", func(t *testing.T) {
		var out struct {
			Result string `json:"result"`
		}
		if err := client.Get(ctx, "/get", &out); err != nil {
			t.Errorf("Get failed: %v", err)
		}
		if out.Result != "ok" {
			t.Errorf("expected result ok, got %s", out.Result)
		}
	})

	// 2. Post
	t.Run("Post", func(t *testing.T) {
		in := map[string]string{"key": "value"}
		var out struct {
			ID int `json:"id"`
		}
		if err := client.Post(ctx, "/post", in, &out); err != nil {
			t.Errorf("Post failed: %v", err)
		}
		if out.ID != 123 {
			t.Errorf("expected id 123, got %d", out.ID)
		}
	})

	// 3. Delete
	t.Run("Delete", func(t *testing.T) {
		var out struct {
			Deleted bool `json:"deleted"`
		}
		if err := client.Delete(ctx, "/delete", &out); err != nil {
			t.Errorf("Delete failed: %v", err)
		}
		if !out.Deleted {
			t.Error("expected deleted true")
		}
	})

	// 4. Error response
	t.Run("Error", func(t *testing.T) {
		var out struct{}
		err := client.Get(ctx, "/error", &out)
		if err == nil {
			t.Error("expected error, got nil")
		}
		// err 타입 검사 (httpError)
		// private type이라 직접 assert는 못하지만 Error() 문자열 체크
		if err.Error() != "http error status=400 body={\"error\":\"bad_request\"}" {
			t.Errorf("unexpected error message: %v", err)
		}
	})

	// 5. Empty response body with out != nil
	t.Run("EmptyResponse", func(t *testing.T) {
		var out struct{}
		err := client.Get(ctx, "/empty", &out)
		if err == nil {
			t.Error("expected error for empty body, got nil")
		}
	})

	// 6. Invalid JSON response
	t.Run("InvalidJSON", func(t *testing.T) {
		var out struct{}
		err := client.Get(ctx, "/invalid-json", &out)
		if err == nil {
			t.Error("expected error for invalid json, got nil")
		}
	})

	// 7. With Headers
	t.Run("WithHeaders", func(t *testing.T) {
		headers := map[string]string{"X-Custom": "foo"}
		var out struct{}
		if err := client.GetWithHeaders(ctx, "/headers", headers, &out); err != nil {
			t.Errorf("GetWithHeaders failed: %v", err)
		}
	})
}
