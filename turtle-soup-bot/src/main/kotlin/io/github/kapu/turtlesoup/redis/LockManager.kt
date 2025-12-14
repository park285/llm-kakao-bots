package io.github.kapu.turtlesoup.redis

import io.github.kapu.turtlesoup.config.RedisConstants
import io.github.kapu.turtlesoup.config.RedisKeys
import io.github.kapu.turtlesoup.utils.LockException
import io.github.oshai.kotlinlogging.KotlinLogging
import kotlinx.coroutines.reactor.awaitSingle
import kotlinx.coroutines.reactor.awaitSingleOrNull
import org.redisson.api.RLockReactive
import org.redisson.api.RedissonReactiveClient
import java.time.Duration
import java.util.concurrent.TimeUnit
import org.redisson.client.RedisException as RedissonException

class LockManager(
    private val redisson: RedissonReactiveClient,
) {
    suspend fun tryAcquireSharedLock(
        lockKey: String,
        ttlSeconds: Long,
    ): Boolean {
        val lock = redisson.getLock(lockKey)
        val leaseSeconds = ttlSeconds.coerceAtLeast(1L)
        return lock.tryLock(0, leaseSeconds, TimeUnit.SECONDS).awaitSingle()
    }

    suspend fun releaseSharedLock(lockKey: String) {
        redisson.getLock(lockKey).forceUnlock().awaitSingleOrNull()
    }

    suspend fun <T> withLock(
        sessionId: String,
        holderName: String? = null,
        timeoutSeconds: Long = RedisConstants.LOCK_TIMEOUT_SECONDS,
        block: suspend () -> T,
    ): T {
        val lockKey = "${RedisKeys.LOCK}:$sessionId"
        val holderKey = "${RedisKeys.LOCK}:holder:$sessionId"

        logger.info { "lock_attempting session_id=$sessionId timeout=${timeoutSeconds}s" }

        val lock = redisson.getLock(lockKey)

        val acquired =
            lock.tryLock(timeoutSeconds, RedisConstants.LOCK_TTL_SECONDS, TimeUnit.SECONDS).awaitSingle()

        logger.info { "lock_tryLock_returned session_id=$sessionId acquired=$acquired" }

        if (!acquired) {
            val currentHolder = getHolder(holderKey)
            logger.warn {
                "lock_acquisition_failed session=$sessionId holder=$currentHolder timeout=${timeoutSeconds}s"
            }
            throw LockException("Failed to acquire lock for session: $sessionId", currentHolder)
        }

        // 락 획득 성공 시 소유자 정보 저장
        val effectiveHolder = holderName ?: "다른 사용자"
        setHolder(holderKey, effectiveHolder)

        return try {
            logger.info { "lock_acquired session_id=$sessionId holder=$effectiveHolder" }
            block()
        } finally {
            releaseLock(sessionId, holderKey, lock)
        }
    }

    private suspend fun releaseLock(
        sessionId: String,
        holderKey: String,
        lock: RLockReactive,
    ) {
        try {
            redisson.getBucket<String>(holderKey).delete().awaitSingleOrNull()
            lock.forceUnlock().awaitSingleOrNull()
            logger.info { "lock_released session_id=$sessionId" }
        } catch (e: RedissonException) {
            logger.warn(e) { "lock_release_failed session_id=$sessionId" }
        }
    }

    private suspend fun setHolder(
        key: String,
        name: String,
    ) {
        redisson.getBucket<String>(key)
            .set(name, Duration.ofSeconds(RedisConstants.LOCK_TTL_SECONDS))
            .awaitSingleOrNull()
    }

    private suspend fun getHolder(key: String): String? {
        return redisson.getBucket<String>(key).get().awaitSingleOrNull()
    }

    companion object {
        private val logger = KotlinLogging.logger {}
    }
}
