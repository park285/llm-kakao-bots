package party.qwer.twentyq.model

/** LLM 프롬프트 전용 DTO - intro 제외, details 평탄화 */
data class SecretForHint(
    val target: String,
    val category: String,
    val details: Map<String, Any>? = null,
)
