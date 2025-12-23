package service

import (
	"context"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
)

func TestFrankfurterExchangeRateService(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))

	t.Run("Success_Fetch", func(t *testing.T) {
		ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{"rates":{"KRW": 1500.0}}`))
		}))
		defer ts.Close()

		svc := NewFrankfurterExchangeRateService(logger)
		svc.apiURL = ts.URL

		rate := svc.getUsdKrwRate(context.Background())
		if rate != 1500.0 {
			t.Errorf("expected 1500.0, got %f", rate)
		}

		// Check converter
		converted := svc.UsdToKrw(context.Background(), 2.0)
		if converted != 3000.0 {
			t.Errorf("expected 3000.0, got %f", converted)
		}

		// Check Info
		info := svc.RateInfo(context.Background())
		if info != "1 USD = 1,500 KRW" {
			t.Errorf("expected '1 USD = 1,500 KRW', got '%s'", info)
		}
	})

	t.Run("Cache_Hit", func(t *testing.T) {
		calls := 0
		ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			calls++
			w.Write([]byte(`{"rates":{"KRW": 1000.0}}`))
		}))
		defer ts.Close()

		svc := NewFrankfurterExchangeRateService(logger)
		svc.apiURL = ts.URL

		// First call
		svc.getUsdKrwRate(context.Background())

		// Second call (should hit cache)
		svc.getUsdKrwRate(context.Background())

		if calls != 1 {
			t.Errorf("expected 1 call due to cache, got %d", calls)
		}
	})

	t.Run("Failures_Fallback", func(t *testing.T) {
		tests := []struct {
			name    string
			handler http.HandlerFunc
		}{
			{"ServerError", func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(500) }},
			{"BadJSON", func(w http.ResponseWriter, r *http.Request) { w.Write([]byte(`{invalid`)) }},
			{"MissingRate", func(w http.ResponseWriter, r *http.Request) { w.Write([]byte(`{"rates":{}}`)) }},
			{"ZeroRate", func(w http.ResponseWriter, r *http.Request) { w.Write([]byte(`{"rates":{"KRW":0}}`)) }},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				ts := httptest.NewServer(tt.handler)
				defer ts.Close()

				svc := NewFrankfurterExchangeRateService(logger)
				svc.apiURL = ts.URL

				rate := svc.getUsdKrwRate(context.Background())
				if rate != defaultUsdKrwRate {
					t.Errorf("expected default rate %f, got %f", defaultUsdKrwRate, rate)
				}
			})
		}
	})

	t.Run("Network_Error", func(t *testing.T) {
		svc := NewFrankfurterExchangeRateService(logger)
		svc.apiURL = "http://invalid-url-that-fails.com"
		// Short timeout for test speed? Client has 10s.
		// We can replace client or just trust it eventually fails.
		// Or assume invalid URL fails quickly on DNS/connection.
		// Actually, let's just use empty string or bad protocol to fail creation request?
		// NewRequestWithContext checks URL parse.

		svc.apiURL = "::invalid" // Should fail NewRequest
		rate := svc.getUsdKrwRate(context.Background())
		if rate != defaultUsdKrwRate {
			t.Errorf("expected default rate for bad URL")
		}
	})
}
