package party.qwer.twentyq.service.riddle

import org.slf4j.LoggerFactory
import org.springframework.stereotype.Service
import party.qwer.twentyq.model.FiveScaleKo
import party.qwer.twentyq.model.SecretForHint
import party.qwer.twentyq.rest.TwentyQRestClient
import party.qwer.twentyq.service.dto.LlmAnswerResponse
import party.qwer.twentyq.util.logging.LoggingConstants.LOG_TEXT_SHORT

/** LLM 기반 답변 검증 실행 서비스 */
@Service
class AnswerLlmExecutor(
    private val restClient: TwentyQRestClient,
) {
    companion object {
        private val log = LoggerFactory.getLogger(AnswerLlmExecutor::class.java)
    }

    suspend fun askScale(
        chatId: String,
        secret: SecretForHint,
        question: String,
    ): LlmAnswerResponse {
        log.debug(
            "LLM_CALL chatId={}, question='{}'",
            chatId,
            question.take(LOG_TEXT_SHORT),
        )

        // REST 서버에서 답변 처리 (TOON 변환, 프롬프트 구성 모두 서버에서 처리)
        val response =
            restClient.answerQuestion(
                chatId = chatId,
                target = secret.target,
                category = secret.category,
                question = question,
                details = secret.details,
            )

        if (response.isError) {
            log.error("LLM_ANSWER MCP_ERROR chatId={}, error={}", chatId, response.errorMessage)
            return LlmAnswerResponse(scale = null, thoughtSignature = null)
        }

        val scale = response.scale?.let { FiveScaleKo.fromText(it) }

        if (scale == null) {
            log.warn("LLM_ANSWER PARSE_FAILED chatId={}, raw='{}'", chatId, response.rawText)
        } else {
            log.debug("LLM_ANSWER SUCCESS chatId={}, scale={}", chatId, FiveScaleKo.token(scale))
        }

        return LlmAnswerResponse(
            scale = scale,
            thoughtSignature = response.thoughtSignature,
        )
    }
}
