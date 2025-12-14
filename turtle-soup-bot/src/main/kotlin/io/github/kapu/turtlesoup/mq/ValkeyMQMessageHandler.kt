package io.github.kapu.turtlesoup.mq

import io.github.kapu.turtlesoup.config.StreamKeys
import io.github.kapu.turtlesoup.mq.models.InboundMessage
import io.github.oshai.kotlinlogging.KotlinLogging

/** Stream 메시지를 InboundMessage로 변환 및 처리 */
class ValkeyMQMessageHandler(
    private val gameMessageService: GameMessageService,
) {
    /**
     * Stream 메시지 처리
     */
    suspend fun handleStreamMessage(
        messageId: String,
        fields: Map<String, String>,
    ) {
        try {
            val inboundMessage = parseInboundMessage(fields)

            logger.info {
                "message_received " +
                    "message_id=$messageId " +
                    "chat_id=${inboundMessage.chatId} " +
                    "user_id=${inboundMessage.userId}"
            }

            gameMessageService.handleMessage(inboundMessage)
        } catch (e: IllegalStateException) {
            logger.error(e) {
                "message_handling_failed message_id=$messageId"
            }
        }
    }

    /**
     * Stream 필드를 InboundMessage로 변환
     * Iris 필드: text, room, sender, threadId, rawJson
     */
    private fun parseInboundMessage(fields: Map<String, String>): InboundMessage {
        // Iris: room, 20q-kakao-bot: chatId
        val chatId =
            fields[IRIS_FIELD_ROOM]
                ?: fields[StreamKeys.FIELD_CHAT_ID]
                ?: throw IllegalArgumentException("Missing room/chatId field")

        // Iris: text, 20q-kakao-bot: content
        val content =
            fields[IRIS_FIELD_TEXT]
                ?: fields[StreamKeys.FIELD_CONTENT]
                ?: throw IllegalArgumentException("Missing text/content field")

        // userId: rawJson에서 추출 시도, fallback으로 sender 사용
        val userId =
            extractUserIdFromRawJson(fields[IRIS_FIELD_RAW_JSON])
                ?: fields[StreamKeys.FIELD_USER_ID]
                ?: fields[StreamKeys.FIELD_SENDER]
                ?: chatId

        val threadId = fields[StreamKeys.FIELD_THREAD_ID]
        val sender = fields[StreamKeys.FIELD_SENDER]

        return InboundMessage(
            chatId = chatId,
            userId = userId,
            content = content,
            threadId = threadId,
            sender = sender,
        )
    }

    // rawJson에서 user_id 추출
    private fun extractUserIdFromRawJson(rawJson: String?): String? {
        if (rawJson.isNullOrBlank()) return null
        return try {
            val regex = """"user_id"\s*:\s*"?(\d+)"?""".toRegex()
            regex.find(rawJson)?.groupValues?.get(1)
        } catch (_: Exception) {
            null
        }
    }

    companion object {
        private val logger = KotlinLogging.logger {}

        // Iris 필드명
        private const val IRIS_FIELD_TEXT = "text"
        private const val IRIS_FIELD_ROOM = "room"
        private const val IRIS_FIELD_RAW_JSON = "rawJson"
    }
}
