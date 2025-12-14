package party.qwer.twentyq.util.common.extensions

import org.slf4j.LoggerFactory
import party.qwer.twentyq.logging.debugL
import java.time.Duration
import kotlin.time.Clock

/**
 * Duration 변환 상수
 */
private val log = LoggerFactory.getLogger("DurationExtensions")

private object TimeConstants {
    const val SECONDS_PER_MINUTE = 60L
    const val SECONDS_PER_HOUR = 3600L
    const val SECONDS_PER_DAY = 86400L
}

val Int.seconds: Duration
    get() = Duration.ofSeconds(this.toLong())

val Int.minutes: Duration
    get() = Duration.ofMinutes(this.toLong())

val Int.hours: Duration
    get() = Duration.ofHours(this.toLong())

val Int.days: Duration
    get() = Duration.ofDays(this.toLong())

val Long.seconds: Duration
    get() = Duration.ofSeconds(this)

val Long.minutes: Duration
    get() = Duration.ofMinutes(this)

val Long.hours: Duration
    get() = Duration.ofHours(this)

val Long.days: Duration
    get() = Duration.ofDays(this)

/**
 * Duration을 한국어 문자열로 변환
 *
 * 예:
 * - 30.seconds -> "30초"
 * - 5.minutes -> "5분"
 * - 3.hours -> "3시간"
 * - 7.days -> "7일"
 */
fun Duration.toKoreanString(): String {
    val totalSeconds = seconds

    return when {
        totalSeconds == 0L -> "0초"
        totalSeconds % TimeConstants.SECONDS_PER_DAY == 0L -> "${totalSeconds / TimeConstants.SECONDS_PER_DAY}일"
        totalSeconds % TimeConstants.SECONDS_PER_HOUR == 0L -> "${totalSeconds / TimeConstants.SECONDS_PER_HOUR}시간"
        totalSeconds % TimeConstants.SECONDS_PER_MINUTE == 0L -> "${totalSeconds / TimeConstants.SECONDS_PER_MINUTE}분"
        else -> "${totalSeconds}초"
    }
}

/**
 * Duration을 영문 문자열로 변환
 *
 * 예:
 * - 30.seconds -> "30 seconds"
 * - 5.minutes -> "5 minutes"
 * - 3.hours -> "3 hours"
 */
fun Duration.toReadableString(): String {
    val totalSeconds = seconds

    return when {
        totalSeconds == 0L -> "0 seconds"
        totalSeconds % TimeConstants.SECONDS_PER_DAY == 0L -> {
            val days = totalSeconds / TimeConstants.SECONDS_PER_DAY
            "$days day${if (days > 1) "s" else ""}"
        }
        totalSeconds % TimeConstants.SECONDS_PER_HOUR == 0L -> {
            val hours = totalSeconds / TimeConstants.SECONDS_PER_HOUR
            "$hours hour${if (hours > 1) "s" else ""}"
        }
        totalSeconds % TimeConstants.SECONDS_PER_MINUTE == 0L -> {
            val minutes = totalSeconds / TimeConstants.SECONDS_PER_MINUTE
            "$minutes minute${if (minutes > 1) "s" else ""}"
        }
        else -> "$totalSeconds second${if (totalSeconds > 1) "s" else ""}"
    }
}

/**
 * Duration을 한국어 문자열로 변환
 *
 * 예:
 * - 90.minutes -> "1시간 30분"
 * - 3661.seconds -> "1시간 1분 1초"
 */
fun Duration.toDetailedKoreanString(): String {
    val totalSeconds = seconds

    if (totalSeconds == 0L) return "0초"

    val days = totalSeconds / TimeConstants.SECONDS_PER_DAY
    val hours = (totalSeconds % TimeConstants.SECONDS_PER_DAY) / TimeConstants.SECONDS_PER_HOUR
    val minutes = (totalSeconds % TimeConstants.SECONDS_PER_HOUR) / TimeConstants.SECONDS_PER_MINUTE
    val secs = totalSeconds % TimeConstants.SECONDS_PER_MINUTE

    return buildString {
        if (days > 0) append("${days}일 ")
        if (hours > 0) append("${hours}시간 ")
        if (minutes > 0) append("${minutes}분 ")
        if (secs > 0) append("${secs}초")
    }.trim()
}

/**
 * Duration이 특정 Duration보다 긴지 확인
 *
 * 예: `if (elapsed.isLongerThan(5.seconds)) { ... }`
 */
infix fun Duration.isLongerThan(other: Duration): Boolean = this > other

/**
 * Duration이 특정 Duration보다 짧은지 확인
 *
 * 예: `if (elapsed.isShorterThan(10.seconds)) { ... }`
 */
infix fun Duration.isShorterThan(other: Duration): Boolean = this < other

/**
 * Duration을 밀리초로 안전하게 변환
 * - Long.MAX_VALUE 초과 시 Long.MAX_VALUE 반환
 */
fun Duration.toMillisSafe(): Long =
    try {
        toMillis()
    } catch (e: ArithmeticException) {
        // Duration이 Long.MAX_VALUE 밀리초를 초과하는 경우 (약 292억년)
        log.debugL {
            "Duration overflow detected (${e.message}), returning Long.MAX_VALUE: $this"
        }
        Long.MAX_VALUE
    }

/**
 * Duration을 특정 범위로 제한
 *
 * 예: `timeout.coerceIn(1.seconds, 30.seconds)`
 */
fun Duration.coerceIn(
    min: Duration,
    max: Duration,
): Duration =
    when {
        this < min -> min
        this > max -> max
        else -> this
    }

/**
 * 현재 시각의 밀리초 타임스탬프 반환
 * kotlin.time.Clock.System 기반
 */
fun nowMillis(): Long = Clock.System.now().toEpochMilliseconds()

/**
 * 블록 실행 시간을 밀리초 단위로 측정
 *
 * @param block 측정할 작업
 * @return Pair(블록 실행 결과, 소요 시간 밀리초)
 *
 * 예:
 * ```
 * val (result, duration) = measureDurationMillis {
 *     expensiveOperation()
 * }
 * log.info("Operation completed in {}ms", duration)
 * ```
 */
inline fun <T> measureDurationMillis(block: () -> T): Pair<T, Long> {
    val start = nowMillis()
    val result = block()
    val duration = nowMillis() - start
    return result to duration
}
