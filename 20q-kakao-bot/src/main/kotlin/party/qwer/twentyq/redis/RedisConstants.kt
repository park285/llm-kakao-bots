package party.qwer.twentyq.redis

/**
 * Redis 운영 상수 묶음
 *
 * - TTL, 큐 크기 등 공용 파라미터의 단일 진실 소스 유지
 */
object RedisConstants {
    const val QUEUE_TTL_SECONDS = 300L
    const val MAX_QUEUE_SIZE = 5
    const val LOCK_TTL_SECONDS = 300L
    const val LOCK_BLOCK_TIMEOUT_SECONDS = 180L
    const val PROCESSING_TTL_SECONDS = 200L
    const val CLEANUP_TIMEOUT_MS = 10_000L
}
