package party.qwer.twentyq.util

import org.junit.jupiter.api.Assertions.assertEquals
import org.junit.jupiter.api.Assertions.assertFalse
import org.junit.jupiter.api.Assertions.assertTrue
import org.junit.jupiter.api.Test
import party.qwer.twentyq.util.common.extensions.days
import party.qwer.twentyq.util.common.extensions.hours
import party.qwer.twentyq.util.common.extensions.isLongerThan
import party.qwer.twentyq.util.common.extensions.isShorterThan
import party.qwer.twentyq.util.common.extensions.minutes
import party.qwer.twentyq.util.common.extensions.seconds
import party.qwer.twentyq.util.common.extensions.toDetailedKoreanString
import party.qwer.twentyq.util.common.extensions.toKoreanString
import party.qwer.twentyq.util.common.extensions.toMillisSafe
import party.qwer.twentyq.util.common.extensions.toReadableString
import java.time.Duration

/**
 * DurationExtensions.kt 단위 테스트
 *
 * 테스트 범위:
 * - Property Extensions (Int/Long.seconds/minutes/hours/days)
 * - 한국어 변환 (toKoreanString, toDetailedKoreanString)
 * - 영문 변환 (toReadableString)
 * - 비교 연산 (isLongerThan, isShorterThan)
 * - 유틸리티 (abs, toMillisSafe, coerceIn)
 */
class DurationExtensionsTest {
    @Test
    fun `Int seconds property - should convert to Duration`() {
        // Given/When
        val duration = 30.seconds

        // Then
        assertEquals(Duration.ofSeconds(30), duration)
    }

    @Test
    fun `Int minutes property - should convert to Duration`() {
        // Given/When
        val duration = 5.minutes

        // Then
        assertEquals(Duration.ofMinutes(5), duration)
    }

    @Test
    fun `Int hours property - should convert to Duration`() {
        // Given/When
        val duration = 3.hours

        // Then
        assertEquals(Duration.ofHours(3), duration)
    }

    @Test
    fun `Int days property - should convert to Duration`() {
        // Given/When
        val duration = 7.days

        // Then
        assertEquals(Duration.ofDays(7), duration)
    }

    @Test
    fun `Long seconds property - should convert to Duration`() {
        // Given/When
        val duration = 1000L.seconds

        // Then
        assertEquals(Duration.ofSeconds(1000), duration)
    }

    @Test
    fun `Long minutes property - should convert to Duration`() {
        // Given/When
        val duration = 60L.minutes

        // Then
        assertEquals(Duration.ofMinutes(60), duration)
    }

    @Test
    fun `Long hours property - should convert to Duration`() {
        // Given/When
        val duration = 24L.hours

        // Then
        assertEquals(Duration.ofHours(24), duration)
    }

    @Test
    fun `Long days property - should convert to Duration`() {
        // Given/When
        val duration = 365L.days

        // Then
        assertEquals(Duration.ofDays(365), duration)
    }

    @Test
    fun `toKoreanString - should return 0초 for zero duration`() {
        // Given
        val duration = Duration.ZERO

        // When
        val result = duration.toKoreanString()

        // Then
        assertEquals("0초", result)
    }

    @Test
    fun `toKoreanString - should return seconds for non-divisible duration`() {
        // Given
        val duration = Duration.ofSeconds(35)

        // When
        val result = duration.toKoreanString()

        // Then
        assertEquals("35초", result)
    }

    @Test
    fun `toKoreanString - should return minutes when divisible by 60`() {
        // Given: 5400초 = 90분
        val duration = Duration.ofMinutes(90)

        // When
        val result = duration.toKoreanString()

        // Then: "1시간 30분"이 아니라 "90분"
        assertEquals("90분", result)
    }

    @Test
    fun `toKoreanString - should return hours when divisible by 3600`() {
        // Given: 18000초 = 5시간
        val duration = 5.hours

        // When
        val result = duration.toKoreanString()

        // Then
        assertEquals("5시간", result)
    }

    @Test
    fun `toKoreanString - should return days when divisible by 86400`() {
        // Given: 259200초 = 3일
        val duration = 3.days

        // When
        val result = duration.toKoreanString()

        // Then
        assertEquals("3일", result)
    }

    @Test
    fun `toKoreanString - should return seconds for 61 seconds`() {
        // Given: 61초 (60으로 나누어떨어지지 않음)
        val duration = Duration.ofSeconds(61)

        // When
        val result = duration.toKoreanString()

        // Then
        assertEquals("61초", result)
    }

    @Test
    fun `toReadableString - should return seconds in English`() {
        // Given
        val duration = 30.seconds

        // When
        val result = duration.toReadableString()

        // Then
        assertEquals("30 seconds", result)
    }

    @Test
    fun `toReadableString - should return single second without plural`() {
        // Given
        val duration = 1.seconds

        // When
        val result = duration.toReadableString()

        // Then
        assertEquals("1 second", result)
    }

    @Test
    fun `toReadableString - should return minutes in English`() {
        // Given
        val duration = 5.minutes

        // When
        val result = duration.toReadableString()

        // Then
        assertEquals("5 minutes", result)
    }

    @Test
    fun `toReadableString - should return hours in English`() {
        // Given
        val duration = 3.hours

        // When
        val result = duration.toReadableString()

        // Then
        assertEquals("3 hours", result)
    }

    @Test
    fun `toReadableString - should return days in English`() {
        // Given
        val duration = 7.days

        // When
        val result = duration.toReadableString()

        // Then
        assertEquals("7 days", result)
    }

    @Test
    fun `toDetailedKoreanString - should return 0초 for zero duration`() {
        // Given
        val duration = Duration.ZERO

        // When
        val result = duration.toDetailedKoreanString()

        // Then
        assertEquals("0초", result)
    }

    @Test
    fun `toDetailedKoreanString - should show hours and minutes for 90 minutes`() {
        // Given
        val duration = Duration.ofMinutes(90)

        // When
        val result = duration.toDetailedKoreanString()

        // Then: toKoreanString()과 달리 복합 표현
        assertEquals("1시간 30분", result)
    }

    @Test
    fun `toDetailedKoreanString - should show all units for complex duration`() {
        // Given: 3661초 = 1시간 1분 1초
        val duration = Duration.ofSeconds(3661)

        // When
        val result = duration.toDetailedKoreanString()

        // Then
        assertEquals("1시간 1분 1초", result)
    }

    @Test
    fun `toDetailedKoreanString - should show days hours minutes seconds`() {
        // Given: 90061초 = 1일 1시간 1분 1초
        val duration = Duration.ofSeconds(90061)

        // When
        val result = duration.toDetailedKoreanString()

        // Then
        assertEquals("1일 1시간 1분 1초", result)
    }

    @Test
    fun `toDetailedKoreanString - should omit zero units`() {
        // Given: 3600초 = 1시간 0분 0초
        val duration = 1.hours

        // When
        val result = duration.toDetailedKoreanString()

        // Then: 0인 단위는 생략
        assertEquals("1시간", result)
    }

    @Test
    fun `toDetailedKoreanString - should show only minutes and seconds`() {
        // Given: 125초 = 2분 5초
        val duration = Duration.ofSeconds(125)

        // When
        val result = duration.toDetailedKoreanString()

        // Then
        assertEquals("2분 5초", result)
    }

    @Test
    fun `isLongerThan - should return true when duration is longer`() {
        // Given
        val longer = 10.minutes
        val shorter = 5.minutes

        // When/Then: infix 함수 사용
        assertTrue(longer isLongerThan shorter)
    }

    @Test
    fun `isLongerThan - should return false when duration is equal`() {
        // Given
        val duration1 = 5.minutes
        val duration2 = 5.minutes

        // When/Then
        assertFalse(duration1 isLongerThan duration2)
    }

    @Test
    fun `isLongerThan - should return false when duration is shorter`() {
        // Given
        val shorter = 3.seconds
        val longer = 10.seconds

        // When/Then
        assertFalse(shorter isLongerThan longer)
    }

    @Test
    fun `isShorterThan - should return true when duration is shorter`() {
        // Given
        val shorter = 3.seconds
        val longer = 10.seconds

        // When/Then: infix 함수 사용
        assertTrue(shorter isShorterThan longer)
    }

    @Test
    fun `isShorterThan - should return false when duration is equal`() {
        // Given
        val duration1 = 5.minutes
        val duration2 = 5.minutes

        // When/Then
        assertFalse(duration1 isShorterThan duration2)
    }

    @Test
    fun `isShorterThan - should return false when duration is longer`() {
        // Given
        val longer = 10.minutes
        val shorter = 5.minutes

        // When/Then
        assertFalse(longer isShorterThan shorter)
    }

    @Test
    fun `abs - should return positive duration for negative`() {
        // Given
        val negative = Duration.ofSeconds(-30)

        // When
        val result = negative.abs()

        // Then
        assertEquals(Duration.ofSeconds(30), result)
    }

    @Test
    fun `abs - should return same duration for positive`() {
        // Given
        val positive = Duration.ofSeconds(30)

        // When
        val result = positive.abs()

        // Then
        assertEquals(Duration.ofSeconds(30), result)
    }

    @Test
    fun `abs - should return zero for zero`() {
        // Given
        val zero = Duration.ZERO

        // When
        val result = zero.abs()

        // Then
        assertEquals(Duration.ZERO, result)
    }

    @Test
    fun `toMillisSafe - should return millis for normal duration`() {
        // Given
        val duration = 5.seconds

        // When
        val result = duration.toMillisSafe()

        // Then
        assertEquals(5000L, result)
    }

    @Test
    fun `toMillisSafe - should handle very large duration gracefully`() {
        // Given: Long.MAX_VALUE일 오버플로우 위험
        val hugeDuration = Duration.ofDays(Long.MAX_VALUE / 86400)

        // When
        val result = hugeDuration.toMillisSafe()

        // Then: 예외 없이 값 반환
        assertTrue(result > 0)
    }

    @Test
    fun `coerceIn - should clamp to min boundary`() {
        // Given
        val tooSmall = 1.seconds

        // When
        val result = tooSmall.coerceIn(5.seconds, 10.seconds)

        // Then
        assertEquals(5.seconds, result)
    }

    @Test
    fun `coerceIn - should clamp to max boundary`() {
        // Given
        val tooLarge = 15.seconds

        // When
        val result = tooLarge.coerceIn(5.seconds, 10.seconds)

        // Then
        assertEquals(10.seconds, result)
    }

    @Test
    fun `coerceIn - should return original when within range`() {
        // Given
        val withinRange = 7.seconds

        // When
        val result = withinRange.coerceIn(5.seconds, 10.seconds)

        // Then
        assertEquals(7.seconds, result)
    }

    @Test
    fun `isZero property - should return true for zero duration`() {
        // Given
        val zero = Duration.ZERO

        // When/Then
        assertTrue(zero.isZero)
    }

    @Test
    fun `isZero property - should return false for non-zero duration`() {
        // Given
        val nonZero = 1.seconds

        // When/Then
        assertFalse(nonZero.isZero)
    }

    @Test
    fun `isPositive property - should return true for positive duration`() {
        // Given
        val positive = 1.seconds

        // When/Then
        assertTrue(positive.isPositive)
    }

    @Test
    fun `isPositive property - should return false for zero duration`() {
        // Given
        val zero = Duration.ZERO

        // When/Then
        assertFalse(zero.isPositive)
    }

    @Test
    fun `isPositive property - should return false for negative duration`() {
        // Given
        val negative = Duration.ofSeconds(-1)

        // When/Then
        assertFalse(negative.isPositive)
    }
}
