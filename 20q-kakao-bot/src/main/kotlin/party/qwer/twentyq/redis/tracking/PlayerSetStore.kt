package party.qwer.twentyq.redis.tracking

import kotlinx.coroutines.reactor.awaitSingleOrNull
import org.redisson.api.RedissonReactiveClient
import org.slf4j.LoggerFactory
import org.springframework.boot.autoconfigure.condition.ConditionalOnProperty
import org.springframework.stereotype.Component
import party.qwer.twentyq.config.AppProperties
import party.qwer.twentyq.logging.debugL
import party.qwer.twentyq.model.PlayerInfo
import party.qwer.twentyq.redis.RedisKeys
import party.qwer.twentyq.util.common.extensions.minutes
import tools.jackson.core.type.TypeReference
import tools.jackson.databind.ObjectMapper
import java.time.Duration

/**
 * 방별 참여자 정보 저장소
 * 참여 기준: 해당 방에서 게임 관련 메시지를 보낸 사용자 (게임 완료 여부와 무관)
 *
 * BaseJsonBucketStore 패턴 사용 (RBucket + JSON 직렬화)
 */
@Component
@ConditionalOnProperty(
    prefix = "app.cache",
    name = ["enabled"],
    havingValue = "true",
    matchIfMissing = true,
)
class PlayerSetStore(
    private val redisson: RedissonReactiveClient,
    private val props: AppProperties,
    @param:org.springframework.beans.factory.annotation.Qualifier("kotlinJsonMapper")
    private val objectMapper: ObjectMapper,
) {
    companion object {
        private val log = LoggerFactory.getLogger(PlayerSetStore::class.java)
        private val TYPE_REF = object : TypeReference<Set<PlayerInfo>>() {}
    }

    suspend fun addAsync(
        roomId: String,
        userId: String,
        sender: String,
    ): Boolean {
        val current = getAllAsync(roomId)
        val isNew = !current.any { it.userId == userId }
        val updated = current + PlayerInfo(userId = userId, sender = sender)

        val json = objectMapper.writeValueAsString(updated)
        val bucket = redisson.getBucket<String>(key(roomId))
        bucket.set(json, sessionTtl()).awaitSingleOrNull()

        log.debugL { "VALKEY PLAYER_ADD room=$roomId userId=$userId sender=$sender isNew=$isNew total=${updated.size}" }
        return isNew
    }

    suspend fun getAllAsync(roomId: String): Set<PlayerInfo> {
        val bucket = redisson.getBucket<String>(key(roomId))
        val json = bucket.get().awaitSingleOrNull() ?: return emptySet()

        return runCatching { objectMapper.readValue(json, TYPE_REF) }
            .onFailure { e -> log.warn("PARSE_ERROR room={}, error={}", roomId, e.message) }
            .getOrNull()
            ?: emptySet()
    }

    suspend fun clearAsync(roomId: String) {
        val bucket = redisson.getBucket<String>(key(roomId))
        bucket.delete().awaitSingleOrNull()
        log.debugL { "VALKEY PLAYER_CLEAR room=$roomId" }
    }

    suspend fun setTtlAsync(
        roomId: String,
        ttl: Duration,
    ) {
        val bucket = redisson.getBucket<String>(key(roomId))
        bucket.expire(ttl).awaitSingleOrNull()
    }

    private fun sessionTtl(): Duration = props.cache.sessionTtlMinutes.minutes

    private fun key(roomId: String) = "${RedisKeys.PLAYERS}:$roomId"
}
