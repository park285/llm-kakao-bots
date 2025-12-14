package io.github.kapu.turtlesoup.utils

import kotlin.time.Duration
import kotlin.time.Duration.Companion.days
import kotlin.time.Duration.Companion.hours
import kotlin.time.Duration.Companion.milliseconds
import kotlin.time.Duration.Companion.minutes
import kotlin.time.Duration.Companion.seconds

// 내부 계산용 상수
private const val SECONDS_PER_MINUTE = 60L
private const val SECONDS_PER_HOUR = 3600L
private const val SECONDS_PER_DAY = 86400L
private const val MINUTES_PER_HOUR = 60L

// Int 확장 프로퍼티
val Int.sec: Duration get() = this.seconds
val Int.min: Duration get() = this.minutes
val Int.hrs: Duration get() = this.hours
val Int.dys: Duration get() = this.days

// Long 확장 프로퍼티
val Long.sec: Duration get() = this.seconds
val Long.min: Duration get() = this.minutes
val Long.hrs: Duration get() = this.hours
val Long.dys: Duration get() = this.days
val Long.ms: Duration get() = this.milliseconds

/** Duration을 한국어 문자열로 변환 (예: "1시간 30분 45초") */
fun Duration.toKoreanString(): String {
    val hrs = inWholeHours
    val mins = (inWholeMinutes % MINUTES_PER_HOUR)
    val secs = (inWholeSeconds % SECONDS_PER_MINUTE)

    return buildString {
        if (hrs > 0) append("${hrs}시간 ")
        if (mins > 0) append("${mins}분 ")
        if (secs > 0 || isEmpty()) append("${secs}초")
    }.trim()
}

/** Duration을 간단한 한국어 문자열로 변환 (가장 큰 단위만) */
fun Duration.toSimpleKoreanString(): String {
    val totalSeconds = inWholeSeconds

    return when {
        totalSeconds == 0L -> "0초"
        totalSeconds % SECONDS_PER_DAY == 0L -> "${totalSeconds / SECONDS_PER_DAY}일"
        totalSeconds % SECONDS_PER_HOUR == 0L -> "${totalSeconds / SECONDS_PER_HOUR}시간"
        totalSeconds % SECONDS_PER_MINUTE == 0L -> "${totalSeconds / SECONDS_PER_MINUTE}분"
        else -> "${totalSeconds}초"
    }
}

/** Duration이 특정 Duration보다 긴지 확인 */
infix fun Duration.isLongerThan(other: Duration): Boolean = this > other

/** Duration이 특정 Duration보다 짧은지 확인 */
infix fun Duration.isShorterThan(other: Duration): Boolean = this < other

/** Duration을 밀리초로 안전하게 변환 (overflow 방지) */
fun Duration.toMillisSafe(): Long =
    try {
        inWholeMilliseconds
    } catch (_: ArithmeticException) {
        Long.MAX_VALUE
    }

/** Duration을 특정 범위로 제한 */
fun Duration.coerceIn(
    min: Duration,
    max: Duration,
): Duration =
    when {
        this < min -> min
        this > max -> max
        else -> this
    }

/** 현재 시각의 밀리초 타임스탬프 반환 */
fun nowMillis(): Long = System.currentTimeMillis()

/** 블록 실행 시간을 밀리초 단위로 측정 */
inline fun <T> measureDurationMillis(block: () -> T): Pair<T, Long> {
    val start = nowMillis()
    val result = block()
    val duration = nowMillis() - start
    return result to duration
}
