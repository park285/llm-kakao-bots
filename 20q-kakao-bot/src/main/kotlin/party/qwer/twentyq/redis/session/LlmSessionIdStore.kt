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
 * LLM 세션 ID 저장소
 */
@Component
@ConditionalOnProperty(
    prefix = "app.cache",
    name = ["enabled"],
    havingValue = "true",
    matchIfMissing = true,
)
class LlmSessionIdStore(
    redisson: RedissonReactiveClient,
    objectMapper: ObjectMapper,
    private val props: AppProperties,
) : BaseJsonBucketStore<String>(redisson, objectMapper, String::class.java) {
    override val log = LoggerFactory.getLogger(LlmSessionIdStore::class.java)

    override fun serialize(value: String): String = value

    override fun deserialize(data: String): String = data

    suspend fun getAsync(chatId: String): String? {
        val sessionId = read(key(chatId), contextKey = "chat=$chatId")
        log.debugL { "VALKEY LLM_SESSION_GET chat=$chatId sessionId=$sessionId" }
        return sessionId
    }

    suspend fun saveAsync(
        chatId: String,
        sessionId: String?,
    ) {
        if (sessionId != null) {
            write(key(chatId), sessionId, sessionTtl())
            log.debugL { "VALKEY LLM_SESSION_SAVE chat=$chatId sessionId=$sessionId" }
        } else {
            delete(key(chatId))
            log.debugL { "VALKEY LLM_SESSION_DELETE chat=$chatId" }
        }
    }

    suspend fun deleteAsync(chatId: String) {
        delete(key(chatId))
        log.debugL { "VALKEY LLM_SESSION_DELETE chat=$chatId" }
    }

    suspend fun setTtlAsync(
        chatId: String,
        ttl: Duration,
    ) {
        expire(key(chatId), ttl)
    }

    private fun sessionTtl(): Duration = props.cache.sessionTtlMinutes.minutes

    private fun key(chatId: String) = "${RedisKeys.LLM_SESSION}:$chatId"
}
