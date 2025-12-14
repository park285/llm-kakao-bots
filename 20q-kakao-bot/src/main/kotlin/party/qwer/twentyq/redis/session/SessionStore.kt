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
 * 세션 데이터 저장소
 */
@Component
@ConditionalOnProperty(
    prefix = "app.cache",
    name = ["enabled"],
    havingValue = "true",
    matchIfMissing = true,
)
class SessionStore(
    redisson: RedissonReactiveClient,
    objectMapper: ObjectMapper,
    private val props: AppProperties,
) : BaseJsonBucketStore<String>(redisson, objectMapper, String::class.java) {
    override val log = LoggerFactory.getLogger(SessionStore::class.java)

    override fun serialize(value: String): String = value

    override fun deserialize(data: String): String = data

    suspend fun getAsync(roomId: String): String? {
        val data = read(key(roomId), contextKey = "room=$roomId")
        log.debugL {
            "VALKEY SESSION_GET room=$roomId exists=${data != null} size=${data?.length ?: 0}"
        }
        return data
    }

    suspend fun saveAsync(
        roomId: String,
        data: String,
    ) {
        val ttl = sessionTtl()
        write(key(roomId), data, ttl)
        log.debugL {
            "VALKEY SESSION_SAVE room=$roomId size=${data.length} ttl=${props.cache.sessionTtlMinutes}min"
        }
    }

    suspend fun deleteAsync(roomId: String) {
        delete(key(roomId))
        log.debugL { "VALKEY SESSION_DELETE room=$roomId" }
    }

    suspend fun setTtlAsync(
        roomId: String,
        ttl: Duration,
    ): Boolean {
        val result = expire(key(roomId), ttl)
        log.debugL { "VALKEY SESSION_TTL room=$roomId ttl=$ttl success=$result" }
        return result
    }

    private fun sessionTtl(): Duration = props.cache.sessionTtlMinutes.minutes

    private fun key(roomId: String) = "${RedisKeys.SESSION}:$roomId"
}
