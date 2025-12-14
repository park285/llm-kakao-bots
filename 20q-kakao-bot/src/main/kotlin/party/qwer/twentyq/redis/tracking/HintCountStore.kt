package party.qwer.twentyq.redis.tracking

import kotlinx.coroutines.reactor.awaitSingleOrNull
import org.redisson.api.RedissonReactiveClient
import org.slf4j.LoggerFactory
import org.springframework.boot.autoconfigure.condition.ConditionalOnProperty
import org.springframework.stereotype.Component
import party.qwer.twentyq.config.AppProperties
import party.qwer.twentyq.logging.debugL
import party.qwer.twentyq.redis.RedisKeys
import party.qwer.twentyq.redis.expireAsync
import party.qwer.twentyq.util.common.extensions.minutes
import java.time.Duration

@Component
@ConditionalOnProperty(
    prefix = "app.cache",
    name = ["enabled"],
    havingValue = "true",
    matchIfMissing = true,
)
class HintCountStore(
    private val redisson: RedissonReactiveClient,
    private val props: AppProperties,
) {
    companion object {
        private val log = LoggerFactory.getLogger(HintCountStore::class.java)
    }

    suspend fun getAsync(roomId: String): Int {
        val atomic = redisson.getAtomicLong(key(roomId))
        val count = atomic.get().awaitSingleOrNull()?.toInt() ?: 0
        log.debugL { "VALKEY HINT_GET room=$roomId count=$count" }
        return count
    }

    suspend fun increment(roomId: String): Long {
        val atomic = redisson.getAtomicLong(key(roomId))
        val result = atomic.incrementAndGet().awaitSingleOrNull() ?: 0L
        atomic.expireAsync(
            props.cache.sessionTtlMinutes.minutes,
        )
        log.debugL { "VALKEY HINT_INCREMENT room=$roomId newCount=$result" }
        return result
    }

    suspend fun deleteAsync(roomId: String) {
        redisson.getBucket<String>(key(roomId)).delete().awaitSingleOrNull()
    }

    suspend fun setTtl(
        roomId: String,
        ttl: Duration,
    ) {
        redisson.getBucket<String>(key(roomId)).expireAsync(ttl)
    }

    private fun key(roomId: String) = "${RedisKeys.HINTS}:$roomId"
}
