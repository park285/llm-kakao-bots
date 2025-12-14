package io.github.kapu.turtlesoup.mq

import io.github.kapu.turtlesoup.mq.models.OutboundMessage
import io.github.kapu.turtlesoup.mq.models.PendingMessage
import io.github.kapu.turtlesoup.utils.MessageKeys
import io.github.kapu.turtlesoup.utils.MessageProvider
import io.github.kapu.turtlesoup.utils.TurtleSoupException
import io.github.oshai.kotlinlogging.KotlinLogging

/** 메시지 큐 처리 알림 담당 */
class MessageQueueNotifier(
    private val messageProvider: MessageProvider,
) {
    companion object {
        private val logger = KotlinLogging.logger {}
    }

    /** 큐 처리 시작 알림 */
    suspend fun notifyProcessingStart(
        chatId: String,
        pending: PendingMessage,
        emit: suspend (OutboundMessage) -> Unit,
    ) {
        val userName = pending.displayName(chatId, messageProvider.get(MessageKeys.USER_ANONYMOUS))
        val notifyText = messageProvider.get(MessageKeys.QUEUE_PROCESSING, "user" to userName)
        emit(OutboundMessage.Waiting(chatId, notifyText, pending.threadId))
    }

    /** Lock 획득 실패 시 재큐잉 알림 */
    suspend fun notifyRetry(
        chatId: String,
        pending: PendingMessage,
        emit: suspend (OutboundMessage) -> Unit,
    ) {
        val userName = pending.displayName(chatId, messageProvider.get(MessageKeys.USER_ANONYMOUS))
        val retryText = messageProvider.get(MessageKeys.QUEUE_RETRY, "user" to userName)
        emit(OutboundMessage.Waiting(chatId, retryText, pending.threadId))
    }

    /** 재큐잉 시 중복 발견 알림 */
    suspend fun notifyDuplicate(
        chatId: String,
        pending: PendingMessage,
        emit: suspend (OutboundMessage) -> Unit,
    ) {
        val userName = pending.displayName(chatId, messageProvider.get(MessageKeys.USER_ANONYMOUS))
        val duplicateText = messageProvider.get(MessageKeys.QUEUE_RETRY_DUPLICATE, "user" to userName)
        emit(OutboundMessage.Waiting(chatId, duplicateText, pending.threadId))
    }

    /** 재큐잉 실패 알림 (대기열 가득 참) */
    suspend fun notifyFailed(
        chatId: String,
        pending: PendingMessage,
        emit: suspend (OutboundMessage) -> Unit,
    ) {
        val userName = pending.displayName(chatId, messageProvider.get(MessageKeys.USER_ANONYMOUS))
        val failedText = messageProvider.get(MessageKeys.QUEUE_RETRY_FAILED, "user" to userName)
        emit(OutboundMessage.Error(chatId, failedText, pending.threadId))
    }

    /** 큐 처리 중 에러 알림 */
    suspend fun notifyError(
        chatId: String,
        pending: PendingMessage,
        ex: Throwable,
        emit: suspend (OutboundMessage) -> Unit,
    ) {
        logger.error(ex) { "queue_processing_error chat_id=$chatId user_id=${pending.userId}" }
        val errorMessage =
            when (ex) {
                is TurtleSoupException -> {
                    val mapping = ExceptionMapper.getErrorMapping(ex)
                    messageProvider.get(mapping.key, *mapping.params)
                }
                else -> messageProvider.get(MessageKeys.ERROR_INTERNAL)
            }
        emit(OutboundMessage.Error(chatId, errorMessage, pending.threadId))
    }
}
