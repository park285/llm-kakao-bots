package party.qwer.twentyq.bridge

import party.qwer.twentyq.model.OutboundMessage
import party.qwer.twentyq.model.PendingMessage
import party.qwer.twentyq.util.game.GameMessageProvider
import party.qwer.twentyq.util.game.extensions.displayName

/** 메시지 큐 처리 알림 담당 */
internal class MessageQueueNotifier(
    private val messageProvider: GameMessageProvider,
    private val exceptionHandler: MessageExceptionHandler,
) {
    /** 큐 처리 시작 알림 */
    suspend fun notifyProcessingStart(
        chatId: String,
        pending: PendingMessage,
        emit: suspend (OutboundMessage) -> Unit,
    ) {
        val userName = pending.displayName(chatId, messageProvider.get("user.anonymous"))
        val notifyText = messageProvider.get("queue.processing", "user" to userName)
        emit(OutboundMessage.Waiting(chatId, notifyText, pending.threadId))
    }

    /** Lock 획득 실패 시 재큐잉 알림 */
    suspend fun notifyRetry(
        chatId: String,
        pending: PendingMessage,
        emit: suspend (OutboundMessage) -> Unit,
    ) {
        val userName = pending.displayName(chatId, messageProvider.get("user.anonymous"))
        val retryText = messageProvider.get("queue.retry", "user" to userName)
        emit(OutboundMessage.Waiting(chatId, retryText, pending.threadId))
    }

    /** 재큐잉 시 중복 발견 알림 */
    suspend fun notifyDuplicate(
        chatId: String,
        pending: PendingMessage,
        emit: suspend (OutboundMessage) -> Unit,
    ) {
        val userName = pending.displayName(chatId, messageProvider.get("user.anonymous"))
        val duplicateText = messageProvider.get("queue.retry_duplicate", "user" to userName)
        emit(OutboundMessage.Waiting(chatId, duplicateText, pending.threadId))
    }

    /** 재큐잉 실패 알림 (대기열 가득 참) */
    suspend fun notifyFailed(
        chatId: String,
        pending: PendingMessage,
        emit: suspend (OutboundMessage) -> Unit,
    ) {
        val userName = pending.displayName(chatId, messageProvider.get("user.anonymous"))
        val failedText = messageProvider.get("queue.retry_failed", "user" to userName)
        emit(OutboundMessage.Error(chatId, failedText, pending.threadId))
    }

    /** 큐 처리 중 에러 알림 */
    suspend fun notifyError(
        chatId: String,
        pending: PendingMessage,
        ex: Throwable,
        emit: suspend (OutboundMessage) -> Unit,
    ) {
        val exception = ex as? Exception ?: Exception(ex.message)
        exceptionHandler.logException(exception, chatId, pending.userId, "QUEUE")
        val errorMessage = exceptionHandler.getMessageForException(exception)
        emit(OutboundMessage.Error(chatId, errorMessage, pending.threadId))
    }
}
