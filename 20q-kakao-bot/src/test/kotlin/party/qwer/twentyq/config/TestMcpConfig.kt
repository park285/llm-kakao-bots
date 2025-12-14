package party.qwer.twentyq.config

import io.ktor.client.HttpClient
import io.mockk.coEvery
import io.mockk.mockk
import org.springframework.context.annotation.Bean
import org.springframework.context.annotation.Configuration
import org.springframework.context.annotation.Primary
import org.springframework.context.annotation.Profile
import party.qwer.twentyq.mcp.GuardEvaluation
import party.qwer.twentyq.mcp.NlpAnalysis
import party.qwer.twentyq.mcp.TokenUsage
import party.qwer.twentyq.mcp.TwentyQAnswerResponse
import party.qwer.twentyq.mcp.TwentyQHintsResponse
import party.qwer.twentyq.mcp.TwentyQNormalizeResponse
import party.qwer.twentyq.mcp.TwentyQSynonymResponse
import party.qwer.twentyq.mcp.TwentyQVerifyResponse
import party.qwer.twentyq.rest.GuardRestClient
import party.qwer.twentyq.rest.HealthCheckResult
import party.qwer.twentyq.rest.LlmRestClient
import party.qwer.twentyq.rest.NlpRestClient
import party.qwer.twentyq.rest.TwentyQRestClient
import party.qwer.twentyq.rest.dto.NlpHeuristicsResponse
import party.qwer.twentyq.rest.dto.SessionCreateResponse

/** 테스트용 REST 클라이언트 설정 */
@Configuration
@Profile("!integration")
class TestMcpConfig {
    @Bean
    @Primary
    fun llmHttpClient(): HttpClient = mockk(relaxed = true)

    @Bean
    @Primary
    fun llmRestClient(): LlmRestClient {
        val mock = mockk<LlmRestClient>()

        coEvery { mock.getUsage() } returns TokenUsage()
        coEvery { mock.getTotalUsage() } returns TokenUsage()
        coEvery { mock.checkHealth(any()) } returns HealthCheckResult.Healthy

        return mock
    }

    @Bean
    @Primary
    fun guardRestClient(): GuardRestClient {
        val mock = mockk<GuardRestClient>()

        coEvery { mock.evaluateGuard(any()) } returns
            GuardEvaluation(isMalicious = false, score = 0.0)

        coEvery { mock.isMalicious(any()) } returns false

        return mock
    }

    @Bean
    @Primary
    fun nlpRestClient(): NlpRestClient {
        val mock = mockk<NlpRestClient>()

        coEvery { mock.analyzeNlp(any()) } returns NlpAnalysis()
        coEvery { mock.getAnomalyScore(any()) } returns 0.0
        coEvery { mock.analyzeHeuristics(any()) } returns NlpHeuristicsResponse()

        return mock
    }

    @Bean
    @Primary
    fun twentyQRestClient(): TwentyQRestClient {
        val mock = mockk<TwentyQRestClient>()

        coEvery { mock.createSession(any()) } returns
            SessionCreateResponse(sessionId = "test-session", created = true)

        coEvery { mock.endSession(any()) } returns true

        coEvery { mock.generateHints(any(), any(), any()) } returns
            TwentyQHintsResponse(hints = listOf("mocked hint"))

        coEvery { mock.answerQuestion(any(), any(), any(), any(), any()) } returns
            TwentyQAnswerResponse(scale = "yes", rawText = "yes")

        coEvery { mock.verifyGuess(any(), any()) } returns
            TwentyQVerifyResponse(result = "REJECT", rawText = "REJECT")

        coEvery { mock.normalizeQuestion(any()) } answers {
            val question = firstArg<String>()
            TwentyQNormalizeResponse(normalized = question, original = question)
        }

        coEvery { mock.checkSynonym(any(), any()) } returns
            TwentyQSynonymResponse(result = "NOT_EQUIVALENT", rawText = "NOT_EQUIVALENT")

        return mock
    }
}
