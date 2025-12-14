package party.qwer.twentyq.bridge.handlers

import org.slf4j.LoggerFactory
import org.springframework.stereotype.Component
import party.qwer.twentyq.config.AppProperties
import party.qwer.twentyq.config.properties.GeminiModel
import party.qwer.twentyq.model.UsagePeriod
import party.qwer.twentyq.rest.LlmRestClient
import party.qwer.twentyq.rest.dto.DailyUsageResponse
import party.qwer.twentyq.rest.dto.UsageListResponse
import party.qwer.twentyq.rest.dto.UsageResponse
import party.qwer.twentyq.service.ExchangeRateService
import party.qwer.twentyq.util.common.security.requireAdminOrThrow
import party.qwer.twentyq.util.game.GameMessageProvider
import java.text.NumberFormat
import java.util.Locale

/** 토큰 사용량 조회 핸들러 (MCP 서버 연동) */
@Component
class UsageHandler(
    private val llmRestClient: LlmRestClient,
    private val appProperties: AppProperties,
    private val messageProvider: GameMessageProvider,
    private val exchangeRateService: ExchangeRateService,
) {
    companion object {
        private val log = LoggerFactory.getLogger(UsageHandler::class.java)
        private val NUMBER_FORMAT = NumberFormat.getNumberInstance(Locale.KOREA)
        private const val WEEKLY_DAYS = 7
        private const val MONTHLY_DAYS = 30
    }

    private val defaultModel by lazy { appProperties.pricing.getGeminiModel() }

    suspend fun handle(
        chatId: String,
        userId: String,
        period: UsagePeriod = UsagePeriod.TODAY,
        modelOverride: String? = null,
    ): String {
        log.info("HANDLE_ADMIN_USAGE chatId={}, userId={}, period={}, model={}", chatId, userId, period, modelOverride)
        requireAdminOrThrow(
            adminUserIds = appProperties.admin.userIds,
            userId = userId,
            chatId = chatId,
            logger = log,
            warnMessage = "USAGE_PERMISSION_DENIED chatId={}, userId={}",
        )

        val overrideModel = modelOverride?.let { GeminiModel.fromString(it) }

        return when (period) {
            UsagePeriod.TODAY -> buildTodayReport(overrideModel)
            UsagePeriod.WEEKLY -> buildWeeklyReport(overrideModel)
            UsagePeriod.MONTHLY -> buildMonthlyReport(overrideModel)
        }
    }

    private suspend fun buildTodayReport(overrideModel: GeminiModel?): String {
        val today = llmRestClient.getDailyUsage()
        return if (today != null) {
            val model = resolveModel(overrideModel, today.model)
            buildDailySection(messageProvider.get("stats.period.daily"), today, model)
        } else {
            messageProvider.get("usage.fetch_failed")
        }
    }

    private suspend fun buildWeeklyReport(overrideModel: GeminiModel?): String {
        val weekly = llmRestClient.getRecentUsage(WEEKLY_DAYS)
        return if (weekly != null) {
            val model = resolveModel(overrideModel, weekly.model)
            buildWeeklySection(weekly, model)
        } else {
            messageProvider.get("usage.fetch_failed_weekly")
        }
    }

    private suspend fun buildMonthlyReport(overrideModel: GeminiModel?): String {
        val monthly = llmRestClient.getTotalUsageFromDb(MONTHLY_DAYS)
        return if (monthly != null) {
            val model = resolveModel(overrideModel, monthly.model)
            buildMonthlySection(monthly, model)
        } else {
            messageProvider.get("usage.fetch_failed_monthly")
        }
    }

    private suspend fun buildDailySection(
        label: String,
        usage: DailyUsageResponse,
        model: GeminiModel,
    ): String =
        buildString {
            appendLine(messageProvider.get("usage.header_today", "label" to label))
            appendLine()
            appendLine(messageProvider.get("usage.label_date", "date" to usage.usageDate))
            appendLine(
                messageProvider.get(
                    "usage.label_input_output",
                    "input" to format(usage.inputTokens),
                    "output" to format(usage.outputTokens),
                ),
            )
            if (usage.reasoningTokens > 0) {
                appendLine(messageProvider.get("usage.label_reasoning", "reasoning" to format(usage.reasoningTokens)))
            }
            appendLine(messageProvider.get("usage.label_total", "total" to format(usage.totalTokens)))
            appendLine(messageProvider.get("usage.label_request_count", "count" to format(usage.requestCount)))
            appendLine()
            appendCostSection(usage.inputTokens, usage.outputTokens, usage.reasoningTokens, model)
        }

    private suspend fun buildWeeklySection(
        usage: UsageListResponse,
        model: GeminiModel,
    ): String =
        buildString {
            appendLine(messageProvider.get("usage.header_weekly", "days" to WEEKLY_DAYS))
            appendLine()

            usage.usages.filter { it.requestCount > 0 }.forEach { day ->
                appendLine(
                    messageProvider.get(
                        "usage.label_daily_summary",
                        "date" to day.usageDate,
                        "total" to format(day.totalTokens),
                        "count" to format(day.requestCount),
                    ),
                )
            }
            appendLine()
            appendLine(messageProvider.get("usage.label_sum"))
            appendLine(messageProvider.get("usage.label_input", "input" to format(usage.totalInputTokens)))
            appendLine(messageProvider.get("usage.label_output", "output" to format(usage.totalOutputTokens)))
            appendLine(messageProvider.get("usage.label_total", "total" to format(usage.totalTokens)))
            appendLine(messageProvider.get("usage.label_request_count", "count" to format(usage.totalRequestCount)))
            appendLine()
            val totalReasoning = usage.usages.sumOf { it.reasoningTokens }
            appendCostSection(usage.totalInputTokens, usage.totalOutputTokens, totalReasoning, model)
        }

    private suspend fun buildMonthlySection(
        usage: UsageResponse,
        model: GeminiModel,
    ): String =
        buildString {
            appendLine(messageProvider.get("usage.header_monthly", "days" to MONTHLY_DAYS))
            appendLine()
            appendLine(messageProvider.get("usage.label_input", "input" to format(usage.inputTokens.toLong())))
            appendLine(messageProvider.get("usage.label_output", "output" to format(usage.outputTokens.toLong())))
            if ((usage.reasoningTokens ?: 0) > 0) {
                appendLine(
                    messageProvider.get(
                        "usage.label_reasoning",
                        "reasoning" to format(usage.reasoningTokens?.toLong() ?: 0),
                    ),
                )
            }
            appendLine(messageProvider.get("usage.label_total", "total" to format(usage.totalTokens.toLong())))
            appendLine()
            appendCostSection(
                usage.inputTokens.toLong(),
                usage.outputTokens.toLong(),
                usage.reasoningTokens?.toLong() ?: 0,
                model,
            )
        }

    private suspend fun StringBuilder.appendCostSection(
        inputTokens: Long,
        outputTokens: Long,
        reasoningTokens: Long,
        model: GeminiModel,
    ) {
        val costUsd = model.calculateCostUsd(inputTokens, outputTokens, reasoningTokens)
        val costKrw = exchangeRateService.usdToKrw(costUsd)
        val rateInfo = exchangeRateService.getRateInfo()

        appendLine(messageProvider.get("usage.label_cost_header", "model" to model.displayName))
        appendLine(messageProvider.get("usage.label_cost_value", "cost" to formatKrw(costKrw)))
        appendLine(messageProvider.get("usage.label_exchange_rate", "rate" to rateInfo))
    }

    private fun format(value: Long): String = NUMBER_FORMAT.format(value)

    private fun formatKrw(value: Double): String = "₩${NUMBER_FORMAT.format(value.toLong())}"

    private fun resolveModel(
        overrideModel: GeminiModel?,
        serverModel: String?,
    ): GeminiModel = overrideModel ?: serverModel?.let { GeminiModel.fromString(it) } ?: defaultModel
}
