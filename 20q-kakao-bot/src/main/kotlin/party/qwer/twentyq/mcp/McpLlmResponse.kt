package party.qwer.twentyq.mcp

/** LLM 응답 데이터 */
data class LlmResponse(
    val text: String,
    val thoughtSignature: String? = null,
    val finishReason: String? = null,
    val isError: Boolean = false,
    val errorMessage: String? = null,
)

/** Structured Output 응답 */
data class StructuredResponse<T>(
    val data: T?,
    val isError: Boolean = false,
    val errorMessage: String? = null,
)

/** 토큰 사용량 */
data class TokenUsage(
    val inputTokens: Int = 0,
    val outputTokens: Int = 0,
    val totalTokens: Int = 0,
    val reasoningTokens: Int = 0,
    val model: String? = null,
)

/** Guard 평가 결과 */
data class GuardEvaluation(
    val isMalicious: Boolean,
    val score: Double,
    val reason: String? = null,
    val categories: List<String> = emptyList(),
)

/** NLP 분석 결과 */
data class NlpAnalysis(
    val tokens: List<String> = emptyList(),
    val posTag: List<String> = emptyList(),
    val anomalyScore: Double = 0.0,
    val heuristics: Map<String, Any> = emptyMap(),
)

/** 세션 정보 */
data class SessionInfo(
    val sessionId: String,
    val messageCount: Int = 0,
    val createdAt: Long = 0,
    val isActive: Boolean = true,
)

// Twenty Questions 도메인 응답

/** 힌트 생성 응답 */
data class TwentyQHintsResponse(
    val hints: List<String> = emptyList(),
    val thoughtSignature: String? = null,
    val isError: Boolean = false,
    val errorMessage: String? = null,
)

/** 질문 답변 응답 */
data class TwentyQAnswerResponse(
    val scale: String? = null,
    val rawText: String = "",
    val thoughtSignature: String? = null,
    val isError: Boolean = false,
    val errorMessage: String? = null,
)

/** 정답 검증 응답 (ACCEPT/CLOSE/REJECT) */
data class TwentyQVerifyResponse(
    val result: String? = null,
    val rawText: String = "",
    val isError: Boolean = false,
    val errorMessage: String? = null,
)

/** 질문 정규화 응답 */
data class TwentyQNormalizeResponse(
    val normalized: String = "",
    val original: String = "",
    val isError: Boolean = false,
    val errorMessage: String? = null,
)

/** 동의어 체크 응답 (EQUIVALENT/NOT_EQUIVALENT) */
data class TwentyQSynonymResponse(
    val result: String? = null,
    val rawText: String = "",
    val isError: Boolean = false,
    val errorMessage: String? = null,
)
