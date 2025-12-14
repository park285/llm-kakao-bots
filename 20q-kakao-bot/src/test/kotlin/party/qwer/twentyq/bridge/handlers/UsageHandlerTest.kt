package party.qwer.twentyq.bridge.handlers

import io.mockk.coEvery
import io.mockk.every
import io.mockk.mockk
import kotlinx.coroutines.test.runTest
import org.assertj.core.api.Assertions.assertThat
import org.junit.jupiter.api.Nested
import org.junit.jupiter.api.Test
import party.qwer.twentyq.config.AppProperties
import party.qwer.twentyq.config.properties.Admin
import party.qwer.twentyq.config.properties.GeminiModel
import party.qwer.twentyq.config.properties.Pricing
import party.qwer.twentyq.model.UsagePeriod
import party.qwer.twentyq.rest.LlmRestClient
import party.qwer.twentyq.rest.dto.DailyUsageResponse
import party.qwer.twentyq.rest.dto.UsageListResponse
import party.qwer.twentyq.rest.dto.UsageResponse
import party.qwer.twentyq.service.ExchangeRateService
import party.qwer.twentyq.util.game.GameMessageProvider

/**
 * UsageHandler 테스트
 */
class UsageHandlerTest {
    private val llmRestClient = mockk<LlmRestClient>()
    private val appProperties = mockk<AppProperties>()
    private val messageProvider = mockk<GameMessageProvider>(relaxed = true)
    private val exchangeRateService = mockk<ExchangeRateService>()

    private val handler =
        UsageHandler(
            llmRestClient,
            appProperties,
            messageProvider,
            exchangeRateService,
        )

    init {
        every { appProperties.admin } returns
            mockk<Admin>().apply {
                every { userIds } returns listOf("admin123")
            }
        every { appProperties.pricing } returns
            mockk<Pricing>().apply {
                every { getGeminiModel() } returns GeminiModel.FLASH_25
            }
        coEvery { exchangeRateService.usdToKrw(any()) } answers { firstArg<Double>() * 1400.0 }
        coEvery { exchangeRateService.getRateInfo() } returns "1 USD = 1,400 KRW"
    }

    @Nested
    inner class TodayReport {
        @Test
        fun `should return fetch failed message when daily usage is null`() =
            runTest {
                // Given
                coEvery { llmRestClient.getDailyUsage() } returns null
                every { messageProvider.get("usage.fetch_failed") } returns "사용량 조회 실패"

                // When
                val result = handler.handle("chat123", "admin123", UsagePeriod.TODAY)

                // Then
                assertThat(result).isEqualTo("사용량 조회 실패")
            }

        @Test
        fun `should build daily section with messageProvider when usage exists`() =
            runTest {
                // Given
                val usage =
                    DailyUsageResponse(
                        usageDate = "2025-12-07",
                        inputTokens = 1000L,
                        outputTokens = 500L,
                        reasoningTokens = 100L,
                        totalTokens = 1600L,
                        requestCount = 10L,
                        model = "flash-25",
                    )
                coEvery { llmRestClient.getDailyUsage() } returns usage
                every { messageProvider.get("stats.period.daily") } returns "오늘"
                every { messageProvider.get(any(), *anyVararg()) } answers {
                    val key = firstArg<String>()
                    when (key) {
                        "usage.header_today" -> "📊 토큰 사용량 (오늘)"
                        "usage.label_date" -> "▸ 날짜: 2025-12-07"
                        "usage.label_input_output" -> "  입력: 1,000 / 출력: 500"
                        "usage.label_reasoning" -> "  추론: 100"
                        "usage.label_total" -> "  총: 1,600"
                        "usage.label_request_count" -> "  요청: 10회"
                        "usage.label_cost_header" -> "▸ 예상 비용 (2.5 Flash 기준)"
                        "usage.label_cost_value" -> "  ~₩2"
                        "usage.label_exchange_rate" -> "  (1 USD = 1,400 KRW)"
                        else -> key
                    }
                }

                // When
                val result = handler.handle("chat123", "admin123", UsagePeriod.TODAY)

                // Then
                assertThat(result).contains("📊 토큰 사용량")
                assertThat(result).contains("날짜")
            }
    }

    @Nested
    inner class WeeklyReport {
        @Test
        fun `should return fetch failed weekly message when usage is null`() =
            runTest {
                // Given
                coEvery { llmRestClient.getRecentUsage(7) } returns null
                every { messageProvider.get("usage.fetch_failed_weekly") } returns "주간 사용량 조회 실패"

                // When
                val result = handler.handle("chat123", "admin123", UsagePeriod.WEEKLY)

                // Then
                assertThat(result).isEqualTo("주간 사용량 조회 실패")
            }
    }

    @Nested
    inner class MonthlyReport {
        @Test
        fun `should return fetch failed monthly message when usage is null`() =
            runTest {
                // Given
                coEvery { llmRestClient.getTotalUsageFromDb(30) } returns null
                every { messageProvider.get("usage.fetch_failed_monthly") } returns "월간 사용량 조회 실패"

                // When
                val result = handler.handle("chat123", "admin123", UsagePeriod.MONTHLY)

                // Then
                assertThat(result).isEqualTo("월간 사용량 조회 실패")
            }
    }

    @Nested
    inner class KrwFormatting {
        @Test
        fun `should format cost in KRW using dynamic exchange rate`() =
            runTest {
                // Given
                val usage =
                    DailyUsageResponse(
                        usageDate = "2025-12-07",
                        inputTokens = 1_000_000L,
                        outputTokens = 100_000L,
                        reasoningTokens = 0L,
                        totalTokens = 1_100_000L,
                        requestCount = 50L,
                        model = "flash-25",
                    )
                coEvery { llmRestClient.getDailyUsage() } returns usage

                var capturedCost: String? = null
                every { messageProvider.get("stats.period.daily") } returns "오늘"
                every { messageProvider.get(eq("usage.label_cost_value"), *anyVararg()) } answers {
                    val args = secondArg<Array<Pair<String, Any>>>()
                    capturedCost = args.find { it.first == "cost" }?.second?.toString()
                    "  ~$capturedCost"
                }
                every { messageProvider.get(neq("usage.label_cost_value"), *anyVararg()) } returns "mock"

                // When
                handler.handle("chat123", "admin123", UsagePeriod.TODAY)

                // Then
                // Flash 2.5: input $0.30/1M, output $2.50/1M
                // 1M input = $0.30, 100K output = $0.25, total = $0.55
                // $0.55 * 1400 = ₩770
                assertThat(capturedCost).startsWith("₩")
                assertThat(capturedCost).contains("770")
            }
    }
}
