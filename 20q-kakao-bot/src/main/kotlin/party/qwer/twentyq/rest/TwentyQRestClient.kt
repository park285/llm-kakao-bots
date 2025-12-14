package party.qwer.twentyq.rest

import io.ktor.client.HttpClient
import io.ktor.client.call.body
import io.ktor.client.request.delete
import io.ktor.client.request.header
import io.ktor.client.request.post
import io.ktor.client.request.setBody
import org.springframework.stereotype.Service
import party.qwer.twentyq.mcp.TwentyQAnswerResponse
import party.qwer.twentyq.mcp.TwentyQHintsResponse
import party.qwer.twentyq.mcp.TwentyQNormalizeResponse
import party.qwer.twentyq.mcp.TwentyQSynonymResponse
import party.qwer.twentyq.mcp.TwentyQVerifyResponse
import party.qwer.twentyq.rest.dto.AnswerRequest
import party.qwer.twentyq.rest.dto.AnswerRestResponse
import party.qwer.twentyq.rest.dto.HintsRequest
import party.qwer.twentyq.rest.dto.HintsRestResponse
import party.qwer.twentyq.rest.dto.NormalizeRequest
import party.qwer.twentyq.rest.dto.NormalizeRestResponse
import party.qwer.twentyq.rest.dto.SessionCreateRequest
import party.qwer.twentyq.rest.dto.SessionCreateResponse
import party.qwer.twentyq.rest.dto.SessionCreateRestResponse
import party.qwer.twentyq.rest.dto.SessionEndRestResponse
import party.qwer.twentyq.rest.dto.SynonymRequest
import party.qwer.twentyq.rest.dto.SynonymRestResponse
import party.qwer.twentyq.rest.dto.VerifyRequest
import party.qwer.twentyq.rest.dto.VerifyRestResponse

/** 20Q REST API 클라이언트 */
@Service
class TwentyQRestClient(
    private val llmHttpClient: HttpClient,
    private val restErrorHandler: RestErrorHandler,
) {
    companion object {
        private const val SESSIONS_PATH = "/api/sessions"
        private const val HINTS_PATH = "/api/twentyq/hints"
        private const val ANSWERS_PATH = "/api/twentyq/answers"
        private const val VERIFICATIONS_PATH = "/api/twentyq/verifications"
        private const val NORMALIZATIONS_PATH = "/api/twentyq/normalizations"
        private const val SYNONYM_CHECKS_PATH = "/api/twentyq/synonym-checks"
        private const val LLM_NAMESPACE = "twentyq"
    }

    /** LLM 세션 생성 */
    suspend fun createSession(chatId: String): SessionCreateResponse {
        val requestId = RestErrorHandler.generateRequestId()
        return restErrorHandler.executeRestCall(
            operation = "CREATE_SESSION",
            requestId = requestId,
            request = {
                llmHttpClient.post(SESSIONS_PATH) {
                    header("X-Request-ID", requestId)
                    setBody(SessionCreateRequest(chatId = chatId, namespace = LLM_NAMESPACE))
                }
            },
            onSuccess = { response ->
                val result = response.body<SessionCreateRestResponse>()
                SessionCreateResponse(
                    sessionId = result.sessionId,
                    model = result.model,
                    created = result.created,
                )
            },
            onFailure = { error ->
                SessionCreateResponse(isError = true, errorMessage = error.message)
            },
        )
    }

    /** LLM 세션 종료 */
    suspend fun endSession(sessionId: String): Boolean {
        val requestId = RestErrorHandler.generateRequestId()
        return restErrorHandler.executeRestCall(
            operation = "END_SESSION",
            requestId = requestId,
            request = {
                llmHttpClient.delete("$SESSIONS_PATH/$sessionId") {
                    header("X-Request-ID", requestId)
                }
            },
            onSuccess = { response -> response.body<SessionEndRestResponse>().removed },
            onFailure = { false },
        )
    }

    /** chatId 기반 파생 세션 종료 (namespace:chatId) */
    suspend fun endSessionByChat(chatId: String): Boolean {
        val derivedSessionId = "$LLM_NAMESPACE:$chatId"
        return endSession(derivedSessionId)
    }

    /** 힌트 생성 */
    suspend fun generateHints(
        target: String,
        category: String,
        details: Map<String, Any>? = null,
    ): TwentyQHintsResponse {
        val requestId = RestErrorHandler.generateRequestId()
        return restErrorHandler.executeRestCall(
            operation = "GENERATE_HINTS",
            requestId = requestId,
            request = {
                llmHttpClient.post(HINTS_PATH) {
                    header("X-Request-ID", requestId)
                    setBody(HintsRequest(target = target, category = category, details = details))
                }
            },
            onSuccess = { response ->
                val result = response.body<HintsRestResponse>()
                TwentyQHintsResponse(
                    hints = result.hints,
                    thoughtSignature = result.thoughtSignature,
                )
            },
            onFailure = { error ->
                TwentyQHintsResponse(isError = true, errorMessage = error.message)
            },
        )
    }

    /** 질문 답변 */
    suspend fun answerQuestion(
        chatId: String,
        target: String,
        category: String,
        question: String,
        details: Map<String, Any>? = null,
    ): TwentyQAnswerResponse {
        val requestId = RestErrorHandler.generateRequestId()
        return restErrorHandler.executeRestCall(
            operation = "ANSWER_QUESTION",
            requestId = requestId,
            request = {
                llmHttpClient.post(ANSWERS_PATH) {
                    header("X-Request-ID", requestId)
                    setBody(
                        AnswerRequest(
                            chatId = chatId,
                            namespace = LLM_NAMESPACE,
                            target = target,
                            category = category,
                            question = question,
                            details = details,
                        ),
                    )
                }
            },
            onSuccess = { response ->
                val result = response.body<AnswerRestResponse>()
                TwentyQAnswerResponse(
                    scale = result.scale,
                    rawText = result.rawText,
                    thoughtSignature = result.thoughtSignature,
                )
            },
            onFailure = { error ->
                TwentyQAnswerResponse(isError = true, errorMessage = error.message)
            },
        )
    }

    /** 정답 검증 */
    suspend fun verifyGuess(
        target: String,
        guess: String,
    ): TwentyQVerifyResponse {
        val requestId = RestErrorHandler.generateRequestId()
        return restErrorHandler.executeRestCall(
            operation = "VERIFY_GUESS",
            requestId = requestId,
            request = {
                llmHttpClient.post(VERIFICATIONS_PATH) {
                    header("X-Request-ID", requestId)
                    setBody(VerifyRequest(target = target, guess = guess))
                }
            },
            onSuccess = { response ->
                val result = response.body<VerifyRestResponse>()
                TwentyQVerifyResponse(
                    result = result.result,
                    rawText = result.rawText,
                )
            },
            onFailure = { error ->
                TwentyQVerifyResponse(isError = true, errorMessage = error.message)
            },
        )
    }

    /** 질문 정규화 */
    suspend fun normalizeQuestion(question: String): TwentyQNormalizeResponse {
        val requestId = RestErrorHandler.generateRequestId()
        return restErrorHandler.executeRestCall(
            operation = "NORMALIZE_QUESTION",
            requestId = requestId,
            request = {
                llmHttpClient.post(NORMALIZATIONS_PATH) {
                    header("X-Request-ID", requestId)
                    setBody(NormalizeRequest(question = question))
                }
            },
            onSuccess = { response ->
                val result = response.body<NormalizeRestResponse>()
                TwentyQNormalizeResponse(
                    normalized = result.normalized,
                    original = result.original,
                )
            },
            onFailure = { error ->
                TwentyQNormalizeResponse(
                    original = question,
                    isError = true,
                    errorMessage = error.message,
                )
            },
        )
    }

    /** 동의어 체크 */
    suspend fun checkSynonym(
        target: String,
        guess: String,
    ): TwentyQSynonymResponse {
        val requestId = RestErrorHandler.generateRequestId()
        return restErrorHandler.executeRestCall(
            operation = "CHECK_SYNONYM",
            requestId = requestId,
            request = {
                llmHttpClient.post(SYNONYM_CHECKS_PATH) {
                    header("X-Request-ID", requestId)
                    setBody(SynonymRequest(target = target, guess = guess))
                }
            },
            onSuccess = { response ->
                val result = response.body<SynonymRestResponse>()
                TwentyQSynonymResponse(
                    result = result.result,
                    rawText = result.rawText,
                )
            },
            onFailure = { error ->
                TwentyQSynonymResponse(isError = true, errorMessage = error.message)
            },
        )
    }
}
