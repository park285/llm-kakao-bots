package io.github.kapu.turtlesoup.rest

import io.github.kapu.turtlesoup.config.JsonConfig
import io.github.kapu.turtlesoup.config.LlmRestConfig
import io.github.kapu.turtlesoup.config.TimeConstants
import io.github.kapu.turtlesoup.models.PuzzleGenerationRequest
import io.github.kapu.turtlesoup.models.PuzzleGenerationResponse
import io.github.kapu.turtlesoup.models.ValidationResult
import io.github.oshai.kotlinlogging.KotlinLogging
import io.ktor.client.HttpClient
import io.ktor.client.call.body
import io.ktor.client.engine.okhttp.OkHttp
import io.ktor.client.plugins.HttpTimeout
import io.ktor.client.plugins.ResponseException
import io.ktor.client.plugins.contentnegotiation.ContentNegotiation
import io.ktor.client.request.accept
import io.ktor.client.request.delete
import io.ktor.client.request.get
import io.ktor.client.request.post
import io.ktor.client.request.setBody
import io.ktor.http.ContentType
import io.ktor.http.contentType
import io.ktor.serialization.kotlinx.json.json
import kotlinx.serialization.SerialName
import kotlinx.serialization.Serializable
import okhttp3.Protocol
import java.io.IOException
import java.net.UnknownHostException
import kotlin.coroutines.cancellation.CancellationException

/** REST client for LLM operations (replaces MCP stdio client). */
@Suppress("TooManyFunctions")
class LlmRestClient(
    private val config: LlmRestConfig,
) {
    private var baseUrl: String = config.baseUrl

    @Serializable
    data class HealthResponse(val status: String = "")

    @Serializable
    data class ModelConfigResponse(
        @SerialName("model_default")
        val modelDefault: String,
        @SerialName("model_hints")
        val modelHints: String? = null,
        @SerialName("model_answer")
        val modelAnswer: String? = null,
        @SerialName("model_verify")
        val modelVerify: String? = null,
        val temperature: Double,
        @SerialName("timeout_seconds")
        val timeoutSeconds: Int,
        @SerialName("max_retries")
        val maxRetries: Int,
        @SerialName("http2_enabled")
        val http2Enabled: Boolean,
    )

    init {
        val mode = if (config.http2Enabled) "H2C" else "HTTP/1.1"
        logger.info { "llm_rest_client_init mode=$mode target=$baseUrl" }
    }

    private val httpClient =
        HttpClient(OkHttp) {
            engine {
                config {
                    val protocolList =
                        if (config.http2Enabled) {
                            listOf(Protocol.H2_PRIOR_KNOWLEDGE)
                        } else {
                            listOf(Protocol.HTTP_1_1)
                        }
                    protocols(protocolList)
                }
            }
            install(ContentNegotiation) {
                json(JsonConfig.lenient)
            }
            install(HttpTimeout) {
                requestTimeoutMillis = config.timeoutSeconds * TimeConstants.MILLIS_PER_SECOND
                connectTimeoutMillis = config.connectTimeoutSeconds * TimeConstants.MILLIS_PER_SECOND
            }
        }

    private suspend inline fun <reified Req : Any, reified Res> postJson(
        path: String,
        body: Req,
    ): Res =
        httpClient.post("$baseUrl$path") {
            contentType(ContentType.Application.Json)
            setBody(body)
        }.body()

    suspend fun getModelConfig(): ModelConfigResponse? =
        runCatching {
            httpClient.get("$baseUrl/health/models") {
                accept(ContentType.Application.Json)
            }.body<ModelConfigResponse>()
        }.getOrElse { error ->
            logger.warn(error) { "rest_get_model_config_failed" }
            null
        }

    // turtle_answer_question
    suspend fun answerQuestion(
        chatId: String,
        scenario: String,
        solution: String,
        question: String,
    ): AnswerQuestionResult {
        logger.debug { "rest_answer_question" }
        val response: AnswerResponse =
            postJson(
                path = "/api/turtle-soup/answers",
                body =
                    AnswerRequest(
                        chatId = chatId,
                        namespace = LLM_NAMESPACE,
                        scenario = scenario,
                        solution = solution,
                        question = question,
                    ),
            )

        return AnswerQuestionResult(
            answer = response.answer,
            questionCount = response.questionCount,
            history = response.history.map { QuestionHistoryItem(it.question, it.answer) },
        )
    }

    // turtle_generate_hint
    suspend fun generateHint(
        chatId: String,
        scenario: String,
        solution: String,
        level: Int,
    ): String {
        logger.debug { "rest_generate_hint" }
        val response: HintResponse =
            postJson(
                path = "/api/turtle-soup/hints",
                body =
                    HintRequest(
                        chatId = chatId,
                        namespace = LLM_NAMESPACE,
                        scenario = scenario,
                        solution = solution,
                        level = level,
                    ),
            )

        return response.hint
    }

    // turtle_validate_solution
    suspend fun validateSolution(
        chatId: String,
        solution: String,
        playerAnswer: String,
    ): ValidationResult {
        logger.debug { "rest_validate_solution" }
        val response: ValidateResponse =
            postJson(
                path = "/api/turtle-soup/validations",
                body =
                    ValidateRequest(
                        chatId = chatId,
                        namespace = LLM_NAMESPACE,
                        solution = solution,
                        playerAnswer = playerAnswer,
                    ),
            )

        return ValidationResult.valueOf(response.result.trim().uppercase())
    }

    // turtle_rewrite_scenario
    suspend fun rewriteScenario(
        title: String,
        scenario: String,
        solution: String,
        difficulty: Int,
    ): RewriteResult {
        logger.debug { "rest_rewrite_scenario title=$title" }
        val response: RewriteResponse =
            postJson(
                path = "/api/turtle-soup/rewrites",
                body =
                    RewriteRequest(
                        title = title,
                        scenario = scenario,
                        solution = solution,
                        difficulty = difficulty,
                    ),
            )

        return RewriteResult(
            scenario = response.scenario,
            solution = response.solution,
        )
    }

    // turtle_generate_puzzle
    suspend fun generatePuzzle(request: PuzzleGenerationRequest): PuzzleGenerationResponse {
        logger.debug {
            "rest_generate_puzzle category=${request.category} difficulty=${request.difficulty} theme=${request.theme}"
        }
        return postJson(
            path = "/api/turtle-soup/puzzles",
            body = request,
        )
    }

    // guard_is_malicious
    suspend fun guardIsMalicious(text: String): Boolean {
        logger.debug { "rest_guard_is_malicious" }
        val response: GuardMaliciousResponse =
            postJson(
                path = "/api/guard/checks",
                body = GuardRequest(inputText = text),
            )

        return response.malicious
    }

    // session_create
    suspend fun createSession(chatId: String): SessionCreateResult {
        logger.debug { "rest_create_session" }
        val response: SessionCreateResponse =
            postJson(
                path = "/api/sessions",
                body = SessionCreateRequest(chatId = chatId, namespace = LLM_NAMESPACE),
            )

        return SessionCreateResult(
            sessionId = response.sessionId,
            model = response.model,
            created = response.created,
        )
    }

    // session_end
    suspend fun endSession(sessionId: String): Boolean {
        logger.debug { "rest_end_session session_id=$sessionId" }
        return try {
            val response =
                httpClient.delete("$baseUrl/api/sessions/$sessionId") {
                    contentType(ContentType.Application.Json)
                }.body<SessionEndResponse>()
            response.removed
        } catch (
            e: ResponseException,
        ) {
            logger.warn(e) { "rest_end_session_failed session_id=$sessionId" }
            false
        } catch (e: IOException) {
            logger.warn(e) { "rest_end_session_failed_io session_id=$sessionId" }
            false
        } catch (e: CancellationException) {
            logger.warn(e) { "rest_end_session_cancelled session_id=$sessionId" }
            false
        }
    }

    // chatId 기반 파생 세션 종료 (namespace:chatId)
    suspend fun endSessionByChat(chatId: String): Boolean {
        val derivedSessionId = "$LLM_NAMESPACE:$chatId"
        return endSession(derivedSessionId)
    }

    suspend fun isHealthy(): Boolean {
        val response =
            runCatching {
                httpClient.get("$baseUrl${config.healthPath}") { contentType(ContentType.Application.Json) }
                    .body<HealthResponse>()
            }.getOrElse { error ->
                handleHealthError(baseUrl, error)
                return false
            }

        return response.status.trim().lowercase() == "ok"
    }

    private fun handleHealthError(
        targetUrl: String,
        error: Throwable,
    ) {
        when (error) {
            is ResponseException -> logger.warn(error) { "rest_health_failed_response target=$targetUrl" }
            is IOException -> logger.warn(error) { "rest_health_failed_io target=$targetUrl" }
            is CancellationException -> logger.warn(error) { "rest_health_cancelled target=$targetUrl" }
            is UnknownHostException -> logger.warn(error) { "rest_health_unknown_host target=$targetUrl" }
            else -> logger.error(error) { "rest_health_failed_unexpected target=$targetUrl" }
        }
    }

    // puzzle_get_random
    suspend fun getRandomPuzzle(difficulty: Int? = null): PuzzlePresetResult {
        logger.debug { "rest_get_random_puzzle difficulty=$difficulty" }
        val url =
            buildString {
                append("$baseUrl/api/turtle-soup/puzzles/random")
                if (difficulty != null) {
                    append("?difficulty=$difficulty")
                }
            }
        val response = httpClient.get(url).body<PuzzlePresetResponse>()

        return PuzzlePresetResult(
            id = response.id ?: error("Missing puzzle id in response"),
            title = response.title ?: error("Missing puzzle title in response"),
            question = response.question ?: error("Missing puzzle question in response"),
            answer = response.answer ?: error("Missing puzzle answer in response"),
            difficulty = response.difficulty ?: error("Missing puzzle difficulty in response"),
        )
    }

    fun close() {
        httpClient.close()
        logger.info { "rest_client_closed" }
    }

    companion object {
        private val logger = KotlinLogging.logger {}
        private const val LLM_NAMESPACE = "turtle-soup"
    }
}
