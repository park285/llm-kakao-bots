package party.qwer.twentyq.config

import com.ninjasquad.springmockk.MockkBean
import io.mockk.coEvery
import org.junit.jupiter.api.Test
import org.springframework.beans.factory.annotation.Autowired
import org.springframework.boot.test.context.SpringBootTest
import org.springframework.boot.test.context.TestConfiguration
import org.springframework.context.annotation.Bean
import org.springframework.context.annotation.Import
import org.springframework.http.HttpStatus
import org.springframework.test.annotation.DirtiesContext
import org.springframework.test.web.reactive.server.WebTestClient
import org.springframework.web.reactive.function.client.ExchangeStrategies
import party.qwer.twentyq.service.RiddleService

@SpringBootTest(webEnvironment = SpringBootTest.WebEnvironment.RANDOM_PORT)
@org.springframework.test.context.ActiveProfiles("test")
@org.springframework.test.context.TestPropertySource(
    properties = [
        "mcp.llm.enabled=false",
        "management.health.r2dbc.enabled=false",
    ],
)
@Import(AccessControlFilterTest.WebTestClientConfig::class)
@DirtiesContext(classMode = DirtiesContext.ClassMode.AFTER_CLASS)
class AccessControlFilterTest {
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

    @Test
    fun `should allow API request when access control is disabled`() {
        coEvery { riddleService.createRiddle(any(), any()) } returns "게임 시작"

        webTestClient
            .post()
            .uri("/api/twentyq/riddles")
            .header("X-Session-Id", "test-session")
            .exchange()
            .expectStatus()
            .isOk
    }

    @Test
    fun `should allow actuator endpoint`() {
        webTestClient
            .get()
            .uri("/actuator/health")
            .exchange()
            .expectStatus()
            .isOk
    }
}
