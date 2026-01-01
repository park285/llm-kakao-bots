package service

import (
	"context"
	"log/slog"
	"os"
	"strings"
	"testing"

	"google.golang.org/grpc"

	"github.com/park285/llm-kakao-bots/game-bot-go/internal/common/llmrest"
	llmv1 "github.com/park285/llm-kakao-bots/game-bot-go/internal/common/llmrest/pb/llm/v1"
	"github.com/park285/llm-kakao-bots/game-bot-go/internal/common/messageprovider"
	"github.com/park285/llm-kakao-bots/game-bot-go/internal/common/testhelper"
	qmodel "github.com/park285/llm-kakao-bots/game-bot-go/internal/twentyq/model"
)

type usageTestEnv struct {
	handler   *UsageHandler
	llmClient *llmrest.Client
	stopLLM   func()
	mocks     struct {
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
	stub := &twentyqLLMGRPCStub{
		getDailyUsage: func() (*llmv1.DailyUsageResponse, error) {
			if env.mocks.daily != nil {
				serverModel := ""
				if env.mocks.daily.Model != nil {
					serverModel = *env.mocks.daily.Model
				}
				return &llmv1.DailyUsageResponse{
					UsageDate:       env.mocks.daily.UsageDate,
					InputTokens:     env.mocks.daily.InputTokens,
					OutputTokens:    env.mocks.daily.OutputTokens,
					TotalTokens:     env.mocks.daily.TotalTokens,
					ReasoningTokens: env.mocks.daily.ReasoningTokens,
					RequestCount:    env.mocks.daily.RequestCount,
					Model:           serverModel,
				}, nil
			}
			return &llmv1.DailyUsageResponse{
				UsageDate:       "2023-01-01",
				InputTokens:     1_000_000,
				OutputTokens:    1_000_000,
				TotalTokens:     2_000_000,
				ReasoningTokens: 0,
				RequestCount:    10,
				Model:           model,
			}, nil
		},
		getRecentUsage: func(_ *llmv1.GetRecentUsageRequest) (*llmv1.UsageListResponse, error) {
			if env.mocks.weekly != nil {
				serverModel := ""
				if env.mocks.weekly.Model != nil {
					serverModel = *env.mocks.weekly.Model
				}
				usages := make([]*llmv1.DailyUsageResponse, 0, len(env.mocks.weekly.Usages))
				for _, item := range env.mocks.weekly.Usages {
					itemModel := ""
					if item.Model != nil {
						itemModel = *item.Model
					}
					usages = append(usages, &llmv1.DailyUsageResponse{
						UsageDate:       item.UsageDate,
						InputTokens:     item.InputTokens,
						OutputTokens:    item.OutputTokens,
						TotalTokens:     item.TotalTokens,
						ReasoningTokens: item.ReasoningTokens,
						RequestCount:    item.RequestCount,
						Model:           itemModel,
					})
				}
				return &llmv1.UsageListResponse{
					Usages:            usages,
					TotalInputTokens:  env.mocks.weekly.TotalInputTokens,
					TotalOutputTokens: env.mocks.weekly.TotalOutputTokens,
					TotalTokens:       env.mocks.weekly.TotalTokens,
					TotalRequestCount: env.mocks.weekly.TotalRequestCount,
					Model:             serverModel,
				}, nil
			}
			return &llmv1.UsageListResponse{
				Usages:            nil,
				TotalInputTokens:  7_000_000,
				TotalOutputTokens: 14_000_000,
				TotalTokens:       21_000_000,
				TotalRequestCount: 0,
				Model:             model,
			}, nil
		},
		getTotalUsage: func(_ *llmv1.GetTotalUsageRequest) (*llmv1.UsageResponse, error) {
			if env.mocks.monthly != nil {
				serverModel := ""
				if env.mocks.monthly.Model != nil {
					serverModel = *env.mocks.monthly.Model
				}
				reasoning := int64(0)
				if env.mocks.monthly.ReasoningTokens != nil {
					reasoning = int64(*env.mocks.monthly.ReasoningTokens)
				}
				return &llmv1.UsageResponse{
					InputTokens:     int64(env.mocks.monthly.InputTokens),
					OutputTokens:    int64(env.mocks.monthly.OutputTokens),
					TotalTokens:     int64(env.mocks.monthly.TotalTokens),
					ReasoningTokens: reasoning,
					Model:           serverModel,
				}, nil
			}
			return &llmv1.UsageResponse{
				InputTokens:     1_000_000,
				OutputTokens:    2_000_000,
				TotalTokens:     3_000_000,
				ReasoningTokens: 0,
				Model:           model,
			}, nil
		},
	}

	baseURL, stop := testhelper.StartTestGRPCServer(t, func(s *grpc.Server) {
		llmv1.RegisterLLMServiceServer(s, stub)
	})
	env.stopLLM = stop

	llmClient, err := llmrest.New(llmrest.Config{BaseURL: baseURL})
	if err != nil {
		t.Fatal(err)
	}
	env.llmClient = llmClient
	t.Cleanup(func() {
		_ = llmClient.Close()
	})

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
	if e.llmClient != nil {
		_ = e.llmClient.Close()
	}
	if e.stopLLM != nil {
		e.stopLLM()
	}
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
	stubErr := &twentyqLLMGRPCStub{hasError: func() bool { return true }}
	baseURL, _ := testhelper.StartTestGRPCServer(t, func(s *grpc.Server) {
		llmv1.RegisterLLMServiceServer(s, stubErr)
	})
	llmClient, err := llmrest.New(llmrest.Config{BaseURL: baseURL})
	if err != nil {
		t.Fatalf("llm client init failed: %v", err)
	}
	t.Cleanup(func() {
		_ = llmClient.Close()
	})
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
