package holodex

import (
	"context"
	stdErrors "errors"
	"net/http"
	"reflect"
	"testing"

	"log/slog"
)

func TestHolodexAPIClientRotatesAllKeys(t *testing.T) {
	logger := slog.Default()
	client := &APIClient{
		httpClient: &http.Client{},
		apiKeys: []string{
			"k1",
			"k2",
			"k3",
			"k4",
			"k5",
		},
		logger: logger,
	}

	got := make([]string, 0, 10)
	for i := 0; i < 10; i++ {
		got = append(got, client.getNextAPIKey())
	}

	expected := []string{"k1", "k2", "k3", "k4", "k5", "k1", "k2", "k3", "k4", "k5"}
	if !reflect.DeepEqual(got, expected) {
		t.Fatalf("rotation order mismatch: got %v expected %v", got, expected)
	}
}

func TestHolodexAPIClientDoRequestNoKeys(t *testing.T) {
	logger := slog.Default()
	client := &APIClient{
		httpClient: &http.Client{},
		apiKeys:    nil,
		logger:     logger,
	}

	_, err := client.DoRequest(context.Background(), http.MethodGet, "/live", nil)
	if err == nil {
		t.Fatalf("expected error when no API keys configured")
	}
	if !stdErrors.Is(err, errNoAPIKeys) {
		t.Fatalf("unexpected error: %v", err)
	}
}
