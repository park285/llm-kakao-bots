package party.qwer.twentyq.rest

import io.ktor.client.plugins.ResponseException
import io.ktor.client.statement.HttpResponse
import io.ktor.http.isSuccess
import party.qwer.twentyq.rest.dto.ParsedError
import tools.jackson.core.JacksonException
import java.io.IOException
import kotlin.coroutines.cancellation.CancellationException

/**
 * REST 호출 공통 실행 헬퍼.
 * - 상태 코드 실패/예외를 ParsedError로 통일
 * - TooGenericExceptionCaught 회피를 위해 명시적 예외만 처리
 */
suspend fun <T> RestErrorHandler.executeRestCall(
    operation: String,
    requestId: String,
    request: suspend () -> HttpResponse,
    onSuccess: suspend (HttpResponse) -> T,
    onFailure: (ParsedError) -> T,
): T =
    try {
        val response = request()
        if (response.status.isSuccess()) {
            onSuccess(response)
        } else {
            val error = parseErrorResponse(response)
            logError(operation, error)
            onFailure(error)
        }
    } catch (e: CancellationException) {
        throw e
    } catch (e: ResponseException) {
        val error = fromException(e, requestId)
        logError(operation, error)
        onFailure(error)
    } catch (e: IOException) {
        val error = fromException(e, requestId)
        logError(operation, error)
        onFailure(error)
    } catch (e: JacksonException) {
        val error = fromException(e, requestId)
        logError(operation, error)
        onFailure(error)
    }
