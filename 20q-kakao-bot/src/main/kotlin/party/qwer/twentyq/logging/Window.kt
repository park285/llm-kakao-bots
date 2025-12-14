package party.qwer.twentyq.logging

import java.util.concurrent.atomic.AtomicLong

// 로깅 샘플링 윈도우
internal data class Window(
    val untilMillis: AtomicLong,
    val count: AtomicLong,
)
