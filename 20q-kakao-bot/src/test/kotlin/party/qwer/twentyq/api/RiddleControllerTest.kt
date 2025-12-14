package party.qwer.twentyq.api

import com.ninjasquad.springmockk.MockkBean
import io.mockk.coEvery
import org.junit.jupiter.api.BeforeEach
import org.junit.jupiter.api.Test
import org.springframework.beans.factory.annotation.Autowired
import org.springframework.boot.test.context.SpringBootTest
import org.springframework.boot.test.context.TestConfiguration
import org.springframework.context.annotation.Bean
import org.springframework.context.annotation.Import
import org.springframework.http.MediaType
import org.springframework.test.annotation.DirtiesContext
import org.springframework.test.web.reactive.server.WebTestClient
import org.springframework.web.reactive.function.client.ExchangeStrategies
import party.qwer.twentyq.model.FiveScaleKo
import party.qwer.twentyq.rest.LlmAvailabilityGuard
import party.qwer.twentyq.service.RiddleService
import party.qwer.twentyq.service.dto.AnswerResult
import party.qwer.twentyq.service.dto.AnswerSource

@SpringBootTest(webEnvironment = SpringBootTest.WebEnvironment.RANDOM_PORT)
@org.springframework.test.context.ActiveProfiles("test")
@org.springframework.test.context.TestPropertySource(properties = ["mcp.llm.enabled=false"])
@Import(RiddleControllerTest.WebTestClientConfig::class)
@DirtiesContext(classMode = DirtiesContext.ClassMode.AFTER_CLASS)
class RiddleControllerTest {
    @TestConfiguration
    class WebTestClientConfig {
        @Bean
        fun webTestClient(
            @Autowired(required = false) applicationContext: org.springframework.context.ApplicationContext,
        ): WebTestClient =
            WebTestClient
                .bindToApplicationContext(applicationContext)
                .configureClient()
                .exchangeStrategies(
                    ExchangeStrategies
                        .builder()
                        .codecs { it.defaultCodecs().maxInMemorySize(16 * 1024 * 1024) }
                        .build(),
                ).build()
    }

    @Autowired
    private lateinit var webTestClient: WebTestClient

    @MockkBean
    private lateinit var riddleService: RiddleService

    @MockkBean
    private lateinit var llmAvailabilityGuard: LlmAvailabilityGuard

    @BeforeEach
    fun setup() {
        coEvery { llmAvailabilityGuard.isAvailable() } returns true
    }

    @Test
    fun `create should accept request with valid session header`() {
        coEvery {
            riddleService.createRiddle(any(), any())
        } returns "게임이 시작되었습니다. 20개 질문 안에 정답을 맞춰보세요!"

        webTestClient
            .post()
            .uri("/api/twentyq/riddles")
            .header("X-Session-Id", "test-session")
            .header("X-User-Id", "test-user")
            .contentType(MediaType.APPLICATION_JSON)
            .bodyValue("""{"category": "animal"}""")
            .exchange()
            .expectStatus()
            .isOk
            .expectBody()
            .jsonPath("$.message")
            .isNotEmpty
    }

    @Test
    fun `create should handle request without category`() {
        // Given
        coEvery {
            riddleService.createRiddle(any(), null)
        } returns "게임이 시작되었습니다."

        webTestClient
            .post()
            .uri("/api/twentyq/riddles")
            .header("X-Session-Id", "test-session")
            .contentType(MediaType.APPLICATION_JSON)
            .bodyValue("{}")
            .exchange()
            .expectStatus()
            .isOk
    }

    @Test
    fun `hints should generate hints for valid session`() {
        // Given
        coEvery {
            riddleService.generateHints(any(), any())
        } returns listOf("힌트1", "힌트2", "힌트3")

        // When: POST /api/twentyq/riddles/hints
        webTestClient
            .post()
            .uri("/api/twentyq/riddles/hints")
            .header("X-Session-Id", "test-session")
            .contentType(MediaType.APPLICATION_JSON)
            .bodyValue("""{"count": 3}""")
            .exchange()
            .expectStatus()
            .isOk
            .expectBody()
            .jsonPath("$.hints")
            .isArray
            .jsonPath("$.hints[0]")
            .isEqualTo("힌트1")
    }

    @Test
    fun `answer should process user question`() {
        // Given
        coEvery {
            riddleService.answer(any(), any(), any())
        } returns
            AnswerResult(
                scale = FiveScaleKo.ALWAYS_YES,
                source = AnswerSource.ENUM_SCHEMA_PRIMARY,
                guardDegraded = false,
            )

        webTestClient
            .post()
            .uri("/api/twentyq/riddles/answers")
            .header("X-Session-Id", "test-session")
            .contentType(MediaType.APPLICATION_JSON)
            .bodyValue("""{"question": "동물인가요?"}""")
            .exchange()
            // Then
            .expectStatus()
            .isOk
            .expectBody()
            .jsonPath("$.scale")
            .isNotEmpty
    }

    @Test
    fun `status should return game status`() {
        // Given
        coEvery {
            riddleService.getStatus(any())
        } returns
            party.qwer.twentyq.api.dto.RiddleStatusResponse(
                questionCount = 5,
                questions = emptyList(),
                hints = emptyList(),
                hintCount = 0,
                maxHints = 3,
                selectedCategory = "animal",
            )

        webTestClient
            .get()
            .uri("/api/twentyq/riddles")
            .header("X-Session-Id", "test-session")
            .exchange()
            // Then
            .expectStatus()
            .isOk
            .expectBody()
            .jsonPath("$.questionCount")
            .isEqualTo(5)
            .jsonPath("$.selectedCategory")
            .isEqualTo("animal")
            .jsonPath("$.maxHints")
            .isEqualTo(3)
    }

    @Test
    fun `create should use remote address when session header is missing`() {
        coEvery {
            riddleService.createRiddle(any(), any())
        } returns "게임 시작"

        webTestClient
            .post()
            .uri("/api/twentyq/riddles")
            .contentType(MediaType.APPLICATION_JSON)
            .bodyValue("{}")
            .exchange()
            .expectStatus()
            .isOk
    }
}
