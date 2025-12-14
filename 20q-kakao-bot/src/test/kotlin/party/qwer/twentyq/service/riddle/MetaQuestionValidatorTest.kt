package party.qwer.twentyq.service.riddle

import io.mockk.coEvery
import io.mockk.coVerify
import io.mockk.every
import io.mockk.mockk
import io.mockk.spyk
import kotlinx.coroutines.test.runTest
import org.junit.jupiter.api.Assertions.assertFalse
import org.junit.jupiter.api.Assertions.assertTrue
import org.junit.jupiter.api.BeforeEach
import org.junit.jupiter.api.Test
import party.qwer.twentyq.config.AppProperties
import party.qwer.twentyq.config.properties.Security
import party.qwer.twentyq.rest.GuardRestClient

class MetaQuestionValidatorTest {
    private lateinit var restClient: GuardRestClient
    private lateinit var appProperties: AppProperties
    private lateinit var questionPolicy: QuestionPolicy
    private lateinit var validator: MetaQuestionValidator

    @BeforeEach
    fun setup() {
        restClient = mockk()
        appProperties = mockk()
        questionPolicy = mockk()

        setupDefaultMocks()

        validator =
            spyk(
                MetaQuestionValidator(
                    restClient = restClient,
                    appProperties = appProperties,
                    questionPolicy = questionPolicy,
                ),
            )
    }

    private fun setupDefaultMocks() {
        val securityConfig = mockk<Security>()
        every { appProperties.security } returns securityConfig
        every { securityConfig.metaQuestionLlmEnabled } returns true

        coEvery { questionPolicy.isAnswerLengthMetaQuestion(any()) } returns false
        coEvery { questionPolicy.isAnswerIndexMetaQuestion(any()) } returns false
        coEvery { questionPolicy.isAnswerBoundaryMetaQuestion(any()) } returns false
    }

    @Test
    fun `should return cached result on second call with same question`() =
        runTest {
            coEvery { restClient.isMalicious(any()) } returns true

            val question = "정답이 몇 글자야?"
            val result1 = validator.isMetaQuestion(question)
            val result2 = validator.isMetaQuestion(question)

            assertTrue(result1)
            assertTrue(result2)
            coVerify(exactly = 1) { restClient.isMalicious(any()) }
        }

    @Test
    fun `should call MCP Guard for different questions`() =
        runTest {
            coEvery { restClient.isMalicious("첫 번째 질문") } returns true
            coEvery { restClient.isMalicious("두 번째 질문") } returns false

            val result1 = validator.isMetaQuestion("첫 번째 질문")
            val result2 = validator.isMetaQuestion("두 번째 질문")

            assertTrue(result1)
            assertFalse(result2)
            coVerify(exactly = 2) { restClient.isMalicious(any()) }
        }

    @Test
    fun `should return true when MCP Guard detects malicious`() =
        runTest {
            coEvery { restClient.isMalicious(any()) } returns true

            val result = validator.isMetaQuestion("메타 질문")

            assertTrue(result)
        }

    @Test
    fun `should return false when MCP Guard allows`() =
        runTest {
            coEvery { restClient.isMalicious(any()) } returns false

            val result = validator.isMetaQuestion("일반 질문")

            assertFalse(result)
        }

    @Test
    fun `should return false when MCP Guard throws exception`() =
        runTest {
            coEvery { restClient.isMalicious(any()) } throws RuntimeException("MCP error")

            val result = validator.isMetaQuestion("에러 질문")

            assertFalse(result)
        }

    @Test
    fun `should normalize cache key by trimming`() =
        runTest {
            coEvery { restClient.isMalicious(any()) } returns false

            val q1 = "메타냐?"
            val q2 = "메타냐?   "

            val r1 = validator.isMetaQuestion(q1)
            val r2 = validator.isMetaQuestion(q2)

            assertFalse(r1)
            assertFalse(r2)
            coVerify(exactly = 1) { restClient.isMalicious(any()) }
        }

    @Test
    fun `shouldValidate should return false when metaQuestionLlmEnabled is false`() =
        runTest {
            val securityConfig = mockk<Security>()
            every { appProperties.security } returns securityConfig
            every { securityConfig.metaQuestionLlmEnabled } returns false

            val result = validator.shouldValidate("어떤 질문이든")

            assertFalse(result)
        }

    @Test
    fun `shouldValidate should return true when questionPolicy matches`() =
        runTest {
            coEvery { questionPolicy.isAnswerLengthMetaQuestion(any()) } returns true

            val result = validator.shouldValidate("정답이 몇 글자야?")

            assertTrue(result)
        }

    @Test
    fun `getCacheStats should return formatted stats`() {
        val stats = validator.getCacheStats()

        assertTrue(stats.contains("hits="))
        assertTrue(stats.contains("misses="))
        assertTrue(stats.contains("hitRate="))
    }

    @Test
    fun `refreshCache should always return true`() =
        runTest {
            val result = validator.refreshCache()

            assertTrue(result)
        }

    @Test
    fun `shouldValidate should detect simple jamo question`() =
        runTest {
            coEvery { questionPolicy.isAnswerLengthMetaQuestion("받침 있어?") } returns true

            val result = validator.shouldValidate("받침 있어?")

            assertTrue(result)
        }

    @Test
    fun `shouldValidate should detect non-standard unit question`() =
        runTest {
            coEvery { questionPolicy.isAnswerLengthMetaQuestion("몇 칸이야?") } returns true

            val result = validator.shouldValidate("몇 칸이야?")

            assertTrue(result)
        }

    @Test
    fun `shouldValidate should detect slot question`() =
        runTest {
            coEvery { questionPolicy.isAnswerLengthMetaQuestion("슬롯 3개야?") } returns true

            val result = validator.shouldValidate("슬롯 3개야?")

            assertTrue(result)
        }
}
