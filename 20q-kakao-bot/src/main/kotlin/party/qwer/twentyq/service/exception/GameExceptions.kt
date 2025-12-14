package party.qwer.twentyq.service.exception

sealed class GameException(
    message: String,
    cause: Throwable? = null,
) : RuntimeException(message, cause) {
    init {
        cause?.let { initCause(it) }
    }
}

class SessionNotFoundException(
    message: String? = null,
) : GameException(message ?: "")

class PermissionDeniedException(
    message: String? = null,
) : GameException(message ?: "")

class GameAlreadyExistsException(
    message: String? = null,
) : GameException(message ?: "")

class HintLimitExceededException(
    val maxHints: Int? = null,
    val hintCount: Int? = null,
    val remaining: Int? = null,
    message: String? = null,
) : GameException(message ?: "")

class InvalidQuestionException(
    message: String? = null,
) : GameException(message ?: "")

class DuplicateQuestionException : GameException("DUPLICATE_QUESTION")

class UnknownCommandException(
    message: String? = null,
) : GameException(message ?: "")

class TimeoutException(
    val operationName: String? = null,
    timeoutMillis: Long? = null,
    message: String? = null,
    cause: Throwable? = null,
) : GameException(
        message ?: when {
            operationName != null && timeoutMillis != null -> "$operationName 작업이 ${timeoutMillis}ms 내에 완료되지 않았습니다"
            operationName != null -> "$operationName 작업이 시간 초과되었습니다"
            else -> "작업이 시간 초과되었습니다"
        },
        cause,
    )

class CacheFullException(
    val maxSize: Long,
    message: String? = null,
) : GameException(message ?: "Cache full: maxSize=$maxSize")

class SessionNotInitializedException(
    val chatId: String,
) : GameException("Session not initialized for chatId: $chatId. Call ensureSession() first.")

class ServiceInitializationException(
    message: String,
    cause: Throwable? = null,
) : GameException(message, cause)
