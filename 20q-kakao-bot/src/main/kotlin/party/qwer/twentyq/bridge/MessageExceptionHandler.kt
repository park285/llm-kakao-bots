package party.qwer.twentyq.bridge

import kotlinx.coroutines.TimeoutCancellationException
import org.slf4j.LoggerFactory
import org.springframework.stereotype.Component
import party.qwer.twentyq.logging.LoggingExtensions.sampled
import party.qwer.twentyq.service.exception.DuplicateQuestionException
import party.qwer.twentyq.service.exception.ExceptionMessageResolver
import party.qwer.twentyq.service.exception.GameAlreadyExistsException
import party.qwer.twentyq.service.exception.GameException
import party.qwer.twentyq.service.exception.HintLimitExceededException
import party.qwer.twentyq.service.exception.InvalidQuestionException
import party.qwer.twentyq.service.exception.PermissionDeniedException
import party.qwer.twentyq.service.exception.SessionNotFoundException
import party.qwer.twentyq.service.exception.TimeoutException
import party.qwer.twentyq.util.game.GameMessageProvider

/** 메시지 처리 예외 핸들러 */
@Component
class MessageExceptionHandler(
    private val messageProvider: GameMessageProvider,
) {
    companion object {
        private val log = LoggerFactory.getLogger(MessageExceptionHandler::class.java)
    }

    fun getErrorMessage(
        ex: Exception,
        chatId: String,
        userId: String,
    ): String {
        logException(ex, chatId, userId, "MESSAGE")
        return getMessageForException(ex)
    }

    fun getMessageForException(ex: Exception): String = ExceptionMessageResolver.resolve(ex, messageProvider)

    private fun logSampledWarn(
        key: String,
        chatId: String,
        userId: String,
        message: String,
    ) {
        log.sampled(key = "exception.$key", limit = 10, windowMillis = 5_000) {
            it.warn("{} chatId={}, userId={}", message, chatId, userId)
        }
    }

    private fun logSampledError(
        key: String,
        chatId: String,
        userId: String,
        message: String,
        ex: Exception,
    ) {
        log.sampled(key = "exception.$key", limit = 10, windowMillis = 5_000) {
            it.error("{} chatId={}, userId={}, error={}", message, chatId, userId, ex.message, ex)
        }
    }

    fun logException(
        ex: Exception,
        chatId: String,
        userId: String,
        context: String,
    ) {
        when (ex) {
            is TimeoutException,
            is TimeoutCancellationException,
            is HintLimitExceededException,
            is SessionNotFoundException,
            is PermissionDeniedException,
            is GameAlreadyExistsException,
            is DuplicateQuestionException,
            is InvalidQuestionException,
            -> logWarnException(ex, chatId, userId, context)

            is GameException,
            is IllegalArgumentException,
            is IllegalStateException,
            -> logSampledError("game.error", chatId, userId, "${context}_FAILED", ex)

            else -> logSampledError("unexpected", chatId, userId, "${context}_UNEXPECTED", ex)
        }
    }

    private fun logWarnException(
        ex: Exception,
        chatId: String,
        userId: String,
        context: String,
    ) {
        when (ex) {
            is TimeoutException ->
                logSampledWarn("timeout", chatId, userId, "${context}_TIMEOUT operation=${ex.operationName}")
            is TimeoutCancellationException ->
                logSampledWarn("timeout", chatId, userId, "${context}_AI_TIMEOUT")
            is HintLimitExceededException ->
                logSampledWarn("hint.limit", chatId, userId, "${context}_HINT_LIMIT remaining=${ex.remaining}")
            is SessionNotFoundException ->
                logSampledWarn("no.session", chatId, userId, "${context}_NO_SESSION")
            is PermissionDeniedException ->
                logSampledWarn("permission", chatId, userId, "${context}_PERMISSION_DENIED")
            is GameAlreadyExistsException ->
                logSampledWarn("game.resume", chatId, userId, "${context}_GAME_RESUME message=${ex.message}")
            is DuplicateQuestionException ->
                logSampledWarn("duplicate", chatId, userId, "${context}_DUPLICATE message=${ex.message}")
            is InvalidQuestionException ->
                logSampledWarn(
                    "invalid.question",
                    chatId,
                    userId,
                    "${context}_INVALID_QUESTION message=${ex.message}",
                )
        }
    }
}
