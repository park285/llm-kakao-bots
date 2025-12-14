package io.github.kapu.turtlesoup.mq

import io.github.kapu.turtlesoup.models.Command
import io.github.kapu.turtlesoup.models.waitingMessageKey
import io.github.kapu.turtlesoup.mq.models.InboundMessage
import io.github.kapu.turtlesoup.mq.models.OutboundMessage
import io.github.kapu.turtlesoup.utils.MessageKeys
import io.github.kapu.turtlesoup.utils.MessageProvider
import io.github.kapu.turtlesoup.utils.chunkedByLines

/** 메시지 발송 헬퍼 */
class MessageSender(
    private val messageProvider: MessageProvider,
    private val publisher: ValkeyMQReplyPublisher,
) {
    suspend fun sendFinal(
        message: InboundMessage,
        text: String,
    ) {
        val chunks = text.chunkedByLines()
        if (chunks.isEmpty()) {
            publisher.publish(OutboundMessage.Final(message.chatId, "", message.threadId))
            return
        }
        chunks.forEachIndexed { idx, chunk ->
            val isLast = idx == chunks.lastIndex
            if (isLast) {
                publisher.publish(OutboundMessage.Final(message.chatId, chunk, message.threadId))
            } else {
                publisher.publish(OutboundMessage.Waiting(message.chatId, chunk, message.threadId))
            }
        }
    }

    suspend fun sendWaiting(
        message: InboundMessage,
        command: Command,
    ) {
        val waitingKey = command.waitingMessageKey ?: return
        publisher.publish(
            OutboundMessage.Waiting(
                chatId = message.chatId,
                text = messageProvider.get(waitingKey),
                threadId = message.threadId,
            ),
        )
    }

    suspend fun sendError(
        message: InboundMessage,
        errorMapping: ErrorMapping,
    ) {
        publisher.publish(
            OutboundMessage.Error(
                chatId = message.chatId,
                text = messageProvider.get(errorMapping.key, *errorMapping.params),
                threadId = message.threadId,
            ),
        )
    }

    suspend fun sendLockError(
        message: InboundMessage,
        holderName: String?,
    ) {
        val text =
            if (holderName != null) {
                messageProvider.get(
                    MessageKeys.LOCK_REQUEST_IN_PROGRESS_WITH_HOLDER,
                    "holder" to holderName,
                )
            } else {
                messageProvider.get(MessageKeys.LOCK_REQUEST_IN_PROGRESS)
            }
        publisher.publish(
            OutboundMessage.Error(
                chatId = message.chatId,
                text = text,
                threadId = message.threadId,
            ),
        )
    }
}
