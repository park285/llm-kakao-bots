package party.qwer.twentyq.service.riddle

import org.slf4j.LoggerFactory
import org.springframework.stereotype.Service
import party.qwer.twentyq.api.dto.QuestionHistory
import party.qwer.twentyq.security.McpInjectionGuard
import party.qwer.twentyq.service.exception.DuplicateQuestionException
import party.qwer.twentyq.service.exception.InvalidQuestionException
import party.qwer.twentyq.service.exception.UnknownCommandException
import party.qwer.twentyq.util.logging.LoggingConstants.LOG_TEXT_LONG
import party.qwer.twentyq.util.logging.LoggingConstants.LOG_TEXT_MEDIUM
import party.qwer.twentyq.util.logging.LoggingConstants.LOG_TEXT_SHORT

@Service
class AnswerGuardService(
    private val injectionGuard: McpInjectionGuard,
    private val metaQuestionValidator: MetaQuestionValidator,
    private val questionPolicy: QuestionPolicy,
    private val answerValidator: AnswerValidator,
) {
    companion object {
        private val log = LoggerFactory.getLogger(AnswerGuardService::class.java)
    }

    suspend fun normalizeAndCheckGuard(
        chatId: String,
        question: String,
    ): String {
        if (injectionGuard.isMalicious(question)) {
            log.warn("GUARD_BLOCKED_ORIGINAL chatId={}, question='{}'", chatId, question.take(LOG_TEXT_LONG))
            throw UnknownCommandException()
        }

        val normalized = answerValidator.normalize(question)
        if (normalized != question) {
            log.info(
                "NORMALIZED chatId={}, original='{}', normalized='{}'",
                chatId,
                question.take(LOG_TEXT_SHORT),
                normalized.take(LOG_TEXT_SHORT),
            )
        }

        if (normalized != question && injectionGuard.isMalicious(normalized)) {
            log.warn("GUARD_BLOCKED_NORMALIZED chatId={}, question='{}'", chatId, normalized.take(LOG_TEXT_LONG))
            throw UnknownCommandException()
        }

        return normalized
    }

    suspend fun validateNotMetaQuestion(
        chatId: String,
        question: String,
    ) {
        val violation =
            when {
                questionPolicy.isAnswerLengthMetaQuestion(question) ->
                    "INVALID_LENGTH_META" to InvalidQuestionException()
                questionPolicy.isAnswerIndexMetaQuestion(question) ->
                    "INVALID_INDEX_META" to InvalidQuestionException()
                questionPolicy.isAnswerBoundaryMetaQuestion(question) ->
                    "INVALID_BOUNDARY_META" to InvalidQuestionException()
                metaQuestionValidator.shouldValidate(question) &&
                    metaQuestionValidator.isMetaQuestion(question) ->
                    "LLM_META_BLOCKED" to UnknownCommandException()
                else -> null
            } ?: return

        log.info("{} chatId={}, question='{}'", violation.first, chatId, question.take(LOG_TEXT_SHORT))
        throw violation.second
    }

    fun ensureNotDuplicate(
        chatId: String,
        question: String,
        questionNumber: Int,
        history: List<QuestionHistory>,
    ) {
        val normalizedEq = answerValidator.normalizeForEquality(question)
        val isDuplicateEq =
            history.any {
                it.questionNumber > 0 &&
                    answerValidator.normalizeForEquality(it.question) == normalizedEq
            }
        if (isDuplicateEq) {
            log.info(
                "DUP_QUESTION_BLOCKED chatId={}, questionNumber={}, normalized='{}'",
                chatId,
                questionNumber,
                normalizedEq.take(LOG_TEXT_MEDIUM),
            )
            duplicate()
        }
    }

    private fun duplicate(): Nothing = throw DuplicateQuestionException()
}
