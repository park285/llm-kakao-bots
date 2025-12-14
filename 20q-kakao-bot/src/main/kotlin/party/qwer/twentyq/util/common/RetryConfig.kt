package party.qwer.twentyq.util.common

import kotlinx.coroutines.TimeoutCancellationException
import java.util.concurrent.TimeoutException

/** 재시도 설정 */
data class RetryConfig(
    val maxAttempts: Int = 2,
    val delayMillis: Long = 0,
    val retryOn: (Throwable) -> Boolean = { true },
)

/** 타임아웃 전용 재시도 설정 */
val timeoutRetryConfig =
    RetryConfig(
        maxAttempts = 2,
        retryOn = { it is TimeoutCancellationException || it is TimeoutException },
    )
