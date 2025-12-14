package party.qwer.twentyq.service.riddle

import org.slf4j.LoggerFactory
import org.springframework.stereotype.Component
import party.qwer.twentyq.rest.TwentyQRestClient

/** 답변 유사도 검증 컴포넌트 */
@Component
class SimilarityVerifier(
    private val restClient: TwentyQRestClient,
) {
    companion object {
        private val log = LoggerFactory.getLogger(SimilarityVerifier::class.java)
    }

    /** LLM 기반 유사도 검증 */
    suspend fun verifyHighSimilarity(
        target: String,
        guess: String,
    ): String {
        log.info("LLM verification triggered: target='{}', guess='{}'", target, guess)
        return verifyWithMcp(target, guess)
    }

    private suspend fun verifyWithMcp(
        target: String,
        guess: String,
    ): String {
        val response =
            runCatching { restClient.verifyGuess(target, guess) }
                .onFailure { throwable ->
                    when (throwable) {
                        is kotlin.coroutines.cancellation.CancellationException -> throw throwable
                        is Error -> throw throwable
                    }
                    log.warn("LLM verification failed: {} - {}", throwable.javaClass.simpleName, throwable.message)
                }.getOrNull()
                ?: return VerifyAnswerResponse.RESPONSE_REJECT

        if (response.isError) {
            log.warn("LLM verification MCP_ERROR: {}", response.errorMessage)
            return VerifyAnswerResponse.RESPONSE_REJECT
        }

        val result = response.result?.uppercase()
        return when (result) {
            VerifyAnswerResponse.RESPONSE_ACCEPT,
            VerifyAnswerResponse.RESPONSE_REJECT,
            VerifyAnswerResponse.RESPONSE_CLOSE,
            -> {
                log.info("LLM verification result: target='{}', guess='{}', status={}", target, guess, result)
                result
            }
            else -> {
                log.warn(
                    "LLM verification parse failed: target='{}', guess='{}', raw='{}'",
                    target,
                    guess,
                    response.rawText,
                )
                VerifyAnswerResponse.RESPONSE_REJECT
            }
        }
    }
}
