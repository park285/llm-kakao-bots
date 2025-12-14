package io.github.kapu.turtlesoup.redis

import io.github.kapu.turtlesoup.config.RedisConstants
import io.github.kapu.turtlesoup.config.RedisKeys
import io.github.oshai.kotlinlogging.KotlinLogging
import kotlinx.coroutines.reactor.awaitSingleOrNull
import org.redisson.api.RedissonReactiveClient
import java.time.Duration

/** 메시지 처리 중 상태 관리 서비스 */
class ProcessingLockService(
    private val redisson: RedissonReactiveClient,
) {
    companion object {
        private val logger = KotlinLogging.logger {}
    }

    /** 처리 시작 마킹 */
    suspend fun startProcessing(chatId: String) {
        val key = processingKey(chatId)
        val bucket = redisson.getBucket<String>(key)
        bucket.set("1", Duration.ofSeconds(RedisConstants.PROCESSING_TTL_SECONDS)).awaitSingleOrNull()
        logger.debug { "processing_started chat_id=$chatId" }
    }

    /** 처리 완료 마킹 */
    suspend fun finishProcessing(chatId: String) {
        val key = processingKey(chatId)
        val bucket = redisson.getBucket<String>(key)
        bucket.delete().awaitSingleOrNull()
        logger.debug { "processing_finished chat_id=$chatId" }
    }

    /** 처리 중 여부 확인 */
    suspend fun isProcessing(chatId: String): Boolean {
        val key = processingKey(chatId)
        val bucket = redisson.getBucket<String>(key)
        return bucket.isExists.awaitSingleOrNull() ?: false
    }

    private fun processingKey(chatId: String): String = "${RedisKeys.PROCESSING}:$chatId"
}
