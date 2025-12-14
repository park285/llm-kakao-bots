package party.qwer.twentyq.redis

import kotlinx.coroutines.reactor.awaitSingleOrNull
import org.redisson.api.RedissonReactiveClient
import org.springframework.stereotype.Component
import java.util.concurrent.TimeUnit
import kotlin.math.max

interface RestartLock {
    suspend fun tryAcquire(
        lockKey: String,
        ttlSeconds: Long,
    ): Boolean

    suspend fun release(lockKey: String)
}

@Component
class RedisRestartLock(
    private val redisson: RedissonReactiveClient,
) : RestartLock {
    companion object {
        private const val LOCK_WAIT_SECONDS = 0L
        private const val MIN_TTL_SECONDS = 1L
    }

    override suspend fun tryAcquire(
        lockKey: String,
        ttlSeconds: Long,
    ): Boolean {
        val lock = redisson.getLock(lockKey)
        val leaseSeconds = max(ttlSeconds, MIN_TTL_SECONDS)
        return lock.tryLock(LOCK_WAIT_SECONDS, leaseSeconds, TimeUnit.SECONDS).awaitSingleOrNull() ?: false
    }

    override suspend fun release(lockKey: String) {
        redisson.getLock(lockKey).forceUnlock().awaitSingleOrNull()
    }
}
