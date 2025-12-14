package party.qwer.twentyq.api.dto

/** API 에러 응답 DTO */
data class ErrorResponse(
    val error: String,
    val message: String,
    val details: Map<String, Any>? = null,
)
