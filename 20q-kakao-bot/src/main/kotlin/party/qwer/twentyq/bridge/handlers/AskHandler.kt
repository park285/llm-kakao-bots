package party.qwer.twentyq.bridge.handlers

import org.slf4j.LoggerFactory
import org.springframework.stereotype.Component
import party.qwer.twentyq.model.FiveScaleKo
import party.qwer.twentyq.service.RiddleService
import party.qwer.twentyq.service.dto.AnswerResult
import party.qwer.twentyq.service.exception.GameMessageKeys
import party.qwer.twentyq.util.common.formatting.UserIdFormatter
import party.qwer.twentyq.util.common.security.requireSessionOrThrow
import party.qwer.twentyq.util.game.GameMessageProvider

@Component
class AskHandler(
    private val riddleService: RiddleService,
    private val messageProvider: GameMessageProvider,
) {
    companion object {
        private val log = LoggerFactory.getLogger(AskHandler::class.java)
        private const val LOG_QUESTION_MAX_LENGTH = 50
    }

    suspend fun handle(
        chatId: String,
        question: String,
        userId: String,
        sender: String?,
        isChain: Boolean = false,
    ): String {
        log.info(
            "HANDLE_ASK chatId={}, question='{}', isChain={}",
            chatId,
            question.take(LOG_QUESTION_MAX_LENGTH),
            isChain,
        )
        riddleService.requireSessionOrThrow(chatId)
        val result = riddleService.answer(chatId, question, userId, isChain)
        return buildResponse(result, chatId, userId, sender)
    }

    private fun buildResponse(
        result: AnswerResult,
        chatId: String,
        userId: String,
        sender: String?,
    ): String =
        when {
            result.isCorrect -> handleCorrectAnswer(result, chatId)
            result.isCloseCall -> handleCloseCall(result, chatId)
            result.isWrongGuess -> handleWrongGuess(result, chatId, userId, sender)
            result.guardDegraded -> handleGuardDegraded(chatId)
            else ->
                FiveScaleKo
                    .token(result.scale)
        }

    private fun handleCorrectAnswer(
        result: AnswerResult,
        chatId: String,
    ): String {
        log.info("CORRECT_ANSWER chatId={}", chatId)
        val message = result.successMessage ?: messageProvider.get("answer.correct_default")
        log.info(
            "SUCCESS_RESPONSE chatId={}, hasSuccessMsg={}, msgLen={}",
            chatId,
            result.successMessage != null,
            message.length,
        )
        return message
    }

    private fun handleWrongGuess(
        result: AnswerResult,
        chatId: String,
        userId: String,
        sender: String?,
    ): String {
        log.info("WRONG_GUESS chatId={}, guess='{}'", chatId, result.guessedAnswer)
        val displayName = UserIdFormatter.displayName(userId, sender, chatId, messageProvider.get("user.anonymous"))
        return messageProvider.get(
            GameMessageKeys.ANSWER_WRONG_GUESS,
            "nickname" to displayName,
            "guess" to (result.guessedAnswer ?: ""),
        )
    }

    private fun handleGuardDegraded(chatId: String): String {
        log.info("GUARD_BLOCKED chatId={}", chatId)
        return messageProvider.get("error.invalid_question.default")
    }

    private fun handleCloseCall(
        result: AnswerResult,
        chatId: String,
    ): String {
        log.info("CLOSE_CALL chatId={}, guess='{}'", chatId, result.guessedAnswer)
        return messageProvider.get(GameMessageKeys.ANSWER_CLOSE_CALL)
    }
}
