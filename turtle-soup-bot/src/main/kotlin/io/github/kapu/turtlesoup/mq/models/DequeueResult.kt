package io.github.kapu.turtlesoup.mq.models

/** 디큐 결과 */
sealed class DequeueResult {
    data class Success(val message: PendingMessage) : DequeueResult()

    data object Empty : DequeueResult()

    data object Exhausted : DequeueResult() // 루프 제한 도달, 즉시 재폴링 필요
}
