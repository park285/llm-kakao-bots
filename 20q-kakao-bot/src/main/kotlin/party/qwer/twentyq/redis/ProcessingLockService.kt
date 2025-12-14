package party.qwer.twentyq.redis

import jakarta.annotation.PreDestroy
import kotlinx.coroutines.CoroutineScope
import kotlinx.coroutines.Dispatchers
import kotlinx.coroutines.SupervisorJob
import kotlinx.coroutines.TimeoutCancellationException
import kotlinx.coroutines.cancel
import kotlinx.coroutines.launch
import kotlinx.coroutines.reactor.awaitSingleOrNull
import kotlinx.coroutines.withContext
import kotlinx.coroutines.withTimeout
import org.redisson.api.RedissonReactiveClient
import org.slf4j.LoggerFactory
import org.springframework.boot.context.event.ApplicationReadyEvent
import org.springframework.context.event.EventListener
import org.springframework.stereotype.Component
import party.qwer.twentyq.logging.LoggingExtensions.sampled
import party.qwer.twentyq.util.common.extensions.seconds
import party.qwer.twentyq.util.logging.LoggingConstants
import party.qwer.twentyq.redis.setAsync as setAwait

/**
 * 메시지 처리 락 관리 서비스
 */
@Component
class ProcessingLockService(
    private val redisson: RedissonReactiveClient,
) {
    companion object {
        private val log = LoggerFactory.getLogger(ProcessingLockService::class.java)

        private const val CLEANUP_PARALLELISM = 16
    }

    private val cleanupExceptionHandler =
        kotlinx.coroutines.CoroutineExceptionHandler { _, ex ->
            log.error("CLEANUP_BACKGROUND_TASK_FAILED error={}", ex.message, ex)
        }
    private val cleanupScope =
        CoroutineScope(
            SupervisorJob() + Dispatchers.IO.limitedParallelism(CLEANUP_PARALLELISM) + cleanupExceptionHandler,
        )

    suspend fun startProcessing(chatId: String) {
        val key = processingKey(chatId)
        val bucket = redisson.getBucket<String>(key)
        bucket.setAwait("1", RedisConstants.PROCESSING_TTL_SECONDS.seconds)
        log.sampled(
            key = "redis.processing.started",
            limit = LoggingConstants.LOG_SAMPLE_LIMIT_HIGH,
            windowMillis = LoggingConstants.LOG_SAMPLE_WINDOW_MS,
        ) {
            it.debug("PROCESSING_STARTED chatId={}", chatId)
        }
    }

    suspend fun finishProcessing(chatId: String) {
        val key = processingKey(chatId)
        val bucket = redisson.getBucket<String>(key)
        bucket.delete().awaitSingleOrNull()
        log.sampled(
            key = "redis.processing.finished",
            limit = LoggingConstants.LOG_SAMPLE_LIMIT_HIGH,
            windowMillis = LoggingConstants.LOG_SAMPLE_WINDOW_MS,
        ) {
            it.debug("PROCESSING_FINISHED chatId={}", chatId)
        }
    }

    suspend fun isProcessing(chatId: String): Boolean {
        val key = processingKey(chatId)
        val bucket = redisson.getBucket<String>(key)
        return bucket.isExists.awaitSingleOrNull() ?: false
    }

    @EventListener(ApplicationReadyEvent::class)
    fun cleanupStaleLocksOnStartup() {
        cleanupScope.launch {
            cleanupStaleLocksInternal()
        }
    }

    suspend fun cleanupStaleLocks(): Long = cleanupStaleLocksInternal()

    private suspend fun cleanupStaleLocksInternal(): Long {
        val cleanup =
            withContext(cleanupScope.coroutineContext) {
                runCatching {
                    withTimeout(RedisConstants.CLEANUP_TIMEOUT_MS) {
                        val lockPattern = "${RedisKeys.LOCK}:*"
                        val processingPattern = "${RedisKeys.LOCK}:processing:*"
                        val lockDeleted =
                            redisson.keys.deleteByPattern(lockPattern).awaitSingleOrNull()
                                ?: 0L
                        val processingDeleted =
                            redisson.keys.deleteByPattern(processingPattern).awaitSingleOrNull()
                                ?: 0L
                        lockDeleted + processingDeleted
                    }
                }
            }

        cleanup
            .onSuccess { totalDeleted ->
                if (totalDeleted > 0) {
                    log.warn("CLEANUP_STALE_LOCKS deleted {} stale keys", totalDeleted)
                } else {
                    log.info("CLEANUP_STALE_LOCKS no stale locks found")
                }
            }.onFailure { ex ->
                when (ex) {
                    is TimeoutCancellationException -> log.warn("CLEANUP_STALE_LOCKS timed out after 10 seconds")
                    is IllegalStateException -> log.error("CLEANUP_STALE_LOCKS state error: {}", ex.message, ex)
                    is NoSuchElementException -> log.error("CLEANUP_STALE_LOCKS element error: {}", ex.message, ex)
                    else -> log.error("CLEANUP_STALE_LOCKS unexpected error: {}", ex.message, ex)
                }
            }

        return cleanup.getOrDefault(0L)
    }

    @PreDestroy
    fun shutdown() {
        cleanupScope.cancel("shutdown")
    }

    private fun processingKey(chatId: String): String = "${RedisKeys.LOCK}:processing:$chatId"
}
