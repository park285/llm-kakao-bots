package io.github.kapu.turtlesoup.redis

import io.github.kapu.turtlesoup.config.JsonConfig
import io.github.kapu.turtlesoup.config.RedisConstants
import io.github.kapu.turtlesoup.config.RedisKeys
import io.github.kapu.turtlesoup.mq.models.DequeueResult
import io.github.kapu.turtlesoup.mq.models.EnqueueResult
import io.github.kapu.turtlesoup.mq.models.PendingMessage
import io.github.oshai.kotlinlogging.KotlinLogging
import kotlinx.coroutines.coroutineScope
import kotlinx.coroutines.launch
import kotlinx.coroutines.reactor.awaitSingleOrNull
import kotlinx.serialization.encodeToString
import org.redisson.api.RScript
import org.redisson.api.RedissonReactiveClient
import org.redisson.client.codec.StringCodec

/** 대기 중인 메시지 저장소 (Lua 스크립트 기반 원자적 연산) */
class PendingMessageStore(
    private val redisson: RedissonReactiveClient,
) {
    companion object {
        private val logger = KotlinLogging.logger {}
        private val json = JsonConfig.lenient

        private val ENQUEUE_SCRIPT = LuaScripts.load("pending_enqueue.lua")
        private val DEQUEUE_SCRIPT = LuaScripts.load("pending_dequeue.lua")
        private const val MAX_DEQUEUE_ITERATIONS = 50
        private const val TIMESTAMP_DELIMITER = "|"
    }

    /** 메시지 큐잉 (원자적) - timestamp|JSON 포맷으로 저장 */
    suspend fun enqueue(
        chatId: String,
        message: PendingMessage,
    ): EnqueueResult {
        val queueKey = queueKey(chatId)
        val userSetKey = userSetKey(chatId)
        val jsonValue = json.encodeToString(message)
        val timestampedValue = "${message.timestamp}$TIMESTAMP_DELIMITER$jsonValue"

        val result =
            redisson.getScript(StringCodec.INSTANCE)
                .eval<String>(
                    RScript.Mode.READ_WRITE,
                    ENQUEUE_SCRIPT,
                    RScript.ReturnType.VALUE,
                    listOf(queueKey, userSetKey),
                    message.userId,
                    timestampedValue,
                    RedisConstants.MAX_QUEUE_SIZE.toString(),
                    RedisConstants.QUEUE_TTL_SECONDS.toString(),
                ).awaitSingleOrNull()

        return when (result) {
            "SUCCESS" -> {
                logger.debug { "enqueue_success chat_id=$chatId user_id=${message.userId}" }
                EnqueueResult.SUCCESS
            }
            "DUPLICATE" -> {
                logger.debug { "enqueue_duplicate chat_id=$chatId user_id=${message.userId}" }
                EnqueueResult.DUPLICATE
            }
            "QUEUE_FULL" -> {
                logger.warn { "enqueue_queue_full chat_id=$chatId" }
                EnqueueResult.QUEUE_FULL
            }
            else -> {
                logger.error { "enqueue_unknown_result chat_id=$chatId result=$result" }
                EnqueueResult.QUEUE_FULL
            }
        }
    }

    /** 메시지 디큐 (원자적) */
    suspend fun dequeue(chatId: String): DequeueResult {
        val queueKey = queueKey(chatId)
        val userSetKey = userSetKey(chatId)
        val currentTimestamp = System.currentTimeMillis()

        val result =
            redisson.getScript(StringCodec.INSTANCE)
                .eval<String>(
                    RScript.Mode.READ_WRITE,
                    DEQUEUE_SCRIPT,
                    RScript.ReturnType.VALUE,
                    listOf(queueKey, userSetKey),
                    currentTimestamp.toString(),
                    RedisConstants.STALE_THRESHOLD_MS.toString(),
                    MAX_DEQUEUE_ITERATIONS.toString(),
                ).awaitSingleOrNull()

        return when (result) {
            null -> {
                logger.debug { "dequeue_empty chat_id=$chatId" }
                DequeueResult.Empty
            }
            "EXHAUSTED" -> {
                logger.debug { "dequeue_exhausted chat_id=$chatId" }
                DequeueResult.Exhausted
            }
            else -> {
                val message = json.decodeFromString<PendingMessage>(result)
                logger.debug { "dequeue_success chat_id=$chatId user_id=${message.userId}" }
                DequeueResult.Success(message)
            }
        }
    }

    /** 큐 크기 */
    suspend fun size(chatId: String): Int {
        val queueKey = queueKey(chatId)
        val queue = redisson.getDeque<String>(queueKey)
        return queue.size().awaitSingleOrNull() ?: 0
    }

    /** 대기 중인 메시지 존재 여부 */
    suspend fun hasPending(chatId: String): Boolean = size(chatId) > 0

    /** 큐 상세 정보 (대기 순서 표시용) */
    suspend fun getQueueDetails(chatId: String): String {
        val queueKey = queueKey(chatId)
        val queue = redisson.getDeque<String>(queueKey)
        val entries = queue.readAll().awaitSingleOrNull().orEmpty()
        if (entries.isEmpty()) return ""

        return entries.mapIndexed { idx, entry ->
            val jsonValue = extractJson(entry)
            val msg = json.decodeFromString<PendingMessage>(jsonValue)
            val displayName = msg.sender ?: msg.userId
            "${idx + 1}. $displayName - ${msg.content}"
        }.joinToString("\n")
    }

    /** 큐 초기화 */
    suspend fun clear(chatId: String) {
        val queueKey = queueKey(chatId)
        val userSetKey = userSetKey(chatId)

        coroutineScope {
            launch { redisson.getDeque<String>(queueKey).delete().awaitSingleOrNull() }
            launch { redisson.getSet<String>(userSetKey).delete().awaitSingleOrNull() }
        }

        logger.debug { "queue_cleared chat_id=$chatId" }
    }

    /** timestamp|JSON 포맷에서 JSON 부분 추출 */
    private fun extractJson(entry: String): String {
        val delimiterIndex = entry.indexOf(TIMESTAMP_DELIMITER)
        require(delimiterIndex > 0) { "Invalid format: missing timestamp delimiter" }
        return entry.substring(delimiterIndex + 1)
    }

    private fun queueKey(chatId: String): String = "${RedisKeys.PENDING_MESSAGES}:{$chatId}"

    private fun userSetKey(chatId: String): String = "${RedisKeys.PENDING_MESSAGES}:{$chatId}:users"
}
