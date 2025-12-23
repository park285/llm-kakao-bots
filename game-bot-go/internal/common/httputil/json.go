package httputil

import (
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"

	json "github.com/goccy/go-json"
)

// ErrEmptyBody 는 패키지 변수다.
var ErrEmptyBody = errors.New("empty request body")

// ReadJSON 는 동작을 수행한다.
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

// WriteJSON 는 동작을 수행한다.
func WriteJSON(w http.ResponseWriter, status int, v any) error {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)

	enc := json.NewEncoder(w)
	enc.SetEscapeHTML(false)
	if err := enc.Encode(v); err != nil {
		return fmt.Errorf("encode json failed: %w", err)
	}
	return nil
}

// ErrorResponse 는 타입이다.
type ErrorResponse struct {
	Error   string `json:"error"`
	Message string `json:"message"`
}

// WriteErrorJSON 는 동작을 수행한다.
func WriteErrorJSON(w http.ResponseWriter, status int, code string, message string) error {
	return WriteJSON(w, status, ErrorResponse{
		Error:   strings.TrimSpace(code),
		Message: strings.TrimSpace(message),
	})
}
