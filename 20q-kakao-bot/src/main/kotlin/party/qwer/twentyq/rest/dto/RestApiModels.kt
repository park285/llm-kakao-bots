package party.qwer.twentyq.rest.dto

import com.fasterxml.jackson.annotation.JsonProperty

/** 사용량 응답 */
data class UsageResponse(
    @field:JsonProperty("input_tokens")
    val inputTokens: Int = 0,
    @field:JsonProperty("output_tokens")
    val outputTokens: Int = 0,
    @field:JsonProperty("total_tokens")
    val totalTokens: Int = 0,
    @field:JsonProperty("reasoning_tokens")
    val reasoningTokens: Int? = null,
    @field:JsonProperty("model")
    val model: String? = null,
)

/** 세션 생성 요청 */
data class SessionCreateRequest(
    @field:JsonProperty("session_id")
    val sessionId: String? = null,
    @field:JsonProperty("chat_id")
    val chatId: String? = null,
    val namespace: String? = null,
)

/** 세션 생성 응답 (REST) */
data class SessionCreateRestResponse(
    @field:JsonProperty("session_id")
    val sessionId: String = "",
    val model: String = "",
    val created: Boolean = false,
)

/** 세션 생성 결과 */
data class SessionCreateResponse(
    val sessionId: String = "",
    val model: String = "",
    val created: Boolean = false,
    val isError: Boolean = false,
    val errorMessage: String? = null,
)

/** 세션 종료 응답 */
data class SessionEndRestResponse(
    @field:JsonProperty("session_id")
    val sessionId: String = "",
    val removed: Boolean = false,
)

/** 힌트 생성 요청 */
data class HintsRequest(
    val target: String,
    val category: String,
    val details: Map<String, Any>? = null,
)

/** 질문 답변 요청 */
data class AnswerRequest(
    @field:JsonProperty("session_id")
    val sessionId: String? = null,
    @field:JsonProperty("chat_id")
    val chatId: String? = null,
    val namespace: String? = null,
    val target: String,
    val category: String,
    val question: String,
    val details: Map<String, Any>? = null,
)

/** 정답 검증 요청 */
data class VerifyRequest(
    val target: String,
    val guess: String,
)

/** 질문 정규화 요청 */
data class NormalizeRequest(
    val question: String,
)

/** 동의어 체크 요청 */
data class SynonymRequest(
    val target: String,
    val guess: String,
)

/** 힌트 생성 응답 (REST) */
data class HintsRestResponse(
    val hints: List<String> = emptyList(),
    @field:JsonProperty("thought_signature")
    val thoughtSignature: String? = null,
)

/** 질문 답변 응답 (REST) */
data class AnswerRestResponse(
    val scale: String? = null,
    @field:JsonProperty("raw_text")
    val rawText: String = "",
    @field:JsonProperty("thought_signature")
    val thoughtSignature: String? = null,
)

/** 정답 검증 응답 (REST) */
data class VerifyRestResponse(
    val result: String? = null,
    @field:JsonProperty("raw_text")
    val rawText: String = "",
)

/** 질문 정규화 응답 (REST) */
data class NormalizeRestResponse(
    val normalized: String = "",
    val original: String = "",
)

/** 동의어 체크 응답 (REST) */
data class SynonymRestResponse(
    val result: String? = null,
    @field:JsonProperty("raw_text")
    val rawText: String = "",
)

/** Guard 평가 요청 */
data class GuardRequest(
    @field:JsonProperty("input_text")
    val inputText: String,
)

/** Guard 평가 응답 */
data class GuardEvaluateResponse(
    val malicious: Boolean,
    val score: Double,
    val threshold: Double = 0.7,
    val hits: List<GuardHit> = emptyList(),
)

/** Guard 규칙 매칭 정보 */
data class GuardHit(
    @field:JsonProperty("id")
    val id: String,
    val weight: Double = 0.0,
)

/** Guard 악성 여부 응답 */
data class GuardMaliciousResponse(
    val malicious: Boolean,
)

/** NLP 분석 요청 */
data class NlpRequest(
    val text: String,
)

/** NLP 토큰 정보 */
data class NlpToken(
    val form: String,
    val tag: String,
    val position: Int = 0,
    val length: Int = 0,
)

/** NLP 이상치 점수 응답 */
data class NlpAnomalyResponse(
    val score: Double,
)

/** NLP 휴리스틱 응답 */
data class NlpHeuristicsResponse(
    @field:JsonProperty("numeric_quantifier")
    val numericQuantifier: Boolean = false,
    @field:JsonProperty("unit_noun")
    val unitNoun: Boolean = false,
    @field:JsonProperty("boundary_ref")
    val boundaryRef: Boolean = false,
    @field:JsonProperty("comparison_word")
    val comparisonWord: Boolean = false,
)

// === Error Response Models ===

/** mcp-llm-server 에러 응답 */
data class LlmErrorResponse(
    @field:JsonProperty("error_code")
    val errorCode: String,
    @field:JsonProperty("error_type")
    val errorType: String,
    val message: String,
    @field:JsonProperty("request_id")
    val requestId: String? = null,
    val details: Map<String, Any>? = null,
)

/** 파싱된 에러 정보 */
data class ParsedError(
    val statusCode: Int,
    val errorCode: String?,
    val errorType: String?,
    val message: String,
    val requestId: String?,
) {
    /** 로그용 문자열 */
    fun toLogString(): String =
        buildString {
            append("status=$statusCode")
            errorCode?.let { append(", error_code=$it") }
            errorType?.let { append(", error_type=$it") }
            requestId?.let { append(", request_id=$it") }
            append(", message=$message")
        }
}

// === Usage API Models ===

/** 일별 사용량 응답 */
data class DailyUsageResponse(
    @field:JsonProperty("usage_date")
    val usageDate: String = "",
    @field:JsonProperty("input_tokens")
    val inputTokens: Long = 0,
    @field:JsonProperty("output_tokens")
    val outputTokens: Long = 0,
    @field:JsonProperty("total_tokens")
    val totalTokens: Long = 0,
    @field:JsonProperty("reasoning_tokens")
    val reasoningTokens: Long = 0,
    @field:JsonProperty("request_count")
    val requestCount: Long = 0,
    @field:JsonProperty("model")
    val model: String? = null,
)

/** 사용량 리스트 응답 */
data class UsageListResponse(
    val usages: List<DailyUsageResponse> = emptyList(),
    @field:JsonProperty("total_input_tokens")
    val totalInputTokens: Long = 0,
    @field:JsonProperty("total_output_tokens")
    val totalOutputTokens: Long = 0,
    @field:JsonProperty("total_tokens")
    val totalTokens: Long = 0,
    @field:JsonProperty("total_request_count")
    val totalRequestCount: Long = 0,
    @field:JsonProperty("model")
    val model: String? = null,
)
