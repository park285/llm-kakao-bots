package party.qwer.twentyq.service.riddle

import org.slf4j.LoggerFactory
import org.springframework.stereotype.Service
import party.qwer.twentyq.logging.LoggingExtensions
import party.qwer.twentyq.model.RiddleCategory
import party.qwer.twentyq.model.RiddleSecret
import party.qwer.twentyq.redis.RiddleSessionRepository
import party.qwer.twentyq.redis.tracking.PlayerSetStore
import party.qwer.twentyq.rest.TwentyQRestClient
import party.qwer.twentyq.service.GameResult
import party.qwer.twentyq.service.GameSessionRecorder
import party.qwer.twentyq.service.UserStatsService
import party.qwer.twentyq.service.exception.SessionNotFoundException
import party.qwer.twentyq.util.game.GameMessageProvider

/** 수수께끼 게임 종료 처리 서비스 */
@Service
class RiddleTerminator(
    private val sessionRepo: RiddleSessionRepository,
    private val messageProvider: GameMessageProvider,
    private val userStatsService: UserStatsService,
    private val playerSetStore: PlayerSetStore,
    private val gameSessionRecorder: GameSessionRecorder,
    private val llmRestClient: TwentyQRestClient,
) {
    companion object {
        private val log = LoggerFactory.getLogger(RiddleTerminator::class.java)
    }

    suspend fun surrender(chatId: String): String {
        log.info("surrender START chatId={}", chatId)
        val secret =
            sessionRepo.getSecret(chatId = chatId)
                ?: throw SessionNotFoundException()

        val selectedCategoryRaw = sessionRepo.getSelectedCategory(chatId)
        val selectedCategoryKo =
            selectedCategoryRaw?.let {
                val cat = RiddleCategory.fromString(it)
                if (cat == RiddleCategory.ANY) null else cat.koreanName
            }
        val categoryLine =
            selectedCategoryKo?.let { messageProvider.get("surrender.category_line", "category" to it) }
                ?: ""

        val hintBlock = buildHintBlock(chatId)

        val result =
            messageProvider.get(
                "surrender.result",
                "hintBlock" to hintBlock,
                "target" to secret.target,
                "categoryLine" to categoryLine,
            )

        recordStatsForGameEnd(chatId, secret, GameResult.SURRENDER)

        val category = RiddleCategory.fromString(secret.category).name
        sessionRepo.addCompletedTopic(chatId, category, secret.target)

        sessionRepo.delete(chatId)
        runCatching {
            llmRestClient.endSessionByChat(chatId)
        }.onFailure { ex ->
            log.warn("LLM_SESSION_END_FAILED chatId={}, error={}", chatId, ex.message)
        }
        LoggingExtensions.run {
            log.sampled(key = "riddle.surrender.success", limit = 5, windowMillis = 60_000) {
                it.info("surrender SUCCESS chatId={}, session deleted", chatId)
            }
        }
        return result
    }

    private suspend fun buildHintBlock(chatId: String): String {
        val history = sessionRepo.getHistory(chatId)
        val hints = history.filter { it.questionNumber < 0 }.sortedBy { it.questionNumber }
        return if (hints.isNotEmpty()) {
            val first = hints.first()
            val header = messageProvider.get("surrender.hint_block_header", "hintCount" to 1)
            val line =
                messageProvider.get(
                    "surrender.hint_item",
                    "hintNumber" to 1,
                    "content" to first.answer,
                )
            header + line
        } else {
            ""
        }
    }

    /**
     * 게임 종료 시 스탯 기록 (모든 참여자)
     */
    private suspend fun recordStatsForGameEnd(
        chatId: String,
        secret: RiddleSecret,
        result: GameResult,
    ) {
        val playerIds = sessionRepo.getAllPlayerIds(chatId)
        if (playerIds.isEmpty()) {
            log.warn("STATS_SKIP_NO_PLAYERS chatId={}", chatId)
            return
        }

        val history = sessionRepo.getHistory(chatId)
        val questionCount = history.count { it.questionNumber > 0 }
        val hintCount = history.count { it.questionNumber < 0 }
        val category = RiddleCategory.fromString(secret.category).name

        // sender 정보 조회 (AnswerSuccessHandler와 동일 패턴)
        val playerInfos = playerSetStore.getAllAsync(chatId)
        val userSenderMap = playerInfos.associate { it.userId to it.sender }

        playerIds.forEach { userId ->
            val sender = userSenderMap[userId] ?: ""
            userStatsService.recordGameCompletion(
                chatId = chatId,
                userId = userId,
                sender = sender,
                category = category,
                questionCount = questionCount,
                hintCount = hintCount,
                result = result,
                target = null,
                totalGameQuestionCount = questionCount,
            )
        }

        gameSessionRecorder.recordSession(
            sessionId = null,
            chatId = chatId,
            category = category,
            result = result,
            participantCount = playerIds.size,
            questionCount = questionCount,
            hintCount = hintCount,
        )
        log.info("STATS_RECORDED chatId={}, playerCount={}, result={}", chatId, playerIds.size, result)
    }
}
