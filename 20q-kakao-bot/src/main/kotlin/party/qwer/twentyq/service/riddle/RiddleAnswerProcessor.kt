package party.qwer.twentyq.service.riddle

import org.slf4j.LoggerFactory
import org.springframework.stereotype.Service
import party.qwer.twentyq.model.FiveScaleKo
import party.qwer.twentyq.model.RiddleSecret
import party.qwer.twentyq.model.SecretForHint
import party.qwer.twentyq.redis.RiddleSessionRepository
import party.qwer.twentyq.service.dto.AnswerResult
import party.qwer.twentyq.service.dto.AnswerSource
import party.qwer.twentyq.service.exception.InvalidQuestionException
import party.qwer.twentyq.service.exception.SessionNotFoundException
import party.qwer.twentyq.util.common.json.parseDescriptionOrNull
import party.qwer.twentyq.util.logging.LoggingConstants.LOG_TEXT_EXTRA_LONG
import party.qwer.twentyq.util.logging.LoggingConstants.LOG_TEXT_SHORT

/** 수수께끼 답변 검증 및 처리 서비스 */
@Service
class RiddleAnswerProcessor(
    private val sessionRepo: RiddleSessionRepository,
    private val verificationService: AnswerVerificationService,
    private val answerGuardService: AnswerGuardService,
    private val answerLlmExecutor: AnswerLlmExecutor,
    private val objectMapper: tools.jackson.databind.ObjectMapper,
) {
    companion object {
        private val log = LoggerFactory.getLogger(RiddleAnswerProcessor::class.java)
    }

    suspend fun processAnswer(
        chatId: String,
        question: String,
        userId: String,
        isChain: Boolean = false,
    ): AnswerResult {
        log.info("PROCESS_START chatId={}, question='{}', isChain={}", chatId, question.take(LOG_TEXT_SHORT), isChain)

        val secret = sessionRepo.getSecret(chatId = chatId) ?: throw SessionNotFoundException()

        val normalizedQuestion = answerGuardService.normalizeAndCheckGuard(chatId, question)

        val result =
            verificationService.checkAnswerAttempts(chatId, normalizedQuestion, secret, userId)
                ?: run {
                    val trimmedQuestion = normalizedQuestion.trim()
                    answerGuardService.validateNotMetaQuestion(chatId, trimmedQuestion)
                    handleRegularQuestion(chatId, normalizedQuestion, secret, userId, isChain)
                }

        return result
    }

    private suspend fun handleRegularQuestion(
        chatId: String,
        question: String,
        secret: RiddleSecret,
        userId: String,
        isChain: Boolean = false,
    ): AnswerResult {
        val history = sessionRepo.getHistory(chatId)
        val questionNumber = history.count { it.questionNumber > 0 } + 1

        answerGuardService.ensureNotDuplicate(chatId, question, questionNumber, history)
        log.info(
            "REGULAR_QUESTION chatId={}, questionNumber={}, historySize={}, isChain={}",
            chatId,
            questionNumber,
            history.size,
            isChain,
        )

        val secretForHint = buildSecretForHint(secret)

        val primaryResult =
            executePrimaryLlmAttempt(
                secretForHint,
                question,
                chatId,
                questionNumber,
                userId,
                isChain,
            )
        if (primaryResult != null) {
            return primaryResult
        }

        val (retryResult, signature) = executeRetryLlmAttempt(secretForHint, question, chatId)

        sessionRepo.addHistory(
            chatId,
            questionNumber,
            question,
            FiveScaleKo.token(retryResult.scale),
            isChain,
            signature,
            userId,
        )
        log.info("PROCESS_COMPLETE chatId={}, finalScale='{}'", chatId, FiveScaleKo.token(retryResult.scale))

        return retryResult
    }

    private fun buildSecretForHint(secret: RiddleSecret): SecretForHint {
        val details = objectMapper.parseDescriptionOrNull(secret.description)
        return SecretForHint(
            target = secret.target,
            category = secret.category,
            details = details,
        )
    }

    private suspend fun executePrimaryLlmAttempt(
        secret: SecretForHint,
        question: String,
        chatId: String,
        questionNumber: Int,
        userId: String,
        isChain: Boolean = false,
    ): AnswerResult? {
        val llmResponse = answerLlmExecutor.askScale(chatId, secret, question)
        val firstResult = llmResponse.scale ?: return null

        if (firstResult == FiveScaleKo.INVALID) {
            log.info("INVALID_QUESTION chatId={}, question='{}'", chatId, question.take(LOG_TEXT_SHORT))
            invalid()
        }

        log.info(
            "LLM_ANSWER chatId={}, question='{}', scale='{}', source=PRIMARY, isChain={}",
            chatId,
            question.take(LOG_TEXT_EXTRA_LONG),
            FiveScaleKo.token(firstResult),
            isChain,
        )

        sessionRepo.addHistory(
            chatId,
            questionNumber,
            question,
            FiveScaleKo.token(firstResult),
            isChain,
            llmResponse.thoughtSignature,
            userId,
        )
        return AnswerResult(
            scale = firstResult,
            source = AnswerSource.ENUM_SCHEMA_PRIMARY,
            guardDegraded = false,
        )
    }

    private suspend fun executeRetryLlmAttempt(
        secret: SecretForHint,
        question: String,
        chatId: String,
    ): Pair<AnswerResult, String?> {
        log.warn("LLM_RETRY chatId={}, reason=PRIMARY_FAILED", chatId)
        val candidates = FiveScaleKo.RETRY_CANDIDATES
        val strictQuestion = "$question\n\n(Respond with ONLY one of: $candidates)"
        val llmResponse = answerLlmExecutor.askScale(chatId, secret, strictQuestion)
        val strictResult = llmResponse.scale

        if (strictResult != null) {
            if (strictResult == FiveScaleKo.INVALID) {
                log.info("INVALID_QUESTION chatId={}, question='{}'", chatId, question.take(LOG_TEXT_SHORT))
                invalid()
            }

            log.info(
                "LLM_ANSWER chatId={}, question='{}', scale='{}', source=RETRY_STRICT",
                chatId,
                question.take(LOG_TEXT_EXTRA_LONG),
                FiveScaleKo.token(strictResult),
            )

            return AnswerResult(
                scale = strictResult,
                source = AnswerSource.ENUM_SCHEMA_RETRY_STRICT,
                guardDegraded = false,
            ) to llmResponse.thoughtSignature
        }

        log.warn("LLM_FALLBACK chatId={}, scale=ALWAYS_NO, source=FALLBACK_DEFAULT", chatId)
        return AnswerResult(
            scale = FiveScaleKo.ALWAYS_NO,
            source = AnswerSource.FALLBACK_DEFAULT,
            guardDegraded = false,
        ) to null
    }

    private fun invalid(): Nothing = throw InvalidQuestionException()
}
