package io.github.kapu.turtlesoup.mq

import io.github.kapu.turtlesoup.config.GameConstants
import io.github.kapu.turtlesoup.config.ValidationConstants
import io.github.kapu.turtlesoup.utils.AccessDeniedException
import io.github.kapu.turtlesoup.utils.ChatBlockedException
import io.github.kapu.turtlesoup.utils.GameAlreadySolvedException
import io.github.kapu.turtlesoup.utils.GameAlreadyStartedException
import io.github.kapu.turtlesoup.utils.InvalidQuestionException
import io.github.kapu.turtlesoup.utils.MaxHintsReachedException
import io.github.kapu.turtlesoup.utils.MessageKeys
import io.github.kapu.turtlesoup.utils.PuzzleGenerationException
import io.github.kapu.turtlesoup.utils.SessionNotFoundException
import io.github.kapu.turtlesoup.utils.TurtleSoupException
import io.github.kapu.turtlesoup.utils.UserBlockedException
import kotlin.reflect.KClass

/** 에러 매핑 결과 */
data class ErrorMapping(
    val key: String,
    val params: Array<Pair<String, Any>> = emptyArray(),
) {
    override fun equals(other: Any?): Boolean {
        if (this === other) return true
        if (other !is ErrorMapping) return false
        return key == other.key && params.contentEquals(other.params)
    }

    override fun hashCode(): Int = 31 * key.hashCode() + params.contentHashCode()
}

/** Exception -> MessageKey + Params mapping */
object ExceptionMapper {
    private val keyMapping: Map<KClass<out TurtleSoupException>, String> =
        mapOf(
            SessionNotFoundException::class to MessageKeys.ERROR_NO_SESSION,
            InvalidQuestionException::class to MessageKeys.ERROR_INVALID_QUESTION,
            MaxHintsReachedException::class to MessageKeys.ERROR_MAX_HINTS,
            GameAlreadyStartedException::class to MessageKeys.ERROR_GAME_ALREADY_STARTED,
            GameAlreadySolvedException::class to MessageKeys.ERROR_GAME_ALREADY_SOLVED,
            PuzzleGenerationException::class to MessageKeys.ERROR_PUZZLE_GENERATION,
            AccessDeniedException::class to MessageKeys.ERROR_ACCESS_DENIED,
            UserBlockedException::class to MessageKeys.ERROR_USER_BLOCKED,
            ChatBlockedException::class to MessageKeys.ERROR_CHAT_BLOCKED,
        )

    fun getErrorMapping(exception: TurtleSoupException): ErrorMapping {
        val key = keyMapping[exception::class] ?: MessageKeys.ERROR_INTERNAL
        val params = getParams(exception)
        return ErrorMapping(key, params)
    }

    private fun getParams(exception: TurtleSoupException): Array<Pair<String, Any>> =
        when (exception) {
            is InvalidQuestionException ->
                arrayOf(
                    "minLength" to ValidationConstants.MIN_QUESTION_LENGTH,
                    "maxLength" to ValidationConstants.MAX_QUESTION_LENGTH,
                )
            is MaxHintsReachedException ->
                arrayOf("maxHints" to GameConstants.MAX_HINTS)
            else -> emptyArray()
        }
}
