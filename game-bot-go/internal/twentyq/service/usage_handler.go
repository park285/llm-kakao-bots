package service

import (
	"context"
	"log/slog"
	"strings"

	"golang.org/x/text/language"
	"golang.org/x/text/message"

	"github.com/park285/llm-kakao-bots/game-bot-go/internal/common/llmrest"
	"github.com/park285/llm-kakao-bots/game-bot-go/internal/common/messageprovider"
	qmessages "github.com/park285/llm-kakao-bots/game-bot-go/internal/twentyq/messages"
	qmodel "github.com/park285/llm-kakao-bots/game-bot-go/internal/twentyq/model"
)

const (
	weeklyDays  = 7
	monthlyDays = 30
)

// UsageHandler 토큰 사용량 조회 핸들러.
type UsageHandler struct {
	adminUserIDs []string
	llmClient    *llmrest.Client
	msgProvider  *messageprovider.Provider
	exchangeRate ExchangeRateService
	logger       *slog.Logger
	numPrinter   *message.Printer
}

// NewUsageHandler 생성자.
func NewUsageHandler(
	adminUserIDs []string,
	llmClient *llmrest.Client,
	msgProvider *messageprovider.Provider,
	exchangeRate ExchangeRateService,
	logger *slog.Logger,
) *UsageHandler {
	if exchangeRate == nil {
		exchangeRate = NewFrankfurterExchangeRateService(logger)
	}
	return &UsageHandler{
		adminUserIDs: adminUserIDs,
		llmClient:    llmClient,
		msgProvider:  msgProvider,
		exchangeRate: exchangeRate,
		logger:       logger,
		numPrinter:   message.NewPrinter(language.Korean),
	}
}

// IsAdmin 관리자 여부 확인.
func (h *UsageHandler) IsAdmin(userID string) bool {
	for _, id := range h.adminUserIDs {
		if id == userID {
			return true
		}
	}
	return false
}

// Handle 사용량 조회.
func (h *UsageHandler) Handle(
	ctx context.Context,
	chatID string,
	userID string,
	period qmodel.UsagePeriod,
	modelOverride *string,
) (string, error) {
	overrideValue := ""
	if modelOverride != nil {
		overrideValue = strings.TrimSpace(*modelOverride)
	}
	h.logger.Info("HANDLE_ADMIN_USAGE",
		"chatID", chatID,
		"userID", userID,
		"period", period,
		"modelOverride", overrideValue,
	)

	if !h.IsAdmin(userID) {
		h.logger.Warn("USAGE_PERMISSION_DENIED", "chatID", chatID, "userID", userID)
		return h.msgProvider.Get(qmessages.ErrorNoPermission), nil
	}

	switch period {
	case qmodel.UsagePeriodToday:
		return h.buildTodayReport(ctx, modelOverride)
	case qmodel.UsagePeriodWeekly:
		return h.buildWeeklyReport(ctx, modelOverride)
	case qmodel.UsagePeriodMonthly:
		return h.buildMonthlyReport(ctx, modelOverride)
	default:
		return h.buildTodayReport(ctx, modelOverride)
	}
}

func (h *UsageHandler) buildTodayReport(ctx context.Context, modelOverride *string) (string, error) {
	usage, err := h.llmClient.GetDailyUsage(ctx, nil)
	if err != nil {
		h.logger.Warn("get_daily_usage_failed", "error", err)
		return h.msgProvider.Get(qmessages.UsageFetchFailed), nil
	}
	if usage == nil {
		return h.msgProvider.Get(qmessages.UsageFetchFailed), nil
	}
	model := ResolveGeminiModel(modelOverride, usage.Model)
	return h.formatDailyUsage(ctx, usage, model), nil
}

func (h *UsageHandler) buildWeeklyReport(ctx context.Context, modelOverride *string) (string, error) {
	usage, err := h.llmClient.GetRecentUsage(ctx, weeklyDays, nil)
	if err != nil {
		h.logger.Warn("get_weekly_usage_failed", "error", err)
		return h.msgProvider.Get(qmessages.UsageFetchFailedWeekly), nil
	}
	if usage == nil {
		return h.msgProvider.Get(qmessages.UsageFetchFailedWeekly), nil
	}
	model := ResolveGeminiModel(modelOverride, usage.Model)
	return h.formatWeeklyUsage(ctx, usage, model), nil
}

func (h *UsageHandler) buildMonthlyReport(ctx context.Context, modelOverride *string) (string, error) {
	usage, err := h.llmClient.GetUsageTotalFromDB(ctx, monthlyDays, nil)
	if err != nil {
		h.logger.Warn("get_monthly_usage_failed", "error", err)
		return h.msgProvider.Get(qmessages.UsageFetchFailedMonthly), nil
	}
	if usage == nil {
		return h.msgProvider.Get(qmessages.UsageFetchFailedMonthly), nil
	}
	model := ResolveGeminiModel(modelOverride, usage.Model)
	return h.formatMonthlyUsage(ctx, usage, model), nil
}

func (h *UsageHandler) formatDailyUsage(ctx context.Context, usage *llmrest.DailyUsageResponse, model GeminiModel) string {
	var sb strings.Builder

	sb.WriteString(h.msgProvider.Get(
		qmessages.UsageHeaderToday,
		messageprovider.P("label", h.msgProvider.Get(qmessages.StatsPeriodDaily)),
	))
	sb.WriteString("\n\n")

	sb.WriteString(h.msgProvider.Get(
		qmessages.UsageLabelDate,
		messageprovider.P("date", usage.UsageDate),
	))
	sb.WriteString("\n")

	sb.WriteString(h.msgProvider.Get(
		qmessages.UsageLabelInputOutput,
		messageprovider.P("input", h.formatNum(usage.InputTokens)),
		messageprovider.P("output", h.formatNum(usage.OutputTokens)),
	))
	sb.WriteString("\n")

	if usage.ReasoningTokens > 0 {
		sb.WriteString(h.msgProvider.Get(
			qmessages.UsageLabelReasoning,
			messageprovider.P("reasoning", h.formatNum(usage.ReasoningTokens)),
		))
		sb.WriteString("\n")
	}

	sb.WriteString(h.msgProvider.Get(
		qmessages.UsageLabelTotal,
		messageprovider.P("total", h.formatNum(usage.TotalTokens)),
	))
	sb.WriteString("\n")

	sb.WriteString(h.msgProvider.Get(
		qmessages.UsageLabelReqCount,
		messageprovider.P("count", h.formatNum(usage.RequestCount)),
	))

	sb.WriteString("\n\n")
	h.appendCostSection(ctx, &sb, model, usage.InputTokens, usage.OutputTokens, usage.ReasoningTokens)

	return sb.String()
}

func (h *UsageHandler) formatWeeklyUsage(ctx context.Context, usage *llmrest.UsageListResponse, model GeminiModel) string {
	var sb strings.Builder

	sb.WriteString(h.msgProvider.Get(
		qmessages.UsageHeaderWeekly,
		messageprovider.P("days", weeklyDays),
	))
	sb.WriteString("\n\n")

	for _, day := range usage.Usages {
		if day.RequestCount > 0 {
			sb.WriteString(h.msgProvider.Get(
				qmessages.UsageLabelDailySummary,
				messageprovider.P("date", day.UsageDate),
				messageprovider.P("total", h.formatNum(day.TotalTokens)),
				messageprovider.P("count", h.formatNum(day.RequestCount)),
			))
			sb.WriteString("\n")
		}
	}

	sb.WriteString("\n")
	sb.WriteString(h.msgProvider.Get(qmessages.UsageLabelSum))
	sb.WriteString("\n")

	sb.WriteString(h.msgProvider.Get(
		qmessages.UsageLabelInput,
		messageprovider.P("input", h.formatNum(usage.TotalInputTokens)),
	))
	sb.WriteString("\n")

	sb.WriteString(h.msgProvider.Get(
		qmessages.UsageLabelOutput,
		messageprovider.P("output", h.formatNum(usage.TotalOutputTokens)),
	))
	sb.WriteString("\n")

	sb.WriteString(h.msgProvider.Get(
		qmessages.UsageLabelTotal,
		messageprovider.P("total", h.formatNum(usage.TotalTokens)),
	))
	sb.WriteString("\n")

	sb.WriteString(h.msgProvider.Get(
		qmessages.UsageLabelReqCount,
		messageprovider.P("count", h.formatNum(usage.TotalRequestCount)),
	))

	var totalReasoning int64
	for _, day := range usage.Usages {
		totalReasoning += day.ReasoningTokens
	}

	sb.WriteString("\n\n")
	h.appendCostSection(ctx, &sb, model, usage.TotalInputTokens, usage.TotalOutputTokens, totalReasoning)

	return sb.String()
}

func (h *UsageHandler) formatMonthlyUsage(ctx context.Context, usage *llmrest.UsageResponse, model GeminiModel) string {
	var sb strings.Builder

	sb.WriteString(h.msgProvider.Get(
		qmessages.UsageHeaderMonthly,
		messageprovider.P("days", monthlyDays),
	))
	sb.WriteString("\n\n")

	sb.WriteString(h.msgProvider.Get(
		qmessages.UsageLabelInput,
		messageprovider.P("input", h.formatNumInt(usage.InputTokens)),
	))
	sb.WriteString("\n")

	sb.WriteString(h.msgProvider.Get(
		qmessages.UsageLabelOutput,
		messageprovider.P("output", h.formatNumInt(usage.OutputTokens)),
	))
	sb.WriteString("\n")

	if usage.ReasoningTokens != nil && *usage.ReasoningTokens > 0 {
		sb.WriteString(h.msgProvider.Get(
			qmessages.UsageLabelReasoning,
			messageprovider.P("reasoning", h.formatNumInt(*usage.ReasoningTokens)),
		))
		sb.WriteString("\n")
	}

	sb.WriteString(h.msgProvider.Get(
		qmessages.UsageLabelTotal,
		messageprovider.P("total", h.formatNumInt(usage.TotalTokens)),
	))

	reasoningTokens := int64(0)
	if usage.ReasoningTokens != nil {
		reasoningTokens = int64(*usage.ReasoningTokens)
	}

	sb.WriteString("\n\n")
	h.appendCostSection(ctx, &sb, model, int64(usage.InputTokens), int64(usage.OutputTokens), reasoningTokens)

	return sb.String()
}

func (h *UsageHandler) formatNum(v int64) string {
	return h.numPrinter.Sprintf("%d", v)
}

func (h *UsageHandler) formatNumInt(v int) string {
	return h.numPrinter.Sprintf("%d", v)
}

func (h *UsageHandler) appendCostSection(
	ctx context.Context,
	sb *strings.Builder,
	model GeminiModel,
	inputTokens int64,
	outputTokens int64,
	reasoningTokens int64,
) {
	costUsd := model.CalculateCostUsd(inputTokens, outputTokens, reasoningTokens)
	costKrw := h.exchangeRate.UsdToKrw(ctx, costUsd)
	rateInfo := h.exchangeRate.RateInfo(ctx)

	sb.WriteString(h.msgProvider.Get(
		qmessages.UsageLabelCostHeader,
		messageprovider.P("model", model.DisplayName()),
	))
	sb.WriteString("\n")

	sb.WriteString(h.msgProvider.Get(
		qmessages.UsageLabelCostValue,
		messageprovider.P("cost", h.formatKrw(costKrw)),
	))
	sb.WriteString("\n")

	sb.WriteString(h.msgProvider.Get(
		qmessages.UsageLabelExchangeRate,
		messageprovider.P("rate", rateInfo),
	))
}

func (h *UsageHandler) formatKrw(value float64) string {
	return "₩" + h.numPrinter.Sprintf("%d", int64(value))
}
