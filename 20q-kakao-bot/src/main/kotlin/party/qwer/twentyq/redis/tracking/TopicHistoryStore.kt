package party.qwer.twentyq.redis.tracking

import kotlinx.coroutines.reactor.awaitSingleOrNull
import org.redisson.api.RedissonReactiveClient
import org.slf4j.LoggerFactory
import org.springframework.boot.autoconfigure.condition.ConditionalOnProperty
import org.springframework.stereotype.Component
import party.qwer.twentyq.config.AppProperties
import party.qwer.twentyq.logging.debugL
import party.qwer.twentyq.model.RiddleCategory
import party.qwer.twentyq.redis.RedisKeys
import party.qwer.twentyq.redis.expireAsync
import party.qwer.twentyq.util.common.extensions.hours

@Component
@ConditionalOnProperty(
    prefix = "app.cache",
    name = ["enabled"],
    havingValue = "true",
    matchIfMissing = true,
)
class TopicHistoryStore(
    private val redisson: RedissonReactiveClient,
    private val props: AppProperties,
) {
    companion object {
        private val log = LoggerFactory.getLogger(TopicHistoryStore::class.java)
    }

    suspend fun getRecentAsync(
        roomId: String,
        limit: Int = 20,
    ): List<String> {
        val list = redisson.getList<String>(key(roomId))
        val topics = list.range(0, limit - 1).awaitSingleOrNull() ?: emptyList()
        log.debugL { "GET room=$roomId, limit=$limit, found=${topics.size}" }
        return topics
    }

    suspend fun getRecentAsync(
        roomId: String,
        category: String,
        limit: Int = 20,
    ): List<String> {
        val list = redisson.getList<String>(key(roomId, category))
        val topics = list.range(0, limit - 1).awaitSingleOrNull() ?: emptyList()
        log.debugL { "GET room=$roomId, category=$category, limit=$limit, found=${topics.size}" }
        return topics
    }

    suspend fun getBannedTopics(
        roomId: String,
        category: String?,
        limit: Int,
    ): List<String> {
        val globalRecent = getRecentAsync(roomId, limit)
        val categoryRecents =
            if (category != null) {
                listOf(getRecentAsync(roomId, category, limit))
            } else {
                RiddleCategory
                    .entries
                    .filterNot { it == RiddleCategory.ANY }
                    .map { cat -> getRecentAsync(roomId, cat.name.lowercase(), limit) }
            }
        val merged =
            (sequenceOf(globalRecent) + categoryRecents)
                .flatten()
                .distinct()
                .toList()
        log.debugL {
            "GET_BANNED room=$roomId, category=${category ?: "all"}, " +
                "limit=$limit, global=${globalRecent.size}, categories=${categoryRecents.sumOf { it.size }}"
        }
        return merged
    }

    suspend fun addAsync(
        roomId: String,
        category: String,
        topic: String,
    ) {
        val limit = props.riddle.game.recentTopicsLimit
        addWithLimit(key(roomId), topic, limit)
        addWithLimit(key(roomId, category), topic, limit)
        log.debugL { "ADD room=$roomId, category='$category', topic='$topic', limit=$limit (global+category)" }
    }

    suspend fun clearAllAsync(roomId: String) {
        val pattern = "${RedisKeys.TOPICS}:$roomId*"
        val deletedCount = redisson.keys.deleteByPattern(pattern).awaitSingleOrNull() ?: 0L

        if (deletedCount > 0) {
            log.info("CLEAR_ALL room={}, deletedKeys={}", roomId, deletedCount)
        }
    }

    private fun key(roomId: String) = "${RedisKeys.TOPICS}:$roomId"

    private fun key(
        roomId: String,
        category: String,
    ) = "${RedisKeys.TOPICS}:$roomId:$category"

    private suspend fun addWithLimit(
        key: String,
        topic: String,
        limit: Int,
    ) {
        val list = redisson.getList<String>(key)
        list.add(0, topic).awaitSingleOrNull()
        list.trim(0, limit - 1).awaitSingleOrNull()
        // TTL 설정: 중복 방지를 위해 12시간 유지
        list.expireAsync(12.hours)
    }
}
