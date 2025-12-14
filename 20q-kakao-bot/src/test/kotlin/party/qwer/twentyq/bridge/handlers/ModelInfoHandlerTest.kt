package party.qwer.twentyq.bridge.handlers

import io.mockk.coEvery
import io.mockk.every
import io.mockk.mockk
import kotlinx.coroutines.test.runTest
import org.assertj.core.api.Assertions.assertThat
import org.junit.jupiter.api.Nested
import org.junit.jupiter.api.Test
import party.qwer.twentyq.rest.LlmRestClient
import party.qwer.twentyq.util.game.GameMessageProvider

/**
 * ModelInfoHandler 테스트
 * - 컨벤션: logger companion object 내부 배치
 * - 하드코딩 메시지 제거 (GameMessageProvider 사용)
 */
class ModelInfoHandlerTest {
    private val llmRestClient = mockk<LlmRestClient>()
    private val messageProvider = mockk<GameMessageProvider>(relaxed = true)

    private val handler =
        ModelInfoHandler(
            llmRestClient,
            messageProvider,
        )

    @Nested
    inner class HandleTests {
        @Test
        fun `should return fetch failed message when config is null`() =
            runTest {
                // Given
                coEvery { llmRestClient.getModelConfig() } returns null
                every { messageProvider.get("model_info.fetch_failed") } returns "모델 정보를 가져오지 못했습니다."

                // When
                val result = handler.handle()

                // Then
                assertThat(result).isEqualTo("모델 정보를 가져오지 못했습니다.")
            }

        @Test
        fun `should return fetch failed message when exception occurs`() =
            runTest {
                // Given
                coEvery { llmRestClient.getModelConfig() } throws RuntimeException("Connection failed")
                every { messageProvider.get("model_info.fetch_failed") } returns "모델 정보를 가져오지 못했습니다."

                // When
                val result = handler.handle()

                // Then
                assertThat(result).isEqualTo("모델 정보를 가져오지 못했습니다.")
            }

        @Test
        fun `should build model info with messageProvider when config exists`() =
            runTest {
                // Given
                val config =
                    LlmRestClient.ModelConfigResponse(
                        modelDefault = "gemini-2.5-flash",
                        modelHints = "gemini-2.5-pro",
                        modelAnswer = null,
                        modelVerify = null,
                        temperature = 0.7,
                        timeoutSeconds = 30,
                        maxRetries = 3,
                        http2Enabled = true,
                    )
                coEvery { llmRestClient.getModelConfig() } returns config
                every { messageProvider.get("model_info.header") } returns "모델 설정"
                every { messageProvider.get("model_info.default", "model" to "gemini-2.5-flash") } returns "- 기본: gemini-2.5-flash"
                every { messageProvider.get("model_info.hints", "model" to "gemini-2.5-pro") } returns "- 힌트: gemini-2.5-pro"
                every { messageProvider.get("model_info.answer", "model" to "gemini-2.5-flash") } returns "- 답변: gemini-2.5-flash"
                every { messageProvider.get("model_info.verify", "model" to "gemini-2.5-flash") } returns "- 검증: gemini-2.5-flash"
                every { messageProvider.get("model_info.temperature", "value" to 0.7) } returns "- 온도: 0.7"
                every { messageProvider.get("model_info.max_retries", "value" to 3) } returns "- 재시도: 3"
                every { messageProvider.get("model_info.timeout", "value" to 30) } returns "- 타임아웃: 30s"
                every { messageProvider.get("model_info.transport", "mode" to "H2C") } returns "- 전송: H2C"

                // When
                val result = handler.handle()

                // Then
                assertThat(result).contains("모델 설정")
                assertThat(result).contains("- 기본: gemini-2.5-flash")
                assertThat(result).contains("- 힌트: gemini-2.5-pro")
                assertThat(result).contains("- 온도: 0.7")
                assertThat(result).contains("- 전송: H2C")
            }

        @Test
        fun `should use default model when hints is null`() =
            runTest {
                // Given
                val config =
                    LlmRestClient.ModelConfigResponse(
                        modelDefault = "gemini-2.5-flash",
                        modelHints = null,
                        modelAnswer = null,
                        modelVerify = null,
                        temperature = 0.5,
                        timeoutSeconds = 60,
                        maxRetries = 5,
                        http2Enabled = false,
                    )
                coEvery { llmRestClient.getModelConfig() } returns config
                every { messageProvider.get(any(), *anyVararg()) } answers {
                    val key = firstArg<String>()
                    when {
                        key == "model_info.hints" -> {
                            val args = secondArg<Array<Pair<String, Any>>>()
                            val model = args.find { it.first == "model" }?.second
                            "- 힌트: $model"
                        }
                        key == "model_info.transport" -> "- 전송: HTTP/1.1"
                        else -> key
                    }
                }

                // When
                val result = handler.handle()

                // Then
                // hints가 null이면 modelDefault 사용
                assertThat(result).contains("- 힌트: gemini-2.5-flash")
                assertThat(result).contains("- 전송: HTTP/1.1")
            }
    }
}
