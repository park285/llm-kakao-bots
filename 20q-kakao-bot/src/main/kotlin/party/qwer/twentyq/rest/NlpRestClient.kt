package party.qwer.twentyq.rest

import io.ktor.client.HttpClient
import io.ktor.client.call.body
import io.ktor.client.request.header
import io.ktor.client.request.post
import io.ktor.client.request.setBody
import org.springframework.stereotype.Service
import party.qwer.twentyq.mcp.NlpAnalysis
import party.qwer.twentyq.rest.dto.NlpAnomalyResponse
import party.qwer.twentyq.rest.dto.NlpHeuristicsResponse
import party.qwer.twentyq.rest.dto.NlpRequest
import party.qwer.twentyq.rest.dto.NlpToken

/** NLP REST API 클라이언트 */
@Service
class NlpRestClient(
    private val llmHttpClient: HttpClient,
    private val restErrorHandler: RestErrorHandler,
) {
    companion object {
        private const val ANALYSES_PATH = "/api/nlp/analyses"
        private const val ANOMALY_SCORES_PATH = "/api/nlp/anomaly-scores"
        private const val HEURISTICS_PATH = "/api/nlp/heuristics"
    }

    /** NLP 분석 */
    suspend fun analyzeNlp(text: String): NlpAnalysis {
        val requestId = RestErrorHandler.generateRequestId()
        return restErrorHandler.executeRestCall(
            operation = "NLP_ANALYZE",
            requestId = requestId,
            request = {
                llmHttpClient.post(ANALYSES_PATH) {
                    header("X-Request-ID", requestId)
                    setBody(NlpRequest(text = text))
                }
            },
            onSuccess = { response ->
                val tokens = response.body<List<NlpToken>>()
                NlpAnalysis(
                    tokens = tokens.map { it.form },
                    posTag = tokens.map { it.tag },
                    anomalyScore = 0.0,
                )
            },
            onFailure = { NlpAnalysis() },
        )
    }

    /** 이상치 점수 조회 */
    suspend fun getAnomalyScore(text: String): Double {
        val requestId = RestErrorHandler.generateRequestId()
        return restErrorHandler.executeRestCall(
            operation = "NLP_ANOMALY_SCORE",
            requestId = requestId,
            request = {
                llmHttpClient.post(ANOMALY_SCORES_PATH) {
                    header("X-Request-ID", requestId)
                    setBody(NlpRequest(text = text))
                }
            },
            onSuccess = { response -> response.body<NlpAnomalyResponse>().score },
            onFailure = { 0.0 },
        )
    }

    /** 휴리스틱 분석 */
    suspend fun analyzeHeuristics(text: String): NlpHeuristicsResponse {
        val requestId = RestErrorHandler.generateRequestId()
        return restErrorHandler.executeRestCall(
            operation = "NLP_HEURISTICS",
            requestId = requestId,
            request = {
                llmHttpClient.post(HEURISTICS_PATH) {
                    header("X-Request-ID", requestId)
                    setBody(NlpRequest(text = text))
                }
            },
            onSuccess = { response -> response.body<NlpHeuristicsResponse>() },
            onFailure = { NlpHeuristicsResponse() },
        )
    }
}
