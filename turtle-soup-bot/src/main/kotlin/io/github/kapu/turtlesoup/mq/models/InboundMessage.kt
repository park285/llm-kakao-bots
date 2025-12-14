package io.github.kapu.turtlesoup.mq.models

import kotlinx.serialization.Serializable

/**
 * Iris로부터 수신한 입력 메시지
 */
@Serializable
data class InboundMessage(
    val chatId: String,
    val userId: String,
    val content: String,
    val threadId: String? = null,
    val sender: String? = null,
)
