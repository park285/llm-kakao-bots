package party.qwer.twentyq.redis

import kotlinx.coroutines.reactor.awaitSingleOrNull
import org.redisson.api.RedissonReactiveClient
import org.slf4j.Logger
import tools.jackson.databind.ObjectMapper
import java.time.Duration

/**
 * 공통 JSON Bucket Store (Jackson 기반)
 *
 * - 직렬화/역직렬화 + TTL + 존재/삭제 처리 공용화
 * - 파싱 실패 시 최소 로그 보장 (contextKey 기반)
 * - ObjectMapper 주입받아 중앙 관리형 설정 사용 (CONVENTIONS 준수)
 */
abstract class BaseJsonBucketStore<T : Any>(
    private val redisson: RedissonReactiveClient,
    private val objectMapper: ObjectMapper,
    private val type: Class<T>,
) {
    protected abstract val log: Logger

    protected open fun serialize(value: T): String = objectMapper.writeValueAsString(value)

    protected open fun deserialize(data: String): T = objectMapper.readValue(data, type)

    protected open fun logParseError(
        contextKey: String,
        error: Throwable,
    ) {
        log.warn("PARSE_ERROR key={}, error={}", contextKey, error.message)
    }

    protected suspend fun read(
        key: String,
        contextKey: String = key,
    ): T? {
        val data = redisson.getBucket<String>(key).get().awaitSingleOrNull() ?: return null
        return runCatching { deserialize(data) }
            .onFailure { logParseError(contextKey, it) }
            .getOrNull()
    }

    protected suspend fun write(
        key: String,
        value: T,
        ttl: Duration,
    ) {
        val json = serialize(value)
        redisson.getBucket<String>(key).set(json, ttl).awaitSingleOrNull()
    }

    protected suspend fun delete(key: String) {
        redisson.getBucket<String>(key).delete().awaitSingleOrNull()
    }

    protected suspend fun expire(
        key: String,
        ttl: Duration,
    ): Boolean =
        redisson
            .getBucket<String>(key)
            .expireAsync(ttl)

    protected suspend fun exists(key: String): Boolean =
        redisson
            .getBucket<String>(key)
            .isExists
            .awaitSingleOrNull() ?: false
}
