package party.qwer.twentyq.redis.session

import org.redisson.api.RedissonReactiveClient
import org.slf4j.LoggerFactory
import org.springframework.boot.autoconfigure.condition.ConditionalOnProperty
import org.springframework.stereotype.Component
import party.qwer.twentyq.config.AppProperties
import party.qwer.twentyq.logging.debugL
import party.qwer.twentyq.redis.BaseJsonBucketStore
import party.qwer.twentyq.redis.RedisKeys
import party.qwer.twentyq.util.common.extensions.minutes
import tools.jackson.databind.ObjectMapper
import java.time.Duration

/**
 * 카테고리 정보 저장소
 */
@Component
@ConditionalOnProperty(
    prefix = "app.cache",
    name = ["enabled"],
    havingValue = "true",
    matchIfMissing = true,
)
class CategoryStore(
    redisson: RedissonReactiveClient,
    objectMapper: ObjectMapper,
    private val props: AppProperties,
) : BaseJsonBucketStore<String>(redisson, objectMapper, String::class.java) {
    override val log = LoggerFactory.getLogger(CategoryStore::class.java)

    override fun serialize(value: String): String = value

    override fun deserialize(data: String): String = data

    suspend fun getAsync(roomId: String): String? {
        val category = read(key(roomId), contextKey = "room=$roomId")
        log.debugL { "VALKEY CATEGORY_GET room=$roomId category=$category" }
        return category
    }

    suspend fun saveAsync(
        roomId: String,
        category: String?,
    ) {
        if (category != null) {
            write(key(roomId), category, sessionTtl())
            log.debugL { "VALKEY CATEGORY_SAVE room=$roomId category=$category" }
        } else {
            delete(key(roomId))
            log.debugL { "VALKEY CATEGORY_DELETE room=$roomId" }
        }
    }

    suspend fun setTtlAsync(
        roomId: String,
        ttl: Duration,
    ) {
        expire(key(roomId), ttl)
    }

    private fun sessionTtl(): Duration = props.cache.sessionTtlMinutes.minutes

    private fun key(roomId: String) = "${RedisKeys.CATEGORY}:$roomId"
}
