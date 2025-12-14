package party.qwer.twentyq.util.logging

/**
 * 로깅 관련 상수
 *
 * 로그 텍스트 길이 제한, 샘플링 설정 등
 */
object LoggingConstants {
    // 로그 텍스트 길이 제한
    const val LOG_TEXT_SHORT = 50
    const val LOG_TEXT_MEDIUM = 100
    const val LOG_TEXT_LONG = 120
    const val LOG_TEXT_EXTRA_LONG = 500

    // Easter egg 확률
    const val DEFAULT_EASTER_EGG_PROBABILITY = 20

    // 로그 샘플링 (성능 최적화)
    const val LOG_SAMPLE_COUNT = 2 // 샘플 수
    const val LOG_SAMPLE_LIMIT_HIGH = 10 // 높은 빈도 제한
    const val LOG_SAMPLE_LIMIT_LOW = 5 // 낮은 빈도 제한
    const val LOG_SAMPLE_WINDOW_MS = 1_000L // 샘플링 윈도우 (1초)
    const val LOG_SAMPLE_WINDOW_LONG = 2_000L // 긴 윈도우 (2초)

    // 진행률 계산
    const val PERCENT_MULTIPLIER = 100
}
