package party.qwer.twentyq.service.riddle

/** 답변 검증 응답 상수 */
object VerifyAnswerResponse {
    const val RESPONSE_ACCEPT = "ACCEPT"
    const val RESPONSE_REJECT = "REJECT"
    const val RESPONSE_CLOSE = "CLOSE"

    val RESPONSE_SCHEMA: Map<String, Any> =
        mapOf(
            "type" to "string",
            "enum" to listOf(RESPONSE_ACCEPT, RESPONSE_REJECT, RESPONSE_CLOSE),
        )
}
