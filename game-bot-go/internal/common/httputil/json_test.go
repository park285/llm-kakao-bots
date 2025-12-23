package httputil

import (
	"bytes"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestReadJSON_Success(t *testing.T) {
	body := `{"name":"test","value":123}`
	req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(body))

	var out struct {
		Name  string `json:"name"`
		Value int    `json:"value"`
	}

	err := ReadJSON(req, &out, 1024)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if out.Name != "test" {
		t.Errorf("expected name 'test', got '%s'", out.Name)
	}
	if out.Value != 123 {
		t.Errorf("expected value 123, got %d", out.Value)
	}
}

func TestReadJSON_EmptyBody(t *testing.T) {
	req := httptest.NewRequest(http.MethodPost, "/", nil)

	var out struct{}
	err := ReadJSON(req, &out, 1024)
	if err != ErrEmptyBody {
		t.Errorf("expected ErrEmptyBody, got %v", err)
	}
}

func TestReadJSON_InvalidJSON(t *testing.T) {
	body := `{invalid json}`
	req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(body))

	var out struct{}
	err := ReadJSON(req, &out, 1024)
	if err == nil {
		t.Error("expected error for invalid JSON")
	}
}

func TestWriteJSON_Success(t *testing.T) {
	rr := httptest.NewRecorder()

	data := map[string]any{
		"message": "hello",
		"count":   42,
	}

	err := WriteJSON(rr, http.StatusOK, data)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if rr.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", rr.Code)
	}

	contentType := rr.Header().Get("Content-Type")
	if contentType != "application/json" {
		t.Errorf("expected Content-Type 'application/json', got '%s'", contentType)
	}

	// 응답 바디 확인
	body := rr.Body.String()
	if !strings.Contains(body, "hello") || !strings.Contains(body, "42") {
		t.Errorf("unexpected body: %s", body)
	}
}

func TestWriteJSON_HTMLEscape(t *testing.T) {
	rr := httptest.NewRecorder()

	data := map[string]string{
		"html": "<script>alert('xss')</script>",
	}

	err := WriteJSON(rr, http.StatusOK, data)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	body := rr.Body.String()
	// SetEscapeHTML(false)이므로 이스케이프되지 않아야 함
	if strings.Contains(body, "\\u003c") {
		t.Errorf("HTML should not be escaped: %s", body)
	}
}

func TestWriteErrorJSON(t *testing.T) {
	rr := httptest.NewRecorder()

	err := WriteErrorJSON(rr, http.StatusBadRequest, "INVALID_INPUT", "field is required")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if rr.Code != http.StatusBadRequest {
		t.Errorf("expected status 400, got %d", rr.Code)
	}

	body := rr.Body.String()
	if !strings.Contains(body, "INVALID_INPUT") {
		t.Errorf("expected error code in body: %s", body)
	}
	if !strings.Contains(body, "field is required") {
		t.Errorf("expected message in body: %s", body)
	}
}

func TestWriteErrorJSON_TrimWhitespace(t *testing.T) {
	rr := httptest.NewRecorder()

	err := WriteErrorJSON(rr, http.StatusInternalServerError, "  ERROR_CODE  ", "  message with spaces  ")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	body := rr.Body.String()
	// 공백이 제거되어야 함
	if strings.Contains(body, `"  ERROR_CODE  "`) {
		t.Errorf("whitespace should be trimmed: %s", body)
	}
}

type testReader struct {
	data    []byte
	readPos int
}

func (r *testReader) Read(p []byte) (int, error) {
	if r.readPos >= len(r.data) {
		return 0, io.EOF
	}
	n := copy(p, r.data[r.readPos:])
	r.readPos += n
	return n, nil
}

func TestReadJSON_LargeBody(t *testing.T) {
	// maxBytes보다 큰 바디 테스트
	largeData := bytes.Repeat([]byte("a"), 2000)
	body := `{"data":"` + string(largeData) + `"}`
	req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(body))

	var out struct {
		Data string `json:"data"`
	}

	// maxBytes=1024이면 읽기 제한됨
	err := ReadJSON(req, &out, 1024)
	// LimitReader가 중간에 자르므로 JSON 파싱 에러 발생 예상
	if err == nil {
		t.Error("expected error for body exceeding maxBytes")
	}
}
