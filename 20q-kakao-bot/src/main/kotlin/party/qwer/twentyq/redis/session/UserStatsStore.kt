package party.qwer.twentyq.redis.session

import org.redisson.api.RedissonReactiveClient
import org.slf4j.Logger
import org.slf4j.LoggerFactory
import org.springframework.beans.factory.annotation.Qualifier
import org.springframework.boot.autoconfigure.condition.ConditionalOnProperty
import org.springframework.stereotype.Component
import party.qwer.twentyq.logging.debugL
import party.qwer.twentyq.model.UserStats
import party.qwer.twentyq.redis.BaseJsonBucketStore
import party.qwer.twentyq.redis.RedisKeys
import party.qwer.twentyq.util.common.extensions.minutes
import tools.jackson.databind.ObjectMapper
import java.time.Duration

/**
 * 사용자 통계 캐시 저장소 (읽기 성능 최적화용)
 */
@Component
@ConditionalOnProperty(
    prefix = "app.cache",
    name = ["enabled"],
    havingValue = "true",
    matchIfMissing = true,
)
class UserStatsStore(
    redisson: RedissonReactiveClient,
    @Qualifier("kotlinJsonMapper") objectMapper: ObjectMapper,
) : BaseJsonBucketStore<UserStats>(redisson, objectMapper, UserStats::class.java) {
    companion object {
        private val log = LoggerFactory.getLogger(UserStatsStore::class.java)
        private const val CACHE_TTL_MINUTES = 1440L // 24시간
    }

    override val log: Logger = UserStatsStore.log

    /**
     * 통계 조회 (캐시)
     */
    suspend fun get(
        chatId: String,
        userId: String,
    ): UserStats? {
        val stats = read(key(chatId, userId), contextKey = "stats:$chatId:$userId")
        log.debugL { "VALKEY STATS_GET chatId=$chatId, userId=$userId exists=${stats != null}" }
        return stats
    }

    /**
     * 통계 저장 (캐시)
     */
    suspend fun set(
        chatId: String,
        userId: String,
        stats: UserStats,
    ) {
        write(key(chatId, userId), stats, cacheTtl())
        log.debugL {
            "VALKEY STATS_SET chatId=$chatId, userId=$userId ttl=${CACHE_TTL_MINUTES}min"
        }
    }

    /**
     * 통계 무효화 (게임 완료 시)
     */
    suspend fun invalidate(
        chatId: String,
        userId: String,
    ) {
        delete(key(chatId, userId))
        log.debugL { "VALKEY STATS_INVALIDATE chatId=$chatId, userId=$userId" }
    }

    /**
     * TTL 갱신
     */
    suspend fun setTtl(
        chatId: String,
        userId: String,
        ttl: Duration,
    ): Boolean {
        val result = expire(key(chatId, userId), ttl)
        log.debugL { "VALKEY STATS_TTL chatId=$chatId, userId=$userId ttl=$ttl success=$result" }
        return result
    }

    private fun cacheTtl(): Duration = CACHE_TTL_MINUTES.minutes

    private fun key(
        chatId: String,
        userId: String,
    ) = "${RedisKeys.STATS}:$chatId:$userId"
}
