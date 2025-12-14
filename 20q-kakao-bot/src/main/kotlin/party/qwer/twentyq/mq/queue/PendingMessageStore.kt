package party.qwer.twentyq.mq.queue

import kotlinx.coroutines.coroutineScope
import kotlinx.coroutines.launch
import kotlinx.coroutines.reactor.awaitSingleOrNull
import org.redisson.api.RScript
import org.redisson.api.RedissonReactiveClient
import org.slf4j.LoggerFactory
import org.springframework.stereotype.Component
import party.qwer.twentyq.logging.LoggingExtensions.sampled
import party.qwer.twentyq.logging.debugL
import party.qwer.twentyq.model.PendingMessage
import party.qwer.twentyq.redis.LuaScripts
import party.qwer.twentyq.redis.RedisKeys
import party.qwer.twentyq.util.common.extensions.nowMillis
import party.qwer.twentyq.util.common.extensions.seconds
import tools.jackson.databind.ObjectMapper
import java.time.Duration

/**
 * 대기 중인 메시지 저장소
 */
@Component
class PendingMessageStore(
    private val redisson: RedissonReactiveClient,
    private val objectMapper: ObjectMapper,
) {
    companion object {
        private val log = LoggerFactory.getLogger(PendingMessageStore::class.java)
        private val STRING_CODEC = org.redisson.client.codec.StringCodec.INSTANCE
        private const val QUEUE_TTL_SECONDS = 300L
        private const val MAX_QUEUE_SIZE = 5
        private const val STALE_THRESHOLD_MS = 3600_000L
        private const val CHAIN_SKIP_FLAG_TTL_SECONDS = 300L
        private const val MAX_DEQUEUE_ITERATIONS = 50

        private val ENQUEUE_SCRIPT = LuaScripts.load("pending_enqueue.lua")
        private val DEQUEUE_SCRIPT = LuaScripts.load("pending_dequeue.lua")
    }

    /**
     * 체인 배치 스킵 플래그 설정 (조건 불만족 시)
     * TTL 5분(300초)으로 임시 저장
     */
    suspend fun setChainSkipFlag(
        chatId: String,
        userId: String,
    ) {
        val skipKey = chainSkipKey(chatId, userId)
        redisson
            .getBucket<String>(skipKey, STRING_CODEC)
            .set("1", CHAIN_SKIP_FLAG_TTL_SECONDS.seconds)
            .awaitSingleOrNull()
        log.debugL { "CHAIN_SKIP_FLAG_SET chatId=$chatId, userId=$userId" }
    }

    /**
     * 체인 배치 스킵 플래그 확인
     */
    suspend fun hasChainSkipFlag(
        chatId: String,
        userId: String,
    ): Boolean {
        val skipKey = chainSkipKey(chatId, userId)
        // getAndDelete: 원자적 연산으로 race condition 방지
        val value =
            redisson
                .getBucket<String>(skipKey, STRING_CODEC)
                .getAndDelete()
                .awaitSingleOrNull()
        val hasFlag = value != null
        if (hasFlag) {
            log.debugL { "CHAIN_SKIP_FLAG_FOUND chatId=$chatId, userId=$userId" }
        }
        return hasFlag
    }

    private fun chainSkipKey(
        chatId: String,
        userId: String,
    ) = "pending:chain:skip:$chatId:$userId"

    suspend fun enqueue(
        chatId: String,
        message: PendingMessage,
    ): EnqueueResult {
        val queueKey = queueKey(chatId)
        val userSetKey = userSetKey(chatId)

        val json = objectMapper.writeValueAsString(message)

        val result =
            redisson
                .getScript(STRING_CODEC)
                .eval<String>(
                    RScript.Mode.READ_WRITE,
                    ENQUEUE_SCRIPT,
                    RScript.ReturnType.VALUE,
                    listOf(queueKey, userSetKey),
                    message.userId,
                    json,
                    MAX_QUEUE_SIZE.toString(),
                    QUEUE_TTL_SECONDS.toString(),
                ).awaitSingleOrNull()

        val enumResult =
            when (result) {
                "SUCCESS" -> EnqueueResult.SUCCESS
                "DUPLICATE" -> EnqueueResult.DUPLICATE
                "QUEUE_FULL" -> EnqueueResult.QUEUE_FULL
                else -> EnqueueResult.SUCCESS
            }

        log.sampled(key = "redis.pending.enqueue", limit = 5, windowMillis = 2_000) {
            it.debug(
                "MESSAGE_ENQUEUED chatId={}, userId={}, result={}",
                chatId,
                message.userId,
                enumResult,
            )
        }

        return enumResult
    }

    suspend fun dequeue(chatId: String): PendingMessage? {
        val queueKey = queueKey(chatId)
        val userSetKey = userSetKey(chatId)

        // Lua Script로 원자적 dequeue (LPOP + Stale 체크 + SREM)
        // maxIterations로 Redis 블로킹 방지
        val json =
            redisson
                .getScript(STRING_CODEC)
                .eval<String>(
                    RScript.Mode.READ_WRITE,
                    DEQUEUE_SCRIPT,
                    RScript.ReturnType.VALUE,
                    listOf(queueKey, userSetKey),
                    nowMillis().toString(),
                    STALE_THRESHOLD_MS.toString(),
                    MAX_DEQUEUE_ITERATIONS.toString(),
                ).awaitSingleOrNull() ?: return null

        val message = objectMapper.readValue(json, PendingMessage::class.java)

        log.sampled(key = "redis.pending.dequeue", limit = 5, windowMillis = 2_000) {
            it.debug("MESSAGE_DEQUEUED chatId={}, userId={}", chatId, message.userId)
        }

        return message
    }

    suspend fun size(chatId: String): Int {
        val queueKey = queueKey(chatId)
        return redisson.getQueue<String>(queueKey, STRING_CODEC).size().awaitSingleOrNull() ?: 0
    }

    suspend fun hasPending(chatId: String): Boolean = size(chatId) > 0

    suspend fun getQueueDetails(chatId: String): String {
        val queueKey = queueKey(chatId)
        val entries = redisson.getQueue<String>(queueKey, STRING_CODEC).readAll().awaitSingleOrNull() ?: emptyList()

        if (entries.isEmpty()) return ""

        val messages =
            entries.mapIndexed { idx, json ->
                val msg = objectMapper.readValue(json, PendingMessage::class.java)
                val displayName = msg.sender ?: msg.userId

                // 체인 메시지 처리: batchQuestions 사용
                val content =
                    when {
                        msg.isChainBatch && !msg.batchQuestions.isNullOrEmpty() -> {
                            msg.batchQuestions.joinToString(", ")
                        }
                        else -> msg.content
                    }

                "${idx + 1}. $displayName - $content"
            }
        return messages.joinToString("\n")
    }

    suspend fun clear(chatId: String) {
        val queueKey = queueKey(chatId)
        val userSetKey = userSetKey(chatId)

        val size = size(chatId)

        coroutineScope {
            launch { redisson.getQueue<String>(queueKey, STRING_CODEC).delete().awaitSingleOrNull() }
            launch { redisson.getSet<String>(userSetKey, STRING_CODEC).delete().awaitSingleOrNull() }
        }

        log.sampled(key = "redis.pending.clear", limit = 5, windowMillis = 5_000) {
            it.debug("QUEUE_CLEARED chatId={}, clearedCount={}", chatId, size)
        }
    }

    private fun queueKey(chatId: String): String = "${RedisKeys.PENDING_MESSAGES}:$chatId"

    private fun userSetKey(chatId: String): String = "${RedisKeys.PENDING_MESSAGES}:users:$chatId"
}
