package party.qwer.twentyq.rest

import io.ktor.client.HttpClient
import io.ktor.client.call.body
import io.ktor.client.request.header
import io.ktor.client.request.post
import io.ktor.client.request.setBody
import org.springframework.stereotype.Service
import party.qwer.twentyq.mcp.GuardEvaluation
import party.qwer.twentyq.rest.dto.GuardEvaluateResponse
import party.qwer.twentyq.rest.dto.GuardMaliciousResponse
import party.qwer.twentyq.rest.dto.GuardRequest

/** Guard REST API 클라이언트 */
@Service
class GuardRestClient(
    private val llmHttpClient: HttpClient,
    private val restErrorHandler: RestErrorHandler,
) {
    companion object {
        private const val EVALUATIONS_PATH = "/api/guard/evaluations"
        private const val CHECKS_PATH = "/api/guard/checks"
    }

    /** Guard 평가 */
    suspend fun evaluateGuard(input: String): GuardEvaluation {
        val requestId = RestErrorHandler.generateRequestId()
        return restErrorHandler.executeRestCall(
            operation = "GUARD_EVALUATE",
            requestId = requestId,
            request = {
                llmHttpClient.post(EVALUATIONS_PATH) {
                    header("X-Request-ID", requestId)
                    setBody(GuardRequest(inputText = input))
                }
            },
            onSuccess = { response ->
                val result = response.body<GuardEvaluateResponse>()
                GuardEvaluation(
                    isMalicious = result.malicious,
                    score = result.score,
                    reason = null,
                    categories = result.hits.map { it.id },
                )
            },
            onFailure = { GuardEvaluation(isMalicious = false, score = 0.0) },
        )
    }

    /** 악성 여부 체크 */
    suspend fun isMalicious(input: String): Boolean {
        val requestId = RestErrorHandler.generateRequestId()
        return restErrorHandler.executeRestCall(
            operation = "GUARD_IS_MALICIOUS",
            requestId = requestId,
            request = {
                llmHttpClient.post(CHECKS_PATH) {
                    header("X-Request-ID", requestId)
                    setBody(GuardRequest(inputText = input))
                }
            },
            onSuccess = { response -> response.body<GuardMaliciousResponse>().malicious },
            onFailure = { false },
        )
    }
}
