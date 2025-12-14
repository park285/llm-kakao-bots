package party.qwer.twentyq.redis

import kotlinx.coroutines.CoroutineScope
import kotlinx.coroutines.Job
import kotlinx.coroutines.coroutineScope
import kotlinx.coroutines.delay
import kotlinx.coroutines.isActive
import kotlinx.coroutines.launch
import kotlinx.coroutines.reactor.awaitSingleOrNull
import kotlinx.coroutines.withTimeoutOrNull
import org.redisson.api.RedissonReactiveClient
import org.slf4j.LoggerFactory
import org.springframework.stereotype.Component
import party.qwer.twentyq.logging.LoggingExtensions.sampled
import party.qwer.twentyq.util.logging.LoggingConstants
import java.util.UUID

/**
 * Redis Lua 기반 분산 락 조정자
 */
@Component
class LockCoordinator(
    private val redisson: RedissonReactiveClient,
) {
    companion object {
        private val log = LoggerFactory.getLogger(LockCoordinator::class.java)
        private val STRING_CODEC = org.redisson.client.codec.StringCodec.INSTANCE
        private const val TOKEN_PREVIEW_LENGTH = 8
        private const val MIN_RENEW_INTERVAL_MS = 1_000L
        private const val SECONDS_TO_MILLIS = 1_000L
        private const val RENEW_INTERVAL_DIVISOR = 3
        private val LOCK_TTL_MILLIS = RedisConstants.LOCK_TTL_SECONDS * SECONDS_TO_MILLIS
        private val BLOCK_TIMEOUT_MS = RedisConstants.LOCK_BLOCK_TIMEOUT_SECONDS * SECONDS_TO_MILLIS
        private val DEFAULT_RENEW_INTERVAL_MS =
            (LOCK_TTL_MILLIS / RENEW_INTERVAL_DIVISOR).coerceAtLeast(MIN_RENEW_INTERVAL_MS)

        private val ACQUIRE_READ_SCRIPT = LuaScripts.load("lock_acquire_read.lua")
        private val ACQUIRE_WRITE_SCRIPT = LuaScripts.load("lock_acquire_write.lua")
        private val RELEASE_READ_SCRIPT = LuaScripts.load("lock_release_read.lua")
        private val RELEASE_SCRIPT = LuaScripts.load("lock_release.lua")
        private val RENEW_WRITE_SCRIPT = LuaScripts.load("lock_renew_write.lua")
        private val RENEW_READ_SCRIPT = LuaScripts.load("lock_renew_read.lua")
    }

    /**
     * Read/Write Lock을 획득하고 block 실행, 자동 해제
     *
     * @return block 결과, 획득 실패 시 null
     */
    suspend fun <T> withLock(
        chatId: String,
        userId: String,
        requiresWrite: Boolean,
        blockTimeoutMillis: Long = BLOCK_TIMEOUT_MS,
        block: suspend () -> T,
    ): T? {
        val token = if (requiresWrite) acquireWrite(chatId, userId) else acquireRead(chatId, userId)
        if (token == null) {
            return null
        }

        val blockResult =
            coroutineScope {
                val watchdog = launchRenewWatchdog(chatId, userId, requiresWrite, token)

                try {
                    runBlockWithTimeout(blockTimeoutMillis, block)
                } finally {
                    watchdog.cancel()
                    if (requiresWrite) {
                        releaseWrite(chatId, userId, token)
                    } else {
                        releaseRead(chatId, userId, token)
                    }
                }
            }

        if (blockResult == null) {
            val mode = if (requiresWrite) "WRITE" else "READ"
            log.warn(
                "LOCK_BLOCK_TIMEOUT chatId={}, userId={}, mode={}, timeoutMs={}",
                chatId,
                userId,
                mode,
                blockTimeoutMillis,
            )
        }

        return blockResult
    }

    private fun CoroutineScope.launchRenewWatchdog(
        chatId: String,
        userId: String,
        requiresWrite: Boolean,
        token: String,
    ): Job =
        launch {
            while (isActive) {
                delay(DEFAULT_RENEW_INTERVAL_MS)
                val renewed = if (requiresWrite) renewWrite(chatId, token) else renewRead(chatId, token)
                if (!renewed) {
                    val mode = if (requiresWrite) "WRITE" else "READ"
                    log.warn("LOCK_RENEW_FAILED chatId={}, userId={}, mode={}", chatId, userId, mode)
                    break
                }
            }
        }

    private suspend fun <T> runBlockWithTimeout(
        blockTimeoutMillis: Long,
        block: suspend () -> T,
    ): T? = withTimeoutOrNull(blockTimeoutMillis) { block() }

    private suspend fun acquireWrite(
        chatId: String,
        userId: String,
    ): String? {
        val token = UUID.randomUUID().toString()
        val writeLockKey = lockKey(chatId)
        val readHashKey = readLockKey(chatId)
        val ttlMillis = LOCK_TTL_MILLIS

        val success =
            redisson
                .getScript(STRING_CODEC)
                .eval<Long>(
                    org.redisson.api.RScript.Mode.READ_WRITE,
                    ACQUIRE_WRITE_SCRIPT,
                    org.redisson.api.RScript.ReturnType.INTEGER,
                    listOf(writeLockKey, readHashKey),
                    token,
                    ttlMillis.toString(),
                ).awaitSingleOrNull() ?: 0L

        return if (success == 1L) {
            log.sampled(
                key = "redis.lock.acquired.write",
                limit = LoggingConstants.LOG_SAMPLE_LIMIT_HIGH,
                windowMillis = LoggingConstants.LOG_SAMPLE_WINDOW_MS,
            ) {
                val preview = token.take(TOKEN_PREVIEW_LENGTH)
                it.debug("LOCK_ACQUIRED_WRITE chatId={}, userId={}, token={}", chatId, userId, preview)
            }
            token
        } else {
            log.sampled(
                key = "redis.lock.contention.write",
                limit = LoggingConstants.LOG_SAMPLE_LIMIT_HIGH,
                windowMillis = LoggingConstants.LOG_SAMPLE_WINDOW_MS,
            ) {
                it.warn("LOCK_CONTENTION_WRITE chatId={}, userId={}", chatId, userId)
            }
            null
        }
    }

    private suspend fun acquireRead(
        chatId: String,
        userId: String,
    ): String? {
        val token = UUID.randomUUID().toString()
        val writeLockKey = lockKey(chatId)
        val readHashKey = readLockKey(chatId)
        val ttlMillis = LOCK_TTL_MILLIS

        val success =
            redisson
                .getScript(STRING_CODEC)
                .eval<Long>(
                    org.redisson.api.RScript.Mode.READ_WRITE,
                    ACQUIRE_READ_SCRIPT,
                    org.redisson.api.RScript.ReturnType.INTEGER,
                    listOf(writeLockKey, readHashKey),
                    token,
                    ttlMillis.toString(),
                ).awaitSingleOrNull() ?: 0L

        return if (success == 1L) {
            log.sampled(
                key = "redis.lock.acquired.read",
                limit = LoggingConstants.LOG_SAMPLE_LIMIT_HIGH,
                windowMillis = LoggingConstants.LOG_SAMPLE_WINDOW_MS,
            ) {
                val preview = token.take(TOKEN_PREVIEW_LENGTH)
                it.debug("LOCK_ACQUIRED_READ chatId={}, userId={}, token={}", chatId, userId, preview)
            }
            token
        } else {
            log.sampled(
                key = "redis.lock.contention.read",
                limit = LoggingConstants.LOG_SAMPLE_LIMIT_HIGH,
                windowMillis = LoggingConstants.LOG_SAMPLE_WINDOW_MS,
            ) {
                it.warn("LOCK_CONTENTION_READ chatId={}, userId={}", chatId, userId)
            }
            null
        }
    }

    private suspend fun renewWrite(
        chatId: String,
        token: String,
    ): Boolean =
        redisson
            .getScript(STRING_CODEC)
            .eval<Long>(
                org.redisson.api.RScript.Mode.READ_WRITE,
                RENEW_WRITE_SCRIPT,
                org.redisson.api.RScript.ReturnType.INTEGER,
                listOf(lockKey(chatId)),
                token,
                LOCK_TTL_MILLIS.toString(),
            ).awaitSingleOrNull() == 1L

    private suspend fun renewRead(
        chatId: String,
        token: String,
    ): Boolean =
        redisson
            .getScript(STRING_CODEC)
            .eval<Long>(
                org.redisson.api.RScript.Mode.READ_WRITE,
                RENEW_READ_SCRIPT,
                org.redisson.api.RScript.ReturnType.INTEGER,
                listOf(readLockKey(chatId)),
                token,
                LOCK_TTL_MILLIS.toString(),
            ).awaitSingleOrNull() == 1L

    private suspend fun releaseWrite(
        chatId: String,
        userId: String,
        token: String,
    ) {
        val deleted =
            redisson
                .getScript(STRING_CODEC)
                .eval<Long>(
                    org.redisson.api.RScript.Mode.READ_WRITE,
                    RELEASE_SCRIPT,
                    org.redisson.api.RScript.ReturnType.INTEGER,
                    listOf(lockKey(chatId)),
                    token,
                ).awaitSingleOrNull() ?: 0L

        if (deleted == 1L) {
            log.sampled(
                key = "redis.lock.released.write",
                limit = LoggingConstants.LOG_SAMPLE_LIMIT_HIGH,
                windowMillis = LoggingConstants.LOG_SAMPLE_WINDOW_MS,
            ) {
                val preview = token.take(TOKEN_PREVIEW_LENGTH)
                it.debug("LOCK_RELEASED_WRITE chatId={}, userId={}, token={}", chatId, userId, preview)
            }
        } else {
            log.sampled(
                key = "redis.lock.release_failed.write",
                limit = LoggingConstants.LOG_SAMPLE_LIMIT_HIGH,
                windowMillis = LoggingConstants.LOG_SAMPLE_WINDOW_LONG,
            ) { logger ->
                val preview = token.take(TOKEN_PREVIEW_LENGTH)
                logger.warn(
                    "LOCK_RELEASE_FAILED_WRITE chatId={}, userId={}, token={}",
                    chatId,
                    userId,
                    preview,
                )
            }
        }
    }

    private suspend fun releaseRead(
        chatId: String,
        userId: String,
        token: String,
    ) {
        val deleted =
            redisson
                .getScript(STRING_CODEC)
                .eval<Long>(
                    org.redisson.api.RScript.Mode.READ_WRITE,
                    RELEASE_READ_SCRIPT,
                    org.redisson.api.RScript.ReturnType.INTEGER,
                    listOf(readLockKey(chatId)),
                    token,
                    LOCK_TTL_MILLIS.toString(),
                ).awaitSingleOrNull() ?: 0L

        if (deleted == 1L) {
            log.sampled(
                key = "redis.lock.released.read",
                limit = LoggingConstants.LOG_SAMPLE_LIMIT_HIGH,
                windowMillis = LoggingConstants.LOG_SAMPLE_WINDOW_MS,
            ) {
                val preview = token.take(TOKEN_PREVIEW_LENGTH)
                it.debug("LOCK_RELEASED_READ chatId={}, userId={}, token={}", chatId, userId, preview)
            }
        } else {
            log.sampled(
                key = "redis.lock.release_failed.read",
                limit = LoggingConstants.LOG_SAMPLE_LIMIT_HIGH,
                windowMillis = LoggingConstants.LOG_SAMPLE_WINDOW_LONG,
            ) { logger ->
                val preview = token.take(TOKEN_PREVIEW_LENGTH)
                logger.warn(
                    "LOCK_RELEASE_FAILED_READ chatId={}, userId={}, token={}",
                    chatId,
                    userId,
                    preview,
                )
            }
        }
    }

    private fun lockKey(
        chatId: String,
        suffix: String? = null,
    ): String =
        if (suffix == null) {
            "${RedisKeys.LOCK}:$chatId"
        } else {
            "${RedisKeys.LOCK}:$chatId:$suffix"
        }

    private fun readLockKey(chatId: String): String = lockKey(chatId, "read")
}
