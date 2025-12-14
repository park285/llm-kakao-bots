package party.qwer.twentyq.service.dto

import party.qwer.twentyq.model.FiveScaleKo

enum class AnswerSource {
    ENUM_SCHEMA_PRIMARY,
    ENUM_SCHEMA_RETRY_STRICT,
    FALLBACK_DEFAULT,
}

/**
 * LLM 답변 원시 응답 (파싱 전)
 *
 * - scale: 파싱된 5단계 척도 (실패 시 null)
 * - thoughtSignature: SDK 반환 사고 서명 (없으면 null)
 */
data class LlmAnswerResponse(
    val scale: FiveScaleKo?,
    val thoughtSignature: String?,
)

data class AnswerResult(
    val scale: FiveScaleKo,
    val source: AnswerSource,
    val guardDegraded: Boolean,
    val isWrongGuess: Boolean = false,
    val guessedAnswer: String? = null,
    val isCorrect: Boolean = false,
    val isCloseCall: Boolean = false,
    val successMessage: String? = null,
)
