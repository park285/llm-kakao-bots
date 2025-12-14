package party.qwer.twentyq.service.riddle

import org.slf4j.LoggerFactory
import org.springframework.stereotype.Component
import party.qwer.twentyq.api.dto.QuestionHistory
import party.qwer.twentyq.logging.LoggingExtensions
import party.qwer.twentyq.model.RiddleCategory
import party.qwer.twentyq.model.RiddleSecret
import party.qwer.twentyq.redis.RiddleSessionRepository
import party.qwer.twentyq.redis.tracking.PlayerSetStore
import party.qwer.twentyq.redis.tracking.WrongGuessSetStore
import party.qwer.twentyq.rest.TwentyQRestClient
import party.qwer.twentyq.service.GameResult
import party.qwer.twentyq.service.GameSessionRecorder
import party.qwer.twentyq.service.UserStatsService

/** 정답 성공 후속 처리 핸들러 */
@Component
class AnswerSuccessHandler(
    private val sessionRepo: RiddleSessionRepository,
    private val answerSuccessConfig: AnswerSuccessConfig,
    private val userStatsService: UserStatsService,
    private val wrongGuessSetStore: WrongGuessSetStore,
    private val playerSetStore: PlayerSetStore,
    private val gameSessionRecorder: GameSessionRecorder,
    private val llmRestClient: TwentyQRestClient,
) {
    private val topicHistoryStore = answerSuccessConfig.topicHistoryStore
    private val hintCountStore = answerSuccessConfig.hintCountStore
    private val gameMessageProvider = answerSuccessConfig.gameMessageProvider

    companion object {
        private val log = LoggerFactory.getLogger(AnswerSuccessHandler::class.java)
    }

    suspend fun handleSuccess(
        chatId: String,
        secret: RiddleSecret,
        userId: String,
    ): String {
        val target = secret.target
        val history = sessionRepo.getHistory(chatId)
        val questionCount = history.count { it.questionNumber > 0 }
        val hintCount = hintCountStore.getAsync(chatId)

        val successMessage = buildSuccessMessage(chatId, userId, target, history, hintCount, questionCount)
        recordStatsForCorrectAnswer(chatId, secret, hintCount, target, userId, questionCount)
        cleanupSession(chatId, secret, target, questionCount)

        return successMessage
    }

    private suspend fun buildSuccessMessage(
        chatId: String,
        userId: String,
        target: String,
        history: List<QuestionHistory>,
        hintCount: Int,
        questionCount: Int,
    ): String {
        val hints = history.filter { it.questionNumber < 0 }.sortedBy { it.questionNumber }
        val maxHints = HintPolicy.computeMaxHints(hintCount)
        val hintBlock = buildHintBlock(hints)
        val wrongGuessBlock = buildWrongGuessBlock(chatId, userId)

        return gameMessageProvider
            .get(
                "answer.success",
                "target" to target,
                "questionCount" to questionCount,
                "hintCount" to hintCount,
                "maxHints" to maxHints,
                "wrongGuessBlock" to wrongGuessBlock,
                "hintBlock" to hintBlock,
            ).also {
                log.info(
                    "SUCCESS_MESSAGE_GENERATED chatId={}, target='{}', qCount={}, hintCount={}, msgLen={}",
                    chatId,
                    target,
                    questionCount,
                    hintCount,
                    it.length,
                )
            }
    }

    private suspend fun cleanupSession(
        chatId: String,
        secret: RiddleSecret,
        target: String,
        questionCount: Int,
    ) {
        val category = RiddleCategory.fromString(secret.category).name
        topicHistoryStore.addAsync(chatId, category, target)

        sessionRepo.delete(chatId)

        runCatching {
            llmRestClient.endSessionByChat(chatId)
        }.onFailure { ex ->
            log.warn("LLM_SESSION_END_FAILED chatId={}, error={}", chatId, ex.message)
        }

        LoggingExtensions.run {
            log.sampled(key = "riddle.answer.success", limit = 5, windowMillis = 60_000) { logger ->
                logger.info("CORRECT_GUESS chatId={}, questionCount={}, sessionDeleted=true", chatId, questionCount)
            }
        }
    }

    private fun buildHintBlock(hints: List<QuestionHistory>): String =
        if (hints.isNotEmpty()) {
            val first = hints.first()
            val hintList =
                listOf(
                    gameMessageProvider.get("answer.hint_item", "question" to "", "answer" to first.answer),
                ).joinToString("\n")
            gameMessageProvider.get("answer.hint_section_used", "hintCount" to 1, "hintList" to hintList)
        } else {
            gameMessageProvider.get("answer.hint_section_none")
        }

    private suspend fun buildWrongGuessBlock(
        chatId: String,
        userId: String,
    ): String {
        val wrongGuesses = wrongGuessSetStore.getUserWrongGuessesAsync(chatId, userId)
        return if (wrongGuesses.isNotEmpty()) {
            gameMessageProvider.get("answer.wrong_guess_section", "wrongGuesses" to wrongGuesses.joinToString(", "))
        } else {
            ""
        }
    }

    private suspend fun recordStatsForCorrectAnswer(
        chatId: String,
        secret: RiddleSecret,
        hintCount: Int,
        target: String,
        answererId: String,
        totalGameQuestionCount: Int,
    ) {
        val playerIds = sessionRepo.getAllPlayerIds(chatId)
        if (playerIds.isEmpty()) {
            log.warn("STATS_SKIP_NO_PLAYERS chatId={}", chatId)
            return
        }

        val category = RiddleCategory.fromString(secret.category).name
        val playerInfos = playerSetStore.getAllAsync(chatId)
        val userSenderMap = playerInfos.associate { it.userId to it.sender }
        val userQuestionCounts = buildUserQuestionCounts(chatId)

        playerIds.forEach { userId ->
            val userQuestionCount = userQuestionCounts[userId] ?: 0
            val wrongGuessCount = wrongGuessSetStore.getUserWrongGuessCountAsync(chatId, userId)
            val sender = userSenderMap[userId] ?: ""
            val targetForPlayer = if (userId == answererId) target else null

            recordPlayerStats(
                chatId,
                userId,
                sender,
                category,
                userQuestionCount,
                hintCount,
                wrongGuessCount,
                targetForPlayer,
                totalGameQuestionCount,
            )
        }

        gameSessionRecorder.recordSession(
            sessionId = null,
            chatId = chatId,
            category = category,
            result = GameResult.CORRECT,
            participantCount = playerIds.size,
            questionCount = totalGameQuestionCount,
            hintCount = hintCount,
        )
        log.info("STATS_BATCH_COMPLETE chatId={}, playerCount={}", chatId, playerIds.size)
    }

    private suspend fun buildUserQuestionCounts(chatId: String): Map<String, Int> {
        val history = sessionRepo.getHistory(chatId)
        return history
            .filter { it.questionNumber > 0 && it.userId != null }
            .groupBy { it.userId!! }
            .mapValues { (_, histories) -> histories.size }
    }

    private suspend fun recordPlayerStats(
        chatId: String,
        userId: String,
        sender: String,
        category: String,
        questionCount: Int,
        hintCount: Int,
        wrongGuessCount: Int,
        target: String?,
        totalGameQuestionCount: Int,
    ) {
        userStatsService.recordGameCompletion(
            chatId = chatId,
            userId = userId,
            sender = sender,
            category = category,
            questionCount = questionCount,
            hintCount = hintCount,
            wrongGuessCount = wrongGuessCount,
            result = GameResult.CORRECT,
            target = target,
            totalGameQuestionCount = totalGameQuestionCount,
        )

        log.info(
            "STATS_RECORDED chatId={}, userId={}, sender={}, questions={}, wrongGuesses={}, result=CORRECT",
            chatId,
            userId,
            sender,
            questionCount,
            wrongGuessCount,
        )
    }
}
