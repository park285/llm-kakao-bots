package party.qwer.twentyq.util.common

import org.redisson.client.RedisException
import org.slf4j.Logger
import org.springframework.dao.DataAccessException

/** DB 작업을 안전하게 실행하고 실패 시 로깅 후 fallback 반환 */
inline fun <T> safeDbOperation(
    logTag: String,
    log: Logger,
    fallback: T,
    vararg context: Pair<String, Any?>,
    operation: () -> T,
): T =
    try {
        operation()
    } catch (e: DataAccessException) {
        logWithContext(log, logTag, e, *context)
        fallback
    }

/** Redis 작업을 안전하게 실행하고 실패 시 로깅 후 fallback 반환 */
inline fun <T> safeRedisOperation(
    logTag: String,
    log: Logger,
    fallback: T,
    vararg context: Pair<String, Any?>,
    operation: () -> T,
): T =
    try {
        operation()
    } catch (e: RedisException) {
        logWithContext(log, logTag, e, *context)
        fallback
    }

/** 예외 발생 시 로깅 후 null 반환 */
inline fun <T> safeCallOrNull(
    logTag: String,
    log: Logger,
    vararg context: Pair<String, Any?>,
    operation: () -> T,
): T? =
    runCatching { operation() }
        .onFailure { e -> logWithContext(log, logTag, e, *context) }
        .getOrNull()

/** 예외 발생 시 로깅 후 fallback 반환 */
inline fun <T> safeCallOrDefault(
    logTag: String,
    log: Logger,
    fallback: T,
    vararg context: Pair<String, Any?>,
    operation: () -> T,
): T =
    runCatching { operation() }
        .onFailure { e -> logWithContext(log, logTag, e, *context) }
        .getOrElse { fallback }

/** 예외 발생 시 로깅 후 fallback 반환 (suspend 버전) */
suspend inline fun <T> safeCallOrDefaultSuspend(
    logTag: String,
    log: Logger,
    fallback: T,
    vararg context: Pair<String, Any?>,
    crossinline operation: suspend () -> T,
): T =
    runCatching { operation() }
        .onFailure { e -> logWithContext(log, logTag, e, *context) }
        .getOrElse { fallback }

@PublishedApi
internal fun logWithContext(
    log: Logger,
    logTag: String,
    e: Throwable,
    vararg context: Pair<String, Any?>,
) {
    val contextStr = context.joinToString(", ") { it.first + "={}" }
    val contextValues = context.map { it.second }.toTypedArray()

    val message =
        if (context.isEmpty()) {
            "{} error={}"
        } else {
            "{} " + contextStr + " error={}"
        }

    log.warn(
        message,
        logTag,
        *contextValues,
        e.message,
        e,
    )
}
