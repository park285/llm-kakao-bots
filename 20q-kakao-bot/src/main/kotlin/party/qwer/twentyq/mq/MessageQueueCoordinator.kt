package party.qwer.twentyq.mq

import org.slf4j.LoggerFactory
import org.springframework.stereotype.Component
import party.qwer.twentyq.logging.debugL
import party.qwer.twentyq.model.PendingMessage
import party.qwer.twentyq.mq.queue.EnqueueResult
import party.qwer.twentyq.mq.queue.PendingMessageStore

@Component
class MessageQueueCoordinator(
    private val store: PendingMessageStore,
) {
    companion object {
        private val log = LoggerFactory.getLogger(MessageQueueCoordinator::class.java)
    }

    suspend fun enqueue(
        chatId: String,
        message: PendingMessage,
    ): EnqueueResult {
        val result = store.enqueue(chatId, message)
        when (result) {
            EnqueueResult.QUEUE_FULL ->
                log.warn("ENQUEUE_FAILED chatId={}, userId={}, reason=QUEUE_FULL", chatId, message.userId)
            EnqueueResult.DUPLICATE ->
                log.warn("ENQUEUE_FAILED chatId={}, userId={}, reason=DUPLICATE", chatId, message.userId)
            else -> Unit
        }
        return result
    }

    suspend fun dequeue(chatId: String): PendingMessage? = store.dequeue(chatId)

    suspend fun hasPending(chatId: String): Boolean = store.hasPending(chatId)

    suspend fun size(chatId: String): Int = store.size(chatId)

    suspend fun getQueueDetails(chatId: String): String = store.getQueueDetails(chatId)

    suspend fun hasChainSkipFlag(
        chatId: String,
        userId: String,
    ): Boolean = store.hasChainSkipFlag(chatId, userId)

    suspend fun setChainSkipFlag(
        chatId: String,
        userId: String,
    ) = store.setChainSkipFlag(chatId, userId)

    suspend fun clear(chatId: String) {
        store.clear(chatId)
        log.debugL { "QUEUE_CLEARED chatId=$chatId" }
    }
}
