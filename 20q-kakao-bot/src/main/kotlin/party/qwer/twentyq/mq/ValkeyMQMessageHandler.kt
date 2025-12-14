package party.qwer.twentyq.mq

import org.redisson.api.StreamMessageId
import org.slf4j.LoggerFactory
import org.springframework.stereotype.Component
import party.qwer.twentyq.bridge.TwentyQMessageService
import party.qwer.twentyq.logging.debugL
import party.qwer.twentyq.model.InboundMessage
import party.qwer.twentyq.util.logging.LoggingConstants.LOG_TEXT_SHORT
import tools.jackson.core.JacksonException
import tools.jackson.module.kotlin.readValue

/**
 * Valkey MQ 메시지 핸들러
 */
@Component
class ValkeyMQMessageHandler(
    private val messageService: TwentyQMessageService,
    private val replyPublisher: ValkeyMQReplyPublisher,
    @param:org.springframework.beans.factory.annotation.Qualifier("kotlinJsonMapper")
    private val objectMapper: tools.jackson.databind.ObjectMapper,
) {
    companion object {
        private val log = LoggerFactory.getLogger(ValkeyMQMessageHandler::class.java)
    }

    suspend fun handleMessage(
        streamKey: String,
        id: StreamMessageId,
        fields: Map<String, String>,
    ) {
        log.debugL {
            "VALKEY_MQ_ALL_FIELDS streamKey=$streamKey, id=$id, fields=$fields"
        }
        val text = fields["text"].orEmpty()
        val room = fields["room"].orEmpty()
        val threadId = fields["threadId"]
        val userId = extractUserId(fields)

        if (text.isBlank() || room.isBlank()) {
            log.warn(
                "VALKEY_MQ_MESSAGE_SKIPPED streamKey={}, id={}, reason=BLANK room/text",
                streamKey,
                id,
            )
            return
        }

        val sender = fields["sender"]

        log.info(
            "VALKEY_MQ_MESSAGE_RECEIVED streamKey={}, id={}, room={}, userId={}, sender={}, textPreview={}",
            streamKey,
            id,
            room,
            userId,
            sender,
            text.take(LOG_TEXT_SHORT),
        )

        val inbound =
            InboundMessage(
                chatId = room,
                userId = userId,
                content = text,
                threadId = threadId,
                sender = sender,
            )

        messageService.handle(inbound) { outbound ->
            replyPublisher.publish(outbound)
        }
    }

    private fun extractUserId(fields: Map<String, String>): String {
        val sender = fields["sender"].orEmpty()
        val room = fields["room"].orEmpty()

        // rawJson 파싱 시도
        fun tryExtractFromRawJson(): String? {
            val rawJson = fields["rawJson"] ?: return null
            if (rawJson.isBlank()) return null

            return try {
                val root: Map<String, Any?> = objectMapper.readValue(rawJson)
                (root["json"] as? Map<*, *>)?.get("user_id")?.toString()
            } catch (e: JacksonException) {
                log.warn("VALKEY_MQ_RAWJSON_PARSE_FAILED error={}", e.message)
                null
            }
        }

        // 우선순위: userId -> user_id -> rawJson -> sender -> room
        return fields["userId"]
            ?: fields["user_id"]
            ?: tryExtractFromRawJson()
            ?: sender.ifBlank { room }
    }
}
