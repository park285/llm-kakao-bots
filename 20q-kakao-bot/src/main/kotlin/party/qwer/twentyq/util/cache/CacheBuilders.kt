package party.qwer.twentyq.util.cache

import com.github.benmanes.caffeine.cache.Cache
import com.github.benmanes.caffeine.cache.Caffeine
import java.time.Duration

object CacheBuilders {
    // expireAfterWrite 캐시 생성 헬퍼
    fun <K : Any, V : Any> expireAfterWrite(
        maxSize: Long,
        ttl: Duration,
        recordStats: Boolean = false,
    ): Cache<K, V> {
        val builder =
            Caffeine
                .newBuilder()
                .maximumSize(maxSize)
                .expireAfterWrite(ttl)
        if (recordStats) builder.recordStats()
        return builder.build<K, V>()
    }

    // expireAfterAccess 캐시 생성 헬퍼
    fun <K : Any, V : Any> expireAfterAccess(
        maxSize: Long,
        ttl: Duration,
        recordStats: Boolean = false,
    ): Cache<K, V> {
        val builder =
            Caffeine
                .newBuilder()
                .maximumSize(maxSize)
                .expireAfterAccess(ttl)
        if (recordStats) builder.recordStats()
        return builder.build<K, V>()
    }

    // expireAfterAccess with eviction listener
    fun <K : Any, V : Any> expireAfterAccessWithListener(
        maxSize: Long,
        ttl: Duration,
        recordStats: Boolean = false,
        onEviction: (K, V?) -> Unit,
    ): Cache<K, V> {
        val builder =
            Caffeine
                .newBuilder()
                .maximumSize(maxSize)
                .expireAfterAccess(ttl)
                .removalListener<K, V> { key, value, _ ->
                    if (key != null) {
                        onEviction(key, value)
                    }
                }
        if (recordStats) builder.recordStats()
        return builder.build<K, V>()
    }
}
