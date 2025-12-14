package party.qwer.twentyq.service.riddle

import io.mockk.clearAllMocks
import io.mockk.coEvery
import io.mockk.impl.annotations.MockK
import kotlinx.coroutines.test.runTest
import org.junit.jupiter.api.AfterEach
import org.junit.jupiter.api.Assertions.assertEquals
import org.junit.jupiter.api.BeforeEach
import org.junit.jupiter.api.Test
import org.junit.jupiter.api.extension.ExtendWith
import party.qwer.twentyq.mcp.TwentyQVerifyResponse
import party.qwer.twentyq.rest.TwentyQRestClient

@ExtendWith(io.mockk.junit5.MockKExtension::class)
class SimilarityVerifierTest {
    @MockK
    private lateinit var restClient: TwentyQRestClient

    private lateinit var verifier: SimilarityVerifier

    @BeforeEach
    fun setUp() {
        verifier = SimilarityVerifier(restClient)
    }

    @AfterEach
    fun tearDown() {
        clearAllMocks()
    }

    @Test
    fun `should return ACCEPT when MCP returns ACCEPT`() =
        runTest {
            coEvery { restClient.verifyGuess(any(), any()) } returns
                TwentyQVerifyResponse(result = "ACCEPT", rawText = "ACCEPT")

            val result = verifier.verifyHighSimilarity("휴대폰", "핸드폰")

            assertEquals(VerifyAnswerResponse.RESPONSE_ACCEPT, result)
        }

    @Test
    fun `should return CLOSE when MCP returns CLOSE`() =
        runTest {
            coEvery { restClient.verifyGuess(any(), any()) } returns
                TwentyQVerifyResponse(result = "CLOSE", rawText = "CLOSE")

            val result = verifier.verifyHighSimilarity("운동화", "신발")

            assertEquals(VerifyAnswerResponse.RESPONSE_CLOSE, result)
        }

    @Test
    fun `should return REJECT when MCP returns REJECT`() =
        runTest {
            coEvery { restClient.verifyGuess(any(), any()) } returns
                TwentyQVerifyResponse(result = "REJECT", rawText = "REJECT")

            val result = verifier.verifyHighSimilarity("사자", "호랑이")

            assertEquals(VerifyAnswerResponse.RESPONSE_REJECT, result)
        }

    @Test
    fun `should return REJECT when MCP throws exception`() =
        runTest {
            coEvery { restClient.verifyGuess(any(), any()) } throws RuntimeException("API Error")

            val result = verifier.verifyHighSimilarity("test", "test")

            assertEquals(VerifyAnswerResponse.RESPONSE_REJECT, result)
        }

    @Test
    fun `should return REJECT when MCP returns error`() =
        runTest {
            coEvery { restClient.verifyGuess(any(), any()) } returns
                TwentyQVerifyResponse(isError = true, errorMessage = "MCP Error")

            val result = verifier.verifyHighSimilarity("test", "test")

            assertEquals(VerifyAnswerResponse.RESPONSE_REJECT, result)
        }

    @Test
    fun `should return REJECT when MCP returns invalid result`() =
        runTest {
            coEvery { restClient.verifyGuess(any(), any()) } returns
                TwentyQVerifyResponse(result = "INVALID", rawText = "INVALID")

            val result = verifier.verifyHighSimilarity("test", "test")

            assertEquals(VerifyAnswerResponse.RESPONSE_REJECT, result)
        }

    @Test
    fun `should handle lowercase result`() =
        runTest {
            coEvery { restClient.verifyGuess(any(), any()) } returns
                TwentyQVerifyResponse(result = "accept", rawText = "accept")

            val result = verifier.verifyHighSimilarity("휴대폰", "핸드폰")

            assertEquals(VerifyAnswerResponse.RESPONSE_ACCEPT, result)
        }

    @Test
    fun `should return REJECT when result is null`() =
        runTest {
            coEvery { restClient.verifyGuess(any(), any()) } returns
                TwentyQVerifyResponse(result = null, rawText = "")

            val result = verifier.verifyHighSimilarity("test", "test")

            assertEquals(VerifyAnswerResponse.RESPONSE_REJECT, result)
        }
}
