package party.qwer.twentyq.service.exception

sealed class PromptException(
    message: String,
    cause: Throwable? = null,
) : GameException(message, cause)

class PromptNotFoundException(
    val section: String,
    val key: String? = null,
) : PromptException(
        message =
            buildString {
                append("Prompt not found: section='$section'")
                if (key != null) append(", key='$key'")
            },
    )

class PromptLoadException(
    path: String,
    cause: Throwable,
) : PromptException(
        message = "Failed to load prompt from '$path': ${cause.message}",
        cause = cause,
    )
