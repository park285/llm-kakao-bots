package party.qwer.twentyq.bridge

import org.slf4j.LoggerFactory
import org.springframework.stereotype.Component
import party.qwer.twentyq.model.OutboundMessage
import party.qwer.twentyq.util.common.extensions.chunkedByLines

/**
 * 메시지 청킹 및 전송 담당 클래스
 */
@Component
class MessageEmitter {
    companion object {
        private val log = LoggerFactory.getLogger(MessageEmitter::class.java)
        private const val DEFAULT_CHUNK_LENGTH = 500
    }

    suspend fun sendChunkedReply(
        response: String,
        chatId: String,
        threadId: String?,
        emit: suspend (OutboundMessage) -> Unit,
    ) {
        val chunks = response.chunkedByLines(DEFAULT_CHUNK_LENGTH)
        log.info(
            "EMIT_CHECK chatId={}, response_len={}, chunks_count={}",
            chatId,
            response.length,
            chunks.size,
        )

        if (chunks.isEmpty()) {
            emit(OutboundMessage.Final(chatId, "", threadId))
        } else {
            chunks.forEachIndexed { idx, chunk ->
                val isLast = idx == chunks.lastIndex
                if (isLast) {
                    emit(OutboundMessage.Final(chatId, chunk, threadId))
                } else {
                    emit(OutboundMessage.Waiting(chatId, chunk, threadId))
                }
            }
        }
    }
}
