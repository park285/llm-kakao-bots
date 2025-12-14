package io.github.kapu.turtlesoup.mq

import io.github.kapu.turtlesoup.config.MQConstants
import io.github.kapu.turtlesoup.config.StreamKeys
import io.github.kapu.turtlesoup.mq.models.OutboundMessage
import io.github.oshai.kotlinlogging.KotlinLogging
import kotlinx.coroutines.reactor.awaitSingleOrNull
import org.redisson.api.RedissonReactiveClient
import org.redisson.api.stream.StreamAddArgs
import org.redisson.client.codec.StringCodec

/** Valkey Stream으로 응답 발행 */
class ValkeyMQReplyPublisher(
    private val redissonClient: RedissonReactiveClient,
    private val replyStreamKey: String,
) {
    /**
     * 응답 메시지 발행
     */
    suspend fun publish(message: OutboundMessage) {
        try {
            // Iris (Jedis)와 호환성을 위해 StringCodec 사용
            val stream = redissonClient.getStream<String, String>(replyStreamKey, StringCodec.INSTANCE)

            // StreamAddArgs.entries() 사용 (20q-kakao-bot 방식)
            val fields =
                buildMap<String, String> {
                    put(StreamKeys.FIELD_CHAT_ID, message.chatId)
                    put(StreamKeys.FIELD_TEXT, message.text)
                    put(StreamKeys.FIELD_TYPE, getMessageType(message))
                    message.threadId?.let { put(StreamKeys.FIELD_THREAD_ID, it) }
                }

            val addArgs =
                StreamAddArgs.entries(fields)
                    .trimNonStrict() // 대략적 트림 (~)
                    .maxLen(MQConstants.STREAM_MAX_LEN.toInt())
                    .noLimit()

            // Reactive를 코루틴으로 변환
            val messageId =
                stream.add(addArgs).awaitSingleOrNull()

            logger.debug {
                "message_published " +
                    "stream=$replyStreamKey " +
                    "message_id=$messageId " +
                    "chat_id=${message.chatId} " +
                    "type=${getMessageType(message)}"
            }
        } catch (error: org.redisson.client.RedisException) {
            logger.error(error) {
                "message_publish_failed " +
                    "stream=$replyStreamKey " +
                    "chat_id=${message.chatId}"
            }
            throw error
        }
    }

    /**
     * OutboundMessage 타입을 문자열로 변환
     */
    private fun getMessageType(message: OutboundMessage): String =
        when (message) {
            is OutboundMessage.Waiting -> StreamKeys.TYPE_WAITING
            is OutboundMessage.Final -> StreamKeys.TYPE_FINAL
            is OutboundMessage.Error -> StreamKeys.TYPE_ERROR
        }

    companion object {
        private val logger = KotlinLogging.logger {}
    }
}
