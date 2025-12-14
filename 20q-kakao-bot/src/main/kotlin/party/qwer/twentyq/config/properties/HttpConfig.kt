package party.qwer.twentyq.config.properties

/**
 * HTTP 클라이언트 설정
 *
 * @property responseTimeoutSeconds 응답 대기 시간 (초)
 * @property connectTimeoutMillis 연결 타임아웃 (밀리초)
 */
data class HttpConfig(
    val responseTimeoutSeconds: Long = 30,
    val connectTimeoutMillis: Int = 15000,
)
