package party.qwer.twentyq.service.riddle

import io.mockk.coEvery
import io.mockk.coVerify
import io.mockk.impl.annotations.MockK
import io.mockk.junit5.MockKExtension
import kotlinx.coroutines.test.runTest
import org.junit.jupiter.api.Assertions.assertEquals
import org.junit.jupiter.api.Assertions.fail
import org.junit.jupiter.api.BeforeEach
import org.junit.jupiter.api.Test
import org.junit.jupiter.api.extension.ExtendWith
import party.qwer.twentyq.mcp.TwentyQHintsResponse
import party.qwer.twentyq.model.RiddleSecret
import party.qwer.twentyq.redis.RiddleSessionRepository
import party.qwer.twentyq.rest.TwentyQRestClient
import party.qwer.twentyq.service.exception.HintLimitExceededException
import tools.jackson.databind.json.JsonMapper

@ExtendWith(MockKExtension::class)
class HintGeneratorTest {
    @MockK
    private lateinit var sessionRepo: RiddleSessionRepository

    @MockK
    private lateinit var restClient: TwentyQRestClient

    private lateinit var generator: HintGenerator

    @BeforeEach
    fun setUp() {
        val objectMapper = JsonMapper.builder().build()
        generator =
            HintGenerator(
                sessionRepo = sessionRepo,
                restClient = restClient,
                objectMapper = objectMapper,
            )
    }

    @Test
    fun `generateHints should fail when hint limit exceeded`() =
        runTest {
            val chatId = "chat"
            val secret = RiddleSecret(target = "target", category = "ANY", intro = "intro")

            coEvery { sessionRepo.getSecret(chatId = chatId) } returns secret
            coEvery { sessionRepo.getHintCount(chatId) } returns 10
            coEvery { sessionRepo.getHistory(chatId) } returns emptyList()
            coEvery { sessionRepo.getSelectedCategory(chatId) } returns null

            try {
                generator.generateHints(chatId, count = 1)
                fail("expected HintLimitExceededException")
            } catch (e: HintLimitExceededException) {
                // expected
            }
        }

    @Test
    fun `generateHints should call MCP and save hints`() =
        runTest {
            val chatId = "chat"
            val secret = RiddleSecret(target = "스마트폰", category = "사물", intro = "intro")
            val mcpResponse =
                TwentyQHintsResponse(
                    hints = listOf("손에 들고 다니는 작은 기계"),
                    thoughtSignature = "sig123",
                )

            coEvery { sessionRepo.getSecret(chatId = chatId) } returns secret
            coEvery { sessionRepo.getHintCount(chatId) } returns 0
            coEvery { sessionRepo.getHistory(chatId) } returns emptyList()
            coEvery { sessionRepo.getSelectedCategory(chatId) } returns "사물"
            coEvery { restClient.generateHints(any(), any(), any()) } returns mcpResponse
            coEvery { sessionRepo.incrementHintCount(any()) } returns Unit
            coEvery { sessionRepo.addHistory(any(), any(), any(), any(), any(), any()) } returns Unit

            val result = generator.generateHints(chatId, count = 1)

            assertEquals(listOf("손에 들고 다니는 작은 기계"), result)
            coVerify {
                restClient.generateHints(
                    target = "스마트폰",
                    category = "사물",
                    details = null,
                )
            }
            coVerify { sessionRepo.incrementHintCount(chatId) }
            coVerify {
                sessionRepo.addHistory(
                    chatId = chatId,
                    questionNumber = -1,
                    question = "힌트 #1",
                    answer = "손에 들고 다니는 작은 기계",
                    thoughtSignature = "sig123",
                    userId = null,
                )
            }
        }

    @Test
    fun `generateHints should return empty when MCP returns error`() =
        runTest {
            val chatId = "chat"
            val secret = RiddleSecret(target = "target", category = "ANY", intro = "intro")
            val mcpResponse = TwentyQHintsResponse(isError = true, errorMessage = "MCP error")

            coEvery { sessionRepo.getSecret(chatId = chatId) } returns secret
            coEvery { sessionRepo.getHintCount(chatId) } returns 0
            coEvery { sessionRepo.getHistory(chatId) } returns emptyList()
            coEvery { sessionRepo.getSelectedCategory(chatId) } returns null

            coEvery { restClient.generateHints(any(), any(), any()) } returns mcpResponse

            val result = generator.generateHints(chatId, count = 1)

            assertEquals(emptyList<String>(), result)
        }

    @Test
    fun `generateHints should return empty when MCP returns empty hints`() =
        runTest {
            val chatId = "chat"
            val secret = RiddleSecret(target = "target", category = "ANY", intro = "intro")
            val mcpResponse = TwentyQHintsResponse(hints = emptyList())

            coEvery { sessionRepo.getSecret(chatId = chatId) } returns secret
            coEvery { sessionRepo.getHintCount(chatId) } returns 0
            coEvery { sessionRepo.getHistory(chatId) } returns emptyList()
            coEvery { sessionRepo.getSelectedCategory(chatId) } returns null

            coEvery { restClient.generateHints(any(), any(), any()) } returns mcpResponse

            val result = generator.generateHints(chatId, count = 1)

            assertEquals(emptyList<String>(), result)
        }
}
