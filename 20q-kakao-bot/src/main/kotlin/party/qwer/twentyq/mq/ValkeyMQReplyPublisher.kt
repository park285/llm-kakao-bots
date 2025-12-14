package party.qwer.twentyq.mq

import org.redisson.api.RStreamReactive
import org.redisson.api.RedissonReactiveClient
import org.redisson.api.stream.StreamAddArgs
import org.redisson.client.codec.StringCodec
import org.slf4j.LoggerFactory
import org.springframework.beans.factory.annotation.Qualifier
import org.springframework.stereotype.Component
import party.qwer.twentyq.config.AppProperties
import party.qwer.twentyq.logging.LoggingExtensions.sampled
import party.qwer.twentyq.model.OutboundMessage
import party.qwer.twentyq.redis.awaitSingleOrNull

@Component
class ValkeyMQReplyPublisher(
    private val appProperties: AppProperties,
    @param:Qualifier("redissonMQReactiveClient")
    private val redissonMQReactiveClient: RedissonReactiveClient,
) {
    companion object {
        private val log = LoggerFactory.getLogger(ValkeyMQReplyPublisher::class.java)

        private const val FIELD_CHAT_ID = "chatId"
        private const val FIELD_TEXT = "text"
        private const val FIELD_THREAD_ID = "threadId"
        private const val FIELD_TYPE = "type"

        private const val TYPE_WAITING = "waiting"
        private const val TYPE_FINAL = "final"
        private const val TYPE_ERROR = "error"
    }

    private fun getStream(): RStreamReactive<String, String> =
        redissonMQReactiveClient.getStream(
            appProperties.mq.replyStreamKey,
            StringCodec.INSTANCE,
        )

    suspend fun publish(message: OutboundMessage) {
        val streamKey = appProperties.mq.replyStreamKey
        val stream = getStream()

        val type =
            when (message) {
                is OutboundMessage.Waiting -> TYPE_WAITING
                is OutboundMessage.Final -> TYPE_FINAL
                is OutboundMessage.Error -> TYPE_ERROR
            }

        kotlin
            .runCatching {
                val fields = buildStreamFields(message, type)

                stream
                    .add(
                        StreamAddArgs.entries(fields),
                    ).awaitSingleOrNull()

                log.sampled(key = "valkey.mq.reply.published", limit = 10, windowMillis = 1_000) {
                    it.info(
                        "VALKEY_MQ_REPLY_PUBLISHED streamKey={}, chatId={}, threadId={}, type={}",
                        streamKey,
                        message.chatId,
                        message.threadId ?: "null",
                        type,
                    )
                }
            }.onFailure { ex ->
                log.sampled(key = "valkey.mq.reply.error", limit = 10, windowMillis = 5_000) {
                    it.error(
                        "VALKEY_MQ_REPLY_ERROR streamKey={}, chatId={}, threadId={}, type={}, error={}",
                        streamKey,
                        message.chatId,
                        message.threadId ?: "null",
                        type,
                        ex.message,
                        ex,
                    )
                }
                throw ex
            }
    }

    private fun buildStreamFields(
        message: OutboundMessage,
        type: String,
    ): Map<String, String> =
        mapOf(
            FIELD_CHAT_ID to message.chatId,
            FIELD_TEXT to message.text,
            FIELD_THREAD_ID to (message.threadId ?: ""),
            FIELD_TYPE to type,
        )
}
