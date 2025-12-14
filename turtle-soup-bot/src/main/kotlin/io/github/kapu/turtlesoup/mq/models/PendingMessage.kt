package io.github.kapu.turtlesoup.mq.models

import io.github.kapu.turtlesoup.models.InstantSerializer
import kotlinx.serialization.Serializable
import java.time.Instant

/** 대기 중인 메시지 */
@Serializable
data class PendingMessage(
    val userId: String,
    val content: String,
    val threadId: String? = null,
    val sender: String? = null,
    @Serializable(with = InstantSerializer::class)
    val enqueuedAt: Instant = Instant.now(),
    val timestamp: Long = System.currentTimeMillis(),
) {
    /** 표시용 이름 (sender 우선, 없으면 userId, 둘 다 없으면 기본값) */
    fun displayName(
        chatId: String,
        anonymousName: String,
    ): String =
        sender?.takeIf { it.isNotBlank() }
            ?: userId.takeIf { it.isNotBlank() && it != chatId }
            ?: anonymousName
}
