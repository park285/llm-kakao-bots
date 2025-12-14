package party.qwer.twentyq.rest

import io.ktor.client.plugins.HttpRequestTimeoutException
import io.ktor.client.statement.HttpResponse
import io.ktor.client.statement.bodyAsText
import org.slf4j.LoggerFactory
import org.springframework.beans.factory.annotation.Qualifier
import org.springframework.stereotype.Component
import party.qwer.twentyq.rest.dto.LlmErrorResponse
import party.qwer.twentyq.rest.dto.ParsedError
import tools.jackson.core.JacksonException
import tools.jackson.databind.ObjectMapper
import java.io.IOException
import java.net.ConnectException
import java.net.SocketException
import java.net.SocketTimeoutException
import java.util.UUID

/** REST 에러 핸들링 유틸리티 */
@Component
class RestErrorHandler(
    @param:Qualifier("kotlinJsonMapper")
    private val objectMapper: ObjectMapper,
) {
    companion object {
        private val log = LoggerFactory.getLogger(RestErrorHandler::class.java)

        /** request_id 생성 */
        fun generateRequestId(): String = UUID.randomUUID().toString()
    }

    /** HTTP 에러 응답 파싱 */
    suspend fun parseErrorResponse(response: HttpResponse): ParsedError {
        val statusCode = response.status.value
        val requestId = response.headers["X-Request-ID"]

        return try {
            val body = response.bodyAsText()
            val errorResponse = objectMapper.readValue(body, LlmErrorResponse::class.java)

            ParsedError(
                statusCode = statusCode,
                errorCode = errorResponse.errorCode,
                errorType = errorResponse.errorType,
                message = errorResponse.message,
                requestId = errorResponse.requestId ?: requestId,
            )
        } catch (e: JacksonException) {
            // JSON 파싱 실패 시 기본 에러
            log.debug("ERROR_RESPONSE_PARSE_FAILED statusCode={}, error={}", statusCode, e.message)
            ParsedError(
                statusCode = statusCode,
                errorCode = null,
                errorType = null,
                message = "HTTP $statusCode",
                requestId = requestId,
            )
        }
    }

    /** Exception에서 ParsedError 생성 (UDS/네트워크 에러 분류) */
    fun fromException(
        e: Exception,
        requestId: String?,
    ): ParsedError {
        val (errorCode, message) = classifyException(e)
        return ParsedError(
            statusCode = 0,
            errorCode = errorCode,
            errorType = e::class.simpleName,
            message = message,
            requestId = requestId,
        )
    }

    /** 예외 타입별 에러 코드/메시지 분류 */
    private fun classifyException(e: Exception): Pair<String, String> =
        when (e) {
            is ConnectException -> "CONNECTION_FAILED" to "소켓 연결 실패: ${e.message}"
            is HttpRequestTimeoutException -> "TIMEOUT" to "요청 타임아웃"
            is SocketTimeoutException -> "SOCKET_TIMEOUT" to "소켓 타임아웃: ${e.message}"
            is SocketException -> "SOCKET_ERROR" to "소켓 에러: ${e.message}"
            is IOException -> "IO_ERROR" to "I/O 에러: ${e.message}"
            else -> "CLIENT_ERROR" to (e.message ?: "Unknown error")
        }

    /** 에러 로깅 */
    fun logError(
        operation: String,
        error: ParsedError,
    ) {
        log.error(
            "REST_{}_ERROR {}",
            operation,
            error.toLogString(),
        )
    }
}
