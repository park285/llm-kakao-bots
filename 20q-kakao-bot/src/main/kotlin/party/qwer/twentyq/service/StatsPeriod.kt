package party.qwer.twentyq.service

import java.time.Instant
import java.time.LocalDate
import java.time.ZoneId
import java.time.temporal.TemporalAdjusters

/**
 * 통계 조회 기간
 */
enum class StatsPeriod {
    DAILY,
    WEEKLY,
    MONTHLY,
    ALL,
    ;

    /**
     * 기간의 시작 시간 계산 (한국 시간 기준)
     */
    fun getStartTime(): Instant? {
        val koreaZone = ZoneId.of("Asia/Seoul")
        val now = LocalDate.now(koreaZone)

        return when (this) {
            DAILY -> now.atStartOfDay(koreaZone).toInstant()
            WEEKLY ->
                now
                    .with(TemporalAdjusters.previousOrSame(java.time.DayOfWeek.MONDAY))
                    .atStartOfDay(koreaZone)
                    .toInstant()
            MONTHLY -> now.withDayOfMonth(1).atStartOfDay(koreaZone).toInstant()
            ALL -> null
        }
    }

    companion object {
        /**
         * 문자열을 StatsPeriod로 변환
         */
        fun fromString(value: String?): StatsPeriod =
            when (value) {
                "일간", "daily" -> DAILY
                "주간", "weekly" -> WEEKLY
                "월간", "monthly" -> MONTHLY
                else -> ALL
            }
    }
}
