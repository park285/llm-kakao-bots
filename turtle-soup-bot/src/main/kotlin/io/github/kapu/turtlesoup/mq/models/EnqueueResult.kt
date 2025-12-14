package io.github.kapu.turtlesoup.mq.models

/** 큐잉 결과 */
enum class EnqueueResult {
    SUCCESS,
    DUPLICATE,
    QUEUE_FULL,
}
