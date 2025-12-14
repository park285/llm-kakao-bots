package party.qwer.twentyq.redis.tracking

import org.redisson.api.RedissonReactiveClient
import org.slf4j.LoggerFactory
import org.springframework.boot.autoconfigure.condition.ConditionalOnProperty
import org.springframework.stereotype.Component
import party.qwer.twentyq.config.AppProperties
import party.qwer.twentyq.logging.debugL
import party.qwer.twentyq.redis.RedisKeys
import party.qwer.twentyq.redis.awaitSingleOrNull
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
class CandidateCountStore(
    private val redisson: RedissonReactiveClient,
    private val props: AppProperties,
) {
    companion object {
        private val log = LoggerFactory.getLogger(CandidateCountStore::class.java)
    }

    suspend fun getAsync(roomId: String): Int {
        val atomic = redisson.getAtomicLong(key(roomId))
        val count = atomic.get().awaitSingleOrNull()?.toInt() ?: 0
        log.debugL { "VALKEY CANDIDATE_GET room=$roomId count=$count" }
        return count
    }

    suspend fun saveAsync(
        roomId: String,
        count: Int,
    ) {
        val ttl = props.cache.sessionTtlMinutes.minutes
        val atomic = redisson.getAtomicLong(key(roomId))
        atomic.set(count.toLong())
        atomic.expireAsync(ttl)
        log.debugL { "VALKEY CANDIDATE_SAVE room=$roomId count=$count" }
    }

    suspend fun deleteAsync(roomId: String) {
        redisson.getAtomicLong(key(roomId)).delete().awaitSingleOrNull()
    }

    suspend fun setTtl(
        roomId: String,
        ttl: Duration,
    ) {
        redisson.getAtomicLong(key(roomId)).expireAsync(ttl)
    }

    private fun key(roomId: String) = "${RedisKeys.CANDIDATE_COUNT}:$roomId"
}
