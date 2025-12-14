package party.qwer.twentyq.redis.voting

import org.redisson.api.RedissonReactiveClient
import org.slf4j.LoggerFactory
import org.springframework.boot.autoconfigure.condition.ConditionalOnProperty
import org.springframework.stereotype.Component
import party.qwer.twentyq.logging.debugL
import party.qwer.twentyq.model.SurrenderVote
import party.qwer.twentyq.redis.BaseJsonBucketStore
import party.qwer.twentyq.redis.RedisKeys
import party.qwer.twentyq.util.common.extensions.seconds
import tools.jackson.databind.ObjectMapper

/**
 * 항복 투표 저장소
 */
@Component
@ConditionalOnProperty(
    prefix = "app.cache",
    name = ["enabled"],
    havingValue = "true",
    matchIfMissing = true,
)
class SurrenderVoteStore(
    redisson: RedissonReactiveClient,
    objectMapper: ObjectMapper,
) : BaseJsonBucketStore<SurrenderVote>(redisson, objectMapper, SurrenderVote::class.java) {
    override val log = LoggerFactory.getLogger(SurrenderVoteStore::class.java)

    suspend fun getAsync(roomId: String): SurrenderVote? =
        read(
            key(roomId),
            contextKey = "room=$roomId",
        )

    suspend fun saveAsync(
        roomId: String,
        vote: SurrenderVote,
        ttlSeconds: Long = 120,
    ) {
        write(key(roomId), vote, ttlSeconds.seconds)
        log.debugL { "SAVE room=$roomId, vote=$vote, ttl=${ttlSeconds}s" }
    }

    suspend fun isActiveAsync(roomId: String): Boolean = exists(key(roomId))

    suspend fun approveAsync(
        roomId: String,
        userId: String,
        ttlSeconds: Long = 120,
    ): SurrenderVote? {
        val vote = getAsync(roomId) ?: return null
        val updated = vote.approve(userId)
        saveAsync(roomId, updated, ttlSeconds)
        log.debugL {
            "APPROVE room=$roomId, userId=$userId, approvals=${updated.approvals.size}/${updated.requiredApprovals()}"
        }
        return updated
    }

    suspend fun clearAsync(roomId: String) {
        delete(key(roomId))
        log.debugL { "CLEAR room=$roomId" }
    }

    private fun key(roomId: String) = "${RedisKeys.SURRENDER_VOTE}:$roomId"
}
