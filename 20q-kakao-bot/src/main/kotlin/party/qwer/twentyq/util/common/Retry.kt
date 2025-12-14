package party.qwer.twentyq.util.common

import kotlinx.coroutines.delay
import org.slf4j.Logger

/** 재시도 로직을 추상화 */
suspend inline fun <T> withRetry(
    config: RetryConfig,
    logTag: String,
    log: Logger,
    crossinline operation: suspend (attempt: Int) -> T,
): Result<T> {
    var lastException: Throwable? = null

    repeat(config.maxAttempts) { attempt ->
        runCatching { operation(attempt + 1) }
            .onSuccess { return Result.success(it) }
            .onFailure { ex ->
                lastException = ex
                if (!config.retryOn(ex)) {
                    return Result.failure(ex)
                }
                log.warn(
                    "{} attempt={}/{}, error={}",
                    logTag,
                    attempt + 1,
                    config.maxAttempts,
                    ex.message,
                )
                if (config.delayMillis > 0 && attempt < config.maxAttempts - 1) {
                    delay(config.delayMillis)
                }
            }
    }

    val finalException = lastException ?: IllegalStateException("retry failed: attempts=${config.maxAttempts}")
    return Result.failure(finalException)
}

/** 단순화된 재시도 - 최대 시도 횟수만 지정 */
suspend inline fun <T> withRetry(
    maxAttempts: Int,
    logTag: String,
    log: Logger,
    crossinline operation: suspend (attempt: Int) -> T,
): Result<T> =
    withRetry(
        config = RetryConfig(maxAttempts = maxAttempts),
        logTag = logTag,
        log = log,
        operation = operation,
    )

/** 타임아웃 전용 재시도 */
suspend inline fun <T> withTimeoutRetry(
    logTag: String,
    log: Logger,
    crossinline operation: suspend (attempt: Int) -> T,
): Result<T> =
    withRetry(
        config = timeoutRetryConfig,
        logTag = logTag,
        log = log,
        operation = operation,
    )
