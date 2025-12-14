package party.qwer.twentyq.service.riddle

import io.mockk.clearAllMocks
import io.mockk.coEvery
import io.mockk.coVerify
import io.mockk.impl.annotations.MockK
import io.mockk.junit5.MockKExtension
import kotlinx.coroutines.test.runTest
import org.junit.jupiter.api.AfterEach
import org.junit.jupiter.api.Assertions.assertEquals
import org.junit.jupiter.api.Assertions.assertNotNull
import org.junit.jupiter.api.Assertions.assertNull
import org.junit.jupiter.api.BeforeEach
import org.junit.jupiter.api.Test
import org.junit.jupiter.api.extension.ExtendWith
import party.qwer.twentyq.mcp.TwentyQAnswerResponse
import party.qwer.twentyq.model.FiveScaleKo
import party.qwer.twentyq.model.SecretForHint
import party.qwer.twentyq.rest.TwentyQRestClient

@ExtendWith(MockKExtension::class)
class AnswerLlmExecutorTest {
    @MockK
    private lateinit var restClient: TwentyQRestClient

    private lateinit var executor: AnswerLlmExecutor

    @BeforeEach
    fun setUp() {
        executor = AnswerLlmExecutor(restClient)
    }

    @AfterEach
    fun tearDown() {
        clearAllMocks()
    }

    @Test
    fun `askScale returns scale from MCP response`() =
        runTest {
            val secret = SecretForHint(target = "스마트폰", category = "사물")
            val response = TwentyQAnswerResponse(scale = "예", rawText = "예", thoughtSignature = "sig123")

            coEvery { restClient.answerQuestion(any(), any(), any(), any(), any()) } returns response

            val result = executor.askScale("chat-1", secret, "전자기기인가요?")

            assertEquals(FiveScaleKo.ALWAYS_YES, result.scale)
            assertEquals("sig123", result.thoughtSignature)

            coVerify {
                restClient.answerQuestion(
                    chatId = "chat-1",
                    target = "스마트폰",
                    category = "사물",
                    question = "전자기기인가요?",
                    details = null,
                )
            }
        }

    @Test
    fun `askScale returns null scale on MCP error`() =
        runTest {
            val secret = SecretForHint(target = "고양이", category = "동물")
            val response = TwentyQAnswerResponse(isError = true, errorMessage = "Connection failed")

            coEvery { restClient.answerQuestion(any(), any(), any(), any(), any()) } returns response

            val result = executor.askScale("chat-2", secret, "포유류인가요?")

            assertNull(result.scale)
            assertNull(result.thoughtSignature)
        }

    @Test
    fun `askScale passes details to MCP`() =
        runTest {
            val details = mapOf("type" to "전자기기", "brand" to "삼성")
            val secret = SecretForHint(target = "갤럭시", category = "사물", details = details)
            val response = TwentyQAnswerResponse(scale = "예", rawText = "예")

            coEvery { restClient.answerQuestion(any(), any(), any(), any(), any()) } returns response

            val result = executor.askScale("chat-3", secret, "휴대폰인가요?")

            assertNotNull(result.scale)
            coVerify {
                restClient.answerQuestion(
                    chatId = "chat-3",
                    target = "갤럭시",
                    category = "사물",
                    question = "휴대폰인가요?",
                    details = details,
                )
            }
        }
}
