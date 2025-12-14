package party.qwer.twentyq.rest

import io.ktor.client.HttpClient
import io.ktor.client.engine.mock.MockEngine
import io.ktor.client.engine.mock.respond
import io.ktor.client.plugins.contentnegotiation.ContentNegotiation
import io.ktor.http.ContentType
import io.ktor.http.HttpHeaders
import io.ktor.http.HttpStatusCode
import io.ktor.http.headersOf
import io.ktor.serialization.jackson.jackson
import io.mockk.mockk
import kotlinx.coroutines.test.runTest
import org.junit.jupiter.api.Assertions.assertTrue
import org.junit.jupiter.api.Test

class LlmRestClientTest {
    @Test
    fun `should treat actuator UP as healthy`() =
        runTest {
            val client =
                HttpClient(MockEngine) {
                    install(ContentNegotiation) { jackson() }
                    engine {
                        addHandler {
                            respond(
                                content = """{"status":"UP"}""",
                                status = HttpStatusCode.OK,
                                headers =
                                    headersOf(
                                        HttpHeaders.ContentType,
                                        ContentType.Application.Json.toString(),
                                    ),
                            )
                        }
                    }
                }

            val restClient = LlmRestClient(client, LlmRestProperties(), mockk(relaxed = true))

            val result = restClient.checkHealth("http://localhost/health")

            assertTrue(result is HealthCheckResult.Healthy)
        }

    @Test
    fun `should ignore unknown fields in health response`() =
        runTest {
            val client =
                HttpClient(MockEngine) {
                    install(ContentNegotiation) { jackson() }
                    engine {
                        addHandler {
                            respond(
                                content = """{"status":"OK","components":{"diskSpace":{"status":"UP"}}}""",
                                status = HttpStatusCode.OK,
                                headers =
                                    headersOf(
                                        HttpHeaders.ContentType,
                                        ContentType.Application.Json.toString(),
                                    ),
                            )
                        }
                    }
                }

            val restClient = LlmRestClient(client, LlmRestProperties(), mockk(relaxed = true))

            val result = restClient.checkHealth("http://localhost/health")

            assertTrue(result is HealthCheckResult.Healthy)
        }
}
