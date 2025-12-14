package io.github.kapu.turtlesoup.mq.models

import kotlinx.serialization.Serializable

/**
 * Iris로 전송할 응답 메시지
 */
sealed interface OutboundMessage {
    val chatId: String
    val text: String
    val threadId: String?

    /**
     * 대기 메시지 (AI 처리 중)
     */
    @Serializable
    data class Waiting(
        override val chatId: String,
        override val text: String,
        override val threadId: String? = null,
    ) : OutboundMessage

    /**
     * 최종 응답 메시지
     */
    @Serializable
    data class Final(
        override val chatId: String,
        override val text: String,
        override val threadId: String? = null,
    ) : OutboundMessage

    /**
     * 오류 메시지
     */
    @Serializable
    data class Error(
        override val chatId: String,
        override val text: String,
        override val threadId: String? = null,
    ) : OutboundMessage
}
