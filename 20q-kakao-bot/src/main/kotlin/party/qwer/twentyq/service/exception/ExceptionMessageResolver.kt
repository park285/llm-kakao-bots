package party.qwer.twentyq.service.exception

import kotlinx.coroutines.TimeoutCancellationException
import party.qwer.twentyq.util.game.GameMessageProvider

object ExceptionMessageResolver {
    private const val MSG_KEY_GENERIC_ERROR = "error.generic_error"
    private const val MSG_KEY_DUPLICATE_QUESTION = "error.invalid_question.duplicate_question"

    fun resolve(
        ex: Exception,
        messageProvider: GameMessageProvider,
    ): String =
        resolveCommand(ex, messageProvider)
            ?: resolveSession(ex, messageProvider)
            ?: resolvePermission(ex, messageProvider)
            ?: resolveGameLifecycle(ex, messageProvider)
            ?: resolveTimeout(ex, messageProvider)
            ?: resolveHint(ex, messageProvider)
            ?: resolveQuestion(ex, messageProvider)
            ?: resolveGameException(ex, messageProvider)
            ?: messageProvider.get(MSG_KEY_GENERIC_ERROR)

    private fun getHintLimitMessage(
        ex: HintLimitExceededException,
        messageProvider: GameMessageProvider,
    ): String =
        messageProvider.get(
            GameMessageKeys.HINT_LIMIT_EXCEEDED,
            "maxHints" to (ex.maxHints ?: ""),
            "hintCount" to (ex.hintCount ?: ""),
            "remaining" to (ex.remaining ?: ""),
        )

    private fun resolveCommand(
        ex: Exception,
        messageProvider: GameMessageProvider,
    ): String? =
        if (ex is UnknownCommandException) {
            messageProvider.get(GameMessageKeys.UNKNOWN_COMMAND)
        } else {
            null
        }

    private fun resolveSession(
        ex: Exception,
        messageProvider: GameMessageProvider,
    ): String? =
        if (ex is SessionNotFoundException) {
            messageProvider.get(GameMessageKeys.NO_SESSION)
        } else {
            null
        }

    private fun resolvePermission(
        ex: Exception,
        messageProvider: GameMessageProvider,
    ): String? =
        if (ex is PermissionDeniedException) {
            messageProvider.get(GameMessageKeys.NO_PERMISSION)
        } else {
            null
        }

    private fun resolveGameLifecycle(
        ex: Exception,
        messageProvider: GameMessageProvider,
    ): String? =
        if (ex is GameAlreadyExistsException) {
            messageProvider.get(GameMessageKeys.SESSION_ALREADY_EXISTS)
        } else {
            null
        }

    private fun resolveTimeout(
        ex: Exception,
        messageProvider: GameMessageProvider,
    ): String? =
        when (ex) {
            is TimeoutException -> ex.message ?: messageProvider.get("error.ai_timeout")
            is TimeoutCancellationException -> messageProvider.get("error.ai_timeout")
            else -> null
        }

    private fun resolveHint(
        ex: Exception,
        messageProvider: GameMessageProvider,
    ): String? =
        if (ex is HintLimitExceededException) {
            getHintLimitMessage(ex, messageProvider)
        } else {
            null
        }

    private fun resolveQuestion(
        ex: Exception,
        messageProvider: GameMessageProvider,
    ): String? =
        when (ex) {
            is DuplicateQuestionException -> messageProvider.get(MSG_KEY_DUPLICATE_QUESTION)
            is InvalidQuestionException -> ex.message?.ifBlank { null } ?: messageProvider.getInvalidQuestionMessage()
            else -> null
        }

    private fun resolveGameException(
        ex: Exception,
        messageProvider: GameMessageProvider,
    ): String? =
        if (ex is GameException) {
            ex.message?.ifBlank { null } ?: messageProvider.get(MSG_KEY_GENERIC_ERROR)
        } else {
            null
        }
}
