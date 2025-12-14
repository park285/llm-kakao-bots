package party.qwer.twentyq.logging

import com.github.benmanes.caffeine.cache.Cache
import org.slf4j.Logger
import party.qwer.twentyq.util.cache.CacheBuilders
import party.qwer.twentyq.util.common.extensions.nowMillis
import java.time.Duration
import java.util.concurrent.atomic.AtomicLong

/**
 * 경량 지연 로깅 헬퍼 및 핫 패스 오버헤드 감소를 위한 샘플링 유틸리티
 */
object LoggingExtensions {
    private const val CACHE_TTL_MINUTES = 10L

    private val windows: Cache<String, Window> =
        CacheBuilders.expireAfterAccess(
            maxSize = 1000L,
            ttl = Duration.ofMinutes(CACHE_TTL_MINUTES),
            recordStats = false,
        )

    /**
     * 주어진 [key]에 대해 [windowMillis] 동안 최대 [limit]회만 로깅.
     * DEBUG 모드가 아닐 경우 샘플링 자체를 비활성화 (성능 최적화).
     */
    fun Logger.sampled(
        key: String,
        limit: Int = 5,
        windowMillis: Long = 1_000,
        block: (Logger) -> Unit,
    ) {
        if (!isDebugEnabled) return

        val now = nowMillis()
        val w =
            windows.get(key) {
                Window(AtomicLong(now + windowMillis), AtomicLong(0))
            }
        val until = w.untilMillis.get()
        if (now > until) {
            // reset window
            if (w.untilMillis.compareAndSet(until, now + windowMillis)) {
                w.count.set(0)
            }
        }
        val current = w.count.incrementAndGet()
        if (current <= limit) {
            block(this)
        }
    }
}

inline fun Logger.traceL(msg: () -> String) {
    if (isTraceEnabled) trace(msg())
}

inline fun Logger.debugL(msg: () -> String) {
    if (isDebugEnabled) debug(msg())
}

inline fun Logger.infoL(msg: () -> String) {
    if (isInfoEnabled) info(msg())
}

inline fun Logger.warnL(msg: () -> String) {
    if (isWarnEnabled) warn(msg())
}

inline fun Logger.errorL(msg: () -> String) {
    if (isErrorEnabled) error(msg())
}
