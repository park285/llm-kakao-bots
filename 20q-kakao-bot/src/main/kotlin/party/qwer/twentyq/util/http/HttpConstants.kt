package party.qwer.twentyq.util.http

/**
 * HTTP 관련 상수
 *
 * HTTP 상태 코드, 타임아웃, 재시도 설정
 */
object HttpConstants {
    // Time conversion
    const val MILLIS_PER_SECOND = 1000

    // HTTP 상태 코드
    const val HTTP_REQUEST_TIMEOUT = 408
    const val HTTP_TOO_MANY_REQUESTS = 429
    const val HTTP_INTERNAL_SERVER_ERROR = 500
    const val HTTP_BAD_GATEWAY = 502
    const val HTTP_SERVICE_UNAVAILABLE = 503
    const val HTTP_GATEWAY_TIMEOUT = 504

    // HTTP 재시도 설정 (SDK 레벨)
    const val HTTP_RETRY_ATTEMPTS = 4 // 재시도 횟수 (총 5회 시도, 503 대응 강화)
    const val HTTP_RETRY_INITIAL_DELAY_MS = 2_000 // 초기 재시도 대기 시간 (2초, 서버 복구 시간 확보)
    const val HTTP_RETRY_MAX_DELAY_MS = 10_000 // 최대 재시도 대기 시간 (10초)
    const val HTTP_RETRY_EXP_BASE = 2.0 // Exponential backoff base

    // Ktor Client 설정
    const val KEEP_ALIVE_SECONDS = 30L
    const val MAX_CONNECTIONS_PER_ROUTE = 10
}
