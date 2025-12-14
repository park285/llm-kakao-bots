package party.qwer.twentyq.rest

import com.fasterxml.jackson.annotation.JsonIgnoreProperties
import com.fasterxml.jackson.annotation.JsonProperty
import io.ktor.client.HttpClient
import io.ktor.client.call.body
import io.ktor.client.plugins.ResponseException
import io.ktor.client.request.get
import io.ktor.client.request.header
import io.ktor.client.statement.HttpResponse
import io.ktor.http.isSuccess
import org.slf4j.LoggerFactory
import org.springframework.stereotype.Service
import party.qwer.twentyq.mcp.TokenUsage
import party.qwer.twentyq.rest.dto.DailyUsageResponse
import party.qwer.twentyq.rest.dto.UsageListResponse
import party.qwer.twentyq.rest.dto.UsageResponse
import java.io.IOException
import kotlin.coroutines.cancellation.CancellationException

sealed interface HealthCheckResult {
    data object Healthy : HealthCheckResult

    data class Unhealthy(
        val reason: HealthFailureReason,
        val message: String? = null,
    ) : HealthCheckResult
}

enum class HealthFailureReason {
    HTTP_ERROR,
    EXCEPTION,
}

/** LLM REST API 클라이언트 */
@Service
class LlmRestClient(
    private val llmHttpClient: HttpClient,
    private val properties: LlmRestProperties,
    private val restErrorHandler: RestErrorHandler,
) {
    companion object {
        private val log = LoggerFactory.getLogger(LlmRestClient::class.java)
        private const val USAGE_PATH = "/api/llm/usage"
        private const val TOTAL_USAGE_PATH = "/api/llm/usage/total"
        private const val DAILY_USAGE_PATH = "/api/usage/daily"
        private const val RECENT_USAGE_PATH = "/api/usage/recent"
        private const val USAGE_TOTAL_PATH = "/api/usage/total"
    }

    @JsonIgnoreProperties(ignoreUnknown = true)
    data class HealthResponse(
        val status: String = "",
    )

    suspend fun checkHealth(path: String): HealthCheckResult {
        val requestId = RestErrorHandler.generateRequestId()
        return try {
            val response = fetchHealthResponse(path, requestId)
            mapHealthResponse(response)
        } catch (e: ResponseException) {
            handleHttpException(e, requestId)
        } catch (e: IOException) {
            handleIoException(e, requestId)
        } catch (e: CancellationException) {
            val error = restErrorHandler.fromException(e, requestId)
            log.warn("REST_HEALTH_ERROR {}", error.toLogString())
            HealthCheckResult.Unhealthy(HealthFailureReason.EXCEPTION, error.toLogString())
        }
    }

    private suspend fun fetchHealthResponse(
        path: String,
        requestId: String,
    ): HttpResponse =
        llmHttpClient.get(path) {
            header("X-Request-ID", requestId)
        }

    private suspend fun mapHealthResponse(response: HttpResponse): HealthCheckResult {
        val statusCode = response.status.value

        if (!response.status.isSuccess()) {
            val error = restErrorHandler.parseErrorResponse(response)
            log.warn("REST_HEALTH_FAILED {}", error.toLogString())
            return HealthCheckResult.Unhealthy(HealthFailureReason.HTTP_ERROR, error.toLogString())
        }

        val status =
            response
                .body<HealthResponse?>()
                ?.status
                ?.trim()
                .orEmpty()
        if (status.isEmpty()) {
            log.info("REST_HEALTH_OK status={}", statusCode)
            return HealthCheckResult.Healthy
        }
        return if (status.equals("ok", ignoreCase = true) || status.equals("up", ignoreCase = true)) {
            log.info("REST_HEALTH_OK status={}", statusCode)
            HealthCheckResult.Healthy
        } else {
            HealthCheckResult.Unhealthy(HealthFailureReason.HTTP_ERROR, "status=$status")
        }
    }

    private fun handleHttpException(
        e: ResponseException,
        requestId: String,
    ): HealthCheckResult {
        val error = restErrorHandler.fromException(e, requestId)
        log.warn("REST_HEALTH_ERROR {}", error.toLogString())
        return HealthCheckResult.Unhealthy(HealthFailureReason.HTTP_ERROR, error.toLogString())
    }

    private fun handleIoException(
        e: IOException,
        requestId: String,
    ): HealthCheckResult {
        val error = restErrorHandler.fromException(e, requestId)
        val message = error.toLogString()
        log.warn("REST_HEALTH_ERROR {}", message)
        return HealthCheckResult.Unhealthy(HealthFailureReason.EXCEPTION, message)
    }

    /** 마지막 사용량 조회 (메모리) */
    suspend fun getUsage(): TokenUsage {
        val requestId = RestErrorHandler.generateRequestId()
        return restErrorHandler.executeRestCall(
            operation = "USAGE",
            requestId = requestId,
            request = {
                llmHttpClient.get(USAGE_PATH) {
                    header("X-Request-ID", requestId)
                }
            },
            onSuccess = { response ->
                val usage = response.body<UsageResponse>()
                TokenUsage(
                    inputTokens = usage.inputTokens,
                    outputTokens = usage.outputTokens,
                    totalTokens = usage.totalTokens,
                    reasoningTokens = usage.reasoningTokens ?: 0,
                    model = usage.model,
                )
            },
            onFailure = { TokenUsage() },
        )
    }

    /** 서버 누적 사용량 조회 (메모리, 서버 시작 이후) */
    suspend fun getTotalUsage(): TokenUsage {
        val requestId = RestErrorHandler.generateRequestId()
        return restErrorHandler.executeRestCall(
            operation = "TOTAL_USAGE",
            requestId = requestId,
            request = {
                llmHttpClient.get(TOTAL_USAGE_PATH) {
                    header("X-Request-ID", requestId)
                }
            },
            onSuccess = { response ->
                val usage = response.body<UsageResponse>()
                TokenUsage(
                    inputTokens = usage.inputTokens,
                    outputTokens = usage.outputTokens,
                    totalTokens = usage.totalTokens,
                    reasoningTokens = usage.reasoningTokens ?: 0,
                    model = usage.model,
                )
            },
            onFailure = { TokenUsage() },
        )
    }

    /** 오늘 일별 사용량 조회 (DB) */
    suspend fun getDailyUsage(): DailyUsageResponse? {
        val requestId = RestErrorHandler.generateRequestId()
        return restErrorHandler.executeRestCall(
            operation = "DAILY_USAGE",
            requestId = requestId,
            request = {
                llmHttpClient.get(DAILY_USAGE_PATH) {
                    header("X-Request-ID", requestId)
                }
            },
            onSuccess = { response -> response.body<DailyUsageResponse>() },
            onFailure = { null },
        )
    }

    /** 최근 N일간 사용량 조회 (DB) */
    suspend fun getRecentUsage(days: Int = 7): UsageListResponse? {
        val requestId = RestErrorHandler.generateRequestId()
        return restErrorHandler.executeRestCall(
            operation = "RECENT_USAGE",
            requestId = requestId,
            request = {
                llmHttpClient.get("$RECENT_USAGE_PATH?days=$days") {
                    header("X-Request-ID", requestId)
                }
            },
            onSuccess = { response -> response.body<UsageListResponse>() },
            onFailure = { null },
        )
    }

    /** N일간 총 사용량 조회 (DB) */
    suspend fun getTotalUsageFromDb(days: Int = 30): UsageResponse? {
        val requestId = RestErrorHandler.generateRequestId()
        return restErrorHandler.executeRestCall(
            operation = "USAGE_TOTAL",
            requestId = requestId,
            request = {
                llmHttpClient.get("$USAGE_TOTAL_PATH?days=$days") {
                    header("X-Request-ID", requestId)
                }
            },
            onSuccess = { response -> response.body<UsageResponse>() },
            onFailure = { null },
        )
    }

    @JsonIgnoreProperties(ignoreUnknown = true)
    data class ModelConfigResponse(
        @param:JsonProperty("model_default")
        val modelDefault: String = "",
        @param:JsonProperty("model_hints")
        val modelHints: String? = null,
        @param:JsonProperty("model_answer")
        val modelAnswer: String? = null,
        @param:JsonProperty("model_verify")
        val modelVerify: String? = null,
        @param:JsonProperty("temperature")
        val temperature: Double = 0.0,
        @param:JsonProperty("timeout_seconds")
        val timeoutSeconds: Int = 0,
        @param:JsonProperty("max_retries")
        val maxRetries: Int = 0,
        @param:JsonProperty("http2_enabled")
        val http2Enabled: Boolean = true,
    )

    suspend fun getModelConfig(): ModelConfigResponse? {
        val requestId = RestErrorHandler.generateRequestId()
        return restErrorHandler.executeRestCall(
            operation = "MODEL_CONFIG",
            requestId = requestId,
            request = {
                llmHttpClient.get("/health/models") {
                    header("X-Request-ID", requestId)
                }
            },
            onSuccess = { response -> response.body<ModelConfigResponse>() },
            onFailure = { null },
        )
    }
}
