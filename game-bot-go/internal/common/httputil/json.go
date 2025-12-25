package httputil

import (
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"

	json "github.com/goccy/go-json"
)

// HTTP 헤더 관련 상수
const (
	// ContentTypeJSON: JSON 응답을 위한 Content-Type 헤더 값
	ContentTypeJSON = "application/json"
	// HeaderAPIKey: API 키 인증 헤더 이름
	HeaderAPIKey = "X-API-Key"
	// HeaderContentType: Content-Type 헤더 이름
	HeaderContentType = "Content-Type"
)

// ErrEmptyBody: 요청 바디가 비어있을 때 발생하는 에러
var ErrEmptyBody = errors.New("empty request body")

// ReadJSON: HTTP 요청 바디에서 JSON을 읽어 대상 구조체로 디코딩한다.
func ReadJSON(r *http.Request, out any, maxBytes int64) error {
	if r.Body == nil {
		return ErrEmptyBody
	}

	reader := io.LimitReader(r.Body, maxBytes+1)
	dec := json.NewDecoder(reader)

	if err := dec.Decode(out); err != nil {
		if errors.Is(err, io.EOF) {
			return ErrEmptyBody
		}
		return fmt.Errorf("decode json failed: %w", err)
	}
	return nil
}

// WriteJSON: 데이터를 JSON으로 인코딩하여 HTTP 응답으로 전송한다.
func WriteJSON(w http.ResponseWriter, status int, v any) error {
	w.Header().Set("Content-Type", ContentTypeJSON)
	w.WriteHeader(status)

	enc := json.NewEncoder(w)
	enc.SetEscapeHTML(false)
	if err := enc.Encode(v); err != nil {
		return fmt.Errorf("encode json failed: %w", err)
	}
	return nil
}

// ErrorResponse: 표준 에러 응답 구조체
type ErrorResponse struct {
	Error   string `json:"error"`
	Message string `json:"message"`
}

// WriteErrorJSON: 에러 코드와 메시지를 포함한 표준 에러 응답을 전송한다.
func WriteErrorJSON(w http.ResponseWriter, status int, code string, message string) error {
	return WriteJSON(w, status, ErrorResponse{
		Error:   strings.TrimSpace(code),
		Message: strings.TrimSpace(message),
	})
}
