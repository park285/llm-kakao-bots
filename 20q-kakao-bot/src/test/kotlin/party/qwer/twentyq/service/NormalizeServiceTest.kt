package party.qwer.twentyq.service

import io.mockk.coEvery
import io.mockk.impl.annotations.MockK
import io.mockk.junit5.MockKExtension
import kotlinx.coroutines.test.runTest
import org.junit.jupiter.api.Assertions.assertEquals
import org.junit.jupiter.api.BeforeEach
import org.junit.jupiter.api.Test
import org.junit.jupiter.api.extension.ExtendWith
import org.springframework.cache.CacheManager
import party.qwer.twentyq.config.AppProperties
import party.qwer.twentyq.config.properties.RedisDefaults
import party.qwer.twentyq.mcp.TwentyQNormalizeResponse
import party.qwer.twentyq.rest.TwentyQRestClient

@ExtendWith(MockKExtension::class)
class NormalizeServiceTest {
    @MockK
    private lateinit var restClient: TwentyQRestClient

    @MockK
    private lateinit var complexityAnalyzer: ComplexityAnalyzer

    @MockK
    private lateinit var cacheManager: CacheManager

    private lateinit var appProperties: AppProperties
    private lateinit var normalizationConfig: NormalizationConfig
    private lateinit var service: NormalizeService

    @BeforeEach
    fun setUp() {
        appProperties = AppProperties(cache = RedisDefaults())
        normalizationConfig =
            NormalizationConfig(
                appProperties = appProperties,
                cacheEnabled = false,
            )
        service =
            NormalizeService(
                restClient = restClient,
                complexityAnalyzer = complexityAnalyzer,
                cacheManager = cacheManager,
                normalizationConfig = normalizationConfig,
            )
    }

    @Test
    fun `normalize returns regex result when no complex typo`() =
        runTest {
            val text = "이거 뭐에요???"

            coEvery { complexityAnalyzer.hasComplexTypo(any()) } returns false

            val result = service.normalize(text)

            // 연속 물음표가 하나로 정규화됨
            assertEquals("이거 뭐에요?", result.normalized)
        }

    @Test
    fun `normalize calls MCP when complex typo detected`() =
        runTest {
            val text = "복잡한 오타 텍스트"

            coEvery { complexityAnalyzer.hasComplexTypo(any()) } returns true
            coEvery { restClient.normalizeQuestion(any()) } returns
                TwentyQNormalizeResponse(
                    normalized = "정규화된 텍스트",
                    original = text,
                )

            val result = service.normalize(text)

            assertEquals("정규화된 텍스트", result.normalized)
        }

    @Test
    fun `normalize falls back to original when MCP returns error`() =
        runTest {
            val text = "오타 텍스트"

            coEvery { complexityAnalyzer.hasComplexTypo(any()) } returns true
            coEvery { restClient.normalizeQuestion(any()) } returns
                TwentyQNormalizeResponse(
                    original = text,
                    isError = true,
                    errorMessage = "MCP Error",
                )

            val result = service.normalize(text)

            assertEquals(text, result.normalized)
        }

    @Test
    fun `normalize falls back to original when MCP throws exception`() =
        runTest {
            val text = "오타 텍스트"

            coEvery { complexityAnalyzer.hasComplexTypo(any()) } returns true
            coEvery { restClient.normalizeQuestion(any()) } throws RuntimeException("MCP Error")

            val result = service.normalize(text)

            assertEquals(text, result.normalized)
        }

    @Test
    fun `normalize returns trimmed text for blank input`() =
        runTest {
            val result = service.normalize("   ")

            assertEquals("", result.normalized)
        }

    @Test
    fun `normalize handles repeated exclamation marks`() =
        runTest {
            val text = "대박!!!"

            coEvery { complexityAnalyzer.hasComplexTypo(any()) } returns false

            val result = service.normalize(text)

            assertEquals("대박!", result.normalized)
        }

    @Test
    fun `normalize handles repeated ㅋ characters`() =
        runTest {
            val text = "ㅋㅋㅋㅋㅋ 웃겨요"

            coEvery { complexityAnalyzer.hasComplexTypo(any()) } returns false

            val result = service.normalize(text)

            assertEquals("ㅋㅋ 웃겨요", result.normalized)
        }
}
