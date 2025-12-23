package service

import (
	"context"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	json "github.com/goccy/go-json"
	"github.com/park285/llm-kakao-bots/game-bot-go/internal/common/llmrest"
	"github.com/park285/llm-kakao-bots/game-bot-go/internal/common/messageprovider"
	qmodel "github.com/park285/llm-kakao-bots/game-bot-go/internal/twentyq/model"
)

type usageTestEnv struct {
	handler *UsageHandler
	ts      *httptest.Server
	mocks   struct {
		daily   *llmrest.DailyUsageResponse
		weekly  *llmrest.UsageListResponse
		monthly *llmrest.UsageResponse
	}
}

type fixedExchangeRateService struct {
	rate float64
}

func (s fixedExchangeRateService) UsdToKrw(ctx context.Context, usdAmount float64) float64 {
	return usdAmount * s.rate
}

func (s fixedExchangeRateService) RateInfo(ctx context.Context) string {
	return "1 USD = 1,400 KRW"
}

func setupUsageTestEnv(t *testing.T) *usageTestEnv {
	env := &usageTestEnv{}
	model := "gemini-3-flash-preview"
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		if strings.Contains(r.URL.Path, "/api/usage/daily") {
			if env.mocks.daily != nil {
				json.NewEncoder(w).Encode(env.mocks.daily)
			} else {
				// Default mock
					json.NewEncoder(w).Encode(llmrest.DailyUsageResponse{
						UsageDate:    "2023-01-01",
						InputTokens:  1_000_000,
						OutputTokens: 1_000_000,
						TotalTokens:  2_000_000,
						RequestCount: 10,
						Model:        &model,
					})
				}
				return
			}

		if strings.Contains(r.URL.Path, "/api/usage/recent") {
			if env.mocks.weekly != nil {
				json.NewEncoder(w).Encode(env.mocks.weekly)
			} else {
					json.NewEncoder(w).Encode(llmrest.UsageListResponse{
						Usages:            []llmrest.DailyUsageResponse{},
						TotalInputTokens:  7_000_000,
						TotalOutputTokens: 14_000_000,
						TotalTokens:       21_000_000,
						Model:             &model,
					})
				}
				return
			}

		// Monthly via DB endpoint mock (check fetch logic in llmrest if changed)
		// UsageHandler calls GetUsageTotalFromDB -> /api/usage/total
		if strings.Contains(r.URL.Path, "/api/usage/total") {
			if env.mocks.monthly != nil {
				json.NewEncoder(w).Encode(env.mocks.monthly)
			} else {
					json.NewEncoder(w).Encode(llmrest.UsageResponse{
						InputTokens:  1_000_000,
						OutputTokens: 2_000_000,
						TotalTokens:  3_000_000,
						Model:        &model,
					})
				}
				return
			}

		w.WriteHeader(http.StatusNotFound)
	}))
	env.ts = ts

	llmClient, err := llmrest.New(llmrest.Config{BaseURL: ts.URL})
	if err != nil {
		t.Fatal(err)
	}

	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	msgProvider, _ := messageprovider.NewFromYAML(`
stats:
  period:
    daily: "Daily"
usage:
  fetch_failed: "Fetch Failed"
  fetch_failed_weekly: "Fetch Failed Weekly"
  fetch_failed_monthly: "Fetch Failed Monthly"
  header_today: "Today {label}"
  header_weekly: "Weekly {days}"
  header_monthly: "Monthly {days}"
  label_date: "Date: {date}"
  label_input_output: "IO: {input}/{output}"
  label_reasoning: "Reasoning: {reasoning}"
  label_total: "Total: {total}"
  label_request_count: "Reqs: {count}"
  label_sum: "Sum"
  label_input: "Input: {input}"
  label_output: "Output: {output}"
  label_daily_summary: "Day: {date} {total} {count}"
  label_cost_header: "Cost({model})"
  label_cost_value: "CostValue: {cost}"
  label_exchange_rate: "Rate: {rate}"
error:
  no_permission: "Permission Denied"
`)

	exchangeRate := fixedExchangeRateService{rate: 1400.0}
	env.handler = NewUsageHandler(
		[]string{"admin1"},
		llmClient,
		msgProvider,
		exchangeRate,
		logger,
	)

	return env
}

func (e *usageTestEnv) teardown() {
	e.ts.Close()
}

func TestUsageHandler_Permission(t *testing.T) {
	env := setupUsageTestEnv(t)
	defer env.teardown()

	ctx := context.Background()

	// 1. Non-admin
	resp, err := env.handler.Handle(ctx, "chat1", "user1", qmodel.UsagePeriodToday, nil)
	if err != nil {
		t.Errorf("Handle failed: %v", err)
	}
	if !strings.Contains(resp, "Permission Denied") {
		t.Errorf("expected permission denied, got: %s", resp)
	}
}

func TestUsageHandler_Today(t *testing.T) {
	env := setupUsageTestEnv(t)
	defer env.teardown()

	ctx := context.Background()

	resp, err := env.handler.Handle(ctx, "chat1", "admin1", qmodel.UsagePeriodToday, nil)
	if err != nil {
		t.Errorf("Handle failed: %v", err)
	}
	if !strings.Contains(resp, "Today Daily") {
		t.Errorf("expected today usage header, got: %s", resp)
	}
	if !strings.Contains(resp, "Total: 2,000,000") {
		t.Errorf("expected total 2,000,000, got: %s", resp)
	}
	if !strings.Contains(resp, "Cost(3.0 Flash)") {
		t.Errorf("expected cost model, got: %s", resp)
	}
	if !strings.Contains(resp, "CostValue: ₩4,900") {
		t.Errorf("expected cost value, got: %s", resp)
	}
	if !strings.Contains(resp, "Rate: 1 USD = 1,400 KRW") {
		t.Errorf("expected rate info, got: %s", resp)
	}
}

func TestUsageHandler_Weekly(t *testing.T) {
	env := setupUsageTestEnv(t)
	defer env.teardown()
	ctx := context.Background()

	resp, err := env.handler.Handle(ctx, "chat1", "admin1", qmodel.UsagePeriodWeekly, nil)
	if err != nil {
		t.Errorf("Handle failed: %v", err)
	}
	if !strings.Contains(resp, "Weekly 7") {
		t.Errorf("expected weekly usage header, got: %s", resp)
	}
	if !strings.Contains(resp, "Total: 21,000,000") {
		t.Errorf("expected total 21,000,000, got: %s", resp)
	}
	if !strings.Contains(resp, "Cost(3.0 Flash)") {
		t.Errorf("expected cost model, got: %s", resp)
	}
	if !strings.Contains(resp, "CostValue: ₩63,700") {
		t.Errorf("expected cost value, got: %s", resp)
	}
}

func TestUsageHandler_Monthly(t *testing.T) {
	env := setupUsageTestEnv(t)
	defer env.teardown()
	ctx := context.Background()

	resp, err := env.handler.Handle(ctx, "chat1", "admin1", qmodel.UsagePeriodMonthly, nil)
	if err != nil {
		t.Errorf("Handle failed: %v", err)
	}
	if !strings.Contains(resp, "Monthly 30") {
		t.Errorf("expected monthly usage header, got: %s", resp)
	}
	if !strings.Contains(resp, "Total: 3,000,000") {
		t.Errorf("expected total 3,000,000, got: %s", resp)
	}
	if !strings.Contains(resp, "Cost(3.0 Flash)") {
		t.Errorf("expected cost model, got: %s", resp)
	}
	if !strings.Contains(resp, "CostValue: ₩9,100") {
		t.Errorf("expected cost value, got: %s", resp)
	}
}

func TestUsageHandler_ModelOverride(t *testing.T) {
	env := setupUsageTestEnv(t)
	defer env.teardown()
	ctx := context.Background()

	env.mocks.daily = &llmrest.DailyUsageResponse{
		UsageDate:    "2023-01-01",
		InputTokens:  1_000_000,
		OutputTokens: 1_000_000,
		TotalTokens:  2_000_000,
		RequestCount: 10,
	}

	override := "pro-30"
	resp, err := env.handler.Handle(ctx, "chat1", "admin1", qmodel.UsagePeriodToday, &override)
	if err != nil {
		t.Errorf("Handle failed: %v", err)
	}
	if !strings.Contains(resp, "Cost(3.0 Pro)") {
		t.Errorf("expected override model, got: %s", resp)
	}
	if !strings.Contains(resp, "CostValue: ₩19,600") {
		t.Errorf("expected override cost, got: %s", resp)
	}
}

func TestUsageHandler_Errors(t *testing.T) {
	// Setup server that returns 500
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer ts.Close()

	llmClient, _ := llmrest.New(llmrest.Config{BaseURL: ts.URL})
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	msgProvider, _ := messageprovider.NewFromYAML(`
usage:
  fetch_failed: "Fetch Failed"
  fetch_failed_weekly: "Fetch Failed"
  fetch_failed_monthly: "Fetch Failed"
`)

	exchangeRate := fixedExchangeRateService{rate: 1400.0}
	handler := NewUsageHandler([]string{"admin1"}, llmClient, msgProvider, exchangeRate, logger)
	ctx := context.Background()

	t.Run("TodayError", func(t *testing.T) {
		resp, err := handler.Handle(ctx, "c", "admin1", qmodel.UsagePeriodToday, nil)
		if err != nil {
			t.Fatal(err)
		}
		if !strings.Contains(resp, "Fetch Failed") {
			t.Error("expected fetch failed msg")
		}
	})

	t.Run("WeeklyError", func(t *testing.T) {
		resp, err := handler.Handle(ctx, "c", "admin1", qmodel.UsagePeriodWeekly, nil)
		if err != nil {
			t.Fatal(err)
		}
		if !strings.Contains(resp, "Fetch Failed") {
			t.Error("expected fetch failed msg")
		}
	})

	t.Run("MonthlyError", func(t *testing.T) {
		resp, err := handler.Handle(ctx, "c", "admin1", qmodel.UsagePeriodMonthly, nil)
		if err != nil {
			t.Fatal(err)
		}
		if !strings.Contains(resp, "Fetch Failed") {
			t.Error("expected fetch failed msg")
		}
	})
}
