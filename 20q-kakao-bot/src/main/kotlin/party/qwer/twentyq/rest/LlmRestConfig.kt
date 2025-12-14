package party.qwer.twentyq.rest

import io.ktor.client.HttpClient
import io.ktor.client.engine.okhttp.OkHttp
import io.ktor.client.plugins.HttpTimeout
import io.ktor.client.plugins.contentnegotiation.ContentNegotiation
import io.ktor.client.plugins.defaultRequest
import io.ktor.http.ContentType
import io.ktor.http.contentType
import io.ktor.serialization.jackson.jackson
import jakarta.annotation.PreDestroy
import okhttp3.Protocol
import org.slf4j.LoggerFactory
import org.springframework.context.annotation.Bean
import org.springframework.context.annotation.Configuration
import party.qwer.twentyq.util.http.HttpConstants

/** Ktor HttpClient 설정 */
@Configuration
class LlmRestConfig {
    companion object {
        private val log = LoggerFactory.getLogger(LlmRestConfig::class.java)
    }

    private var httpClient: HttpClient? = null

    @Bean
    fun llmRestProperties(): LlmRestProperties = LlmRestProperties()

    @Bean
    fun llmHttpClient(properties: LlmRestProperties): HttpClient {
        val mode = if (properties.http2Enabled) "H2C" else "HTTP/1.1"
        log.info(
            "LLM_HTTP_CLIENT_INIT mode={}, target={}, timeout={}s",
            mode,
            properties.baseUrl,
            properties.timeoutSeconds,
        )

        val client =
            HttpClient(OkHttp) {
                engine {
                    config {
                        val protocolList =
                            if (properties.http2Enabled) {
                                listOf(Protocol.H2_PRIOR_KNOWLEDGE)
                            } else {
                                listOf(Protocol.HTTP_1_1)
                            }
                        protocols(protocolList)
                    }
                }

                install(ContentNegotiation) {
                    jackson()
                }

                install(HttpTimeout) {
                    requestTimeoutMillis = properties.timeoutSeconds * HttpConstants.MILLIS_PER_SECOND
                    connectTimeoutMillis = properties.connectTimeoutSeconds * HttpConstants.MILLIS_PER_SECOND
                    socketTimeoutMillis = properties.timeoutSeconds * HttpConstants.MILLIS_PER_SECOND
                }

                defaultRequest {
                    url(properties.baseUrl)
                    contentType(ContentType.Application.Json)
                    headers.append("X-Request-ID", RestErrorHandler.generateRequestId())
                }

                expectSuccess = false
            }

        httpClient = client
        return client
    }

    @PreDestroy
    fun cleanup() {
        httpClient?.close()
        log.info("LLM_HTTP_CLIENT_CLOSED")
    }
}
