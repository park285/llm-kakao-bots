package io.github.kapu.turtlesoup.mq

import io.github.kapu.turtlesoup.mq.models.DequeueResult
import io.github.kapu.turtlesoup.mq.models.EnqueueResult
import io.github.kapu.turtlesoup.mq.models.PendingMessage
import io.github.kapu.turtlesoup.redis.PendingMessageStore
import io.github.oshai.kotlinlogging.KotlinLogging

/** 메시지 큐 조정자 */
class MessageQueueCoordinator(
    private val store: PendingMessageStore,
) {
    companion object {
        private val logger = KotlinLogging.logger {}
    }

    /** 메시지 큐잉 */
    suspend fun enqueue(
        chatId: String,
        message: PendingMessage,
    ): EnqueueResult {
        val result = store.enqueue(chatId, message)
        when (result) {
            EnqueueResult.QUEUE_FULL ->
                logger.warn { "enqueue_failed chat_id=$chatId user_id=${message.userId} reason=QUEUE_FULL" }
            EnqueueResult.DUPLICATE ->
                logger.debug { "enqueue_failed chat_id=$chatId user_id=${message.userId} reason=DUPLICATE" }
            else -> Unit
        }
        return result
    }

    /** 메시지 디큐 */
    suspend fun dequeue(chatId: String): DequeueResult = store.dequeue(chatId)

    /** 대기 중인 메시지 존재 여부 */
    suspend fun hasPending(chatId: String): Boolean = store.hasPending(chatId)

    /** 큐 크기 */
    suspend fun size(chatId: String): Int = store.size(chatId)

    /** 큐 상세 정보 */
    suspend fun getQueueDetails(chatId: String): String = store.getQueueDetails(chatId)

    /** 큐 초기화 */
    suspend fun clear(chatId: String) {
        store.clear(chatId)
        logger.debug { "queue_cleared chat_id=$chatId" }
    }
}
