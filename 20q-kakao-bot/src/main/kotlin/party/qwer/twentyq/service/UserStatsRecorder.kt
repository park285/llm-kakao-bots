package party.qwer.twentyq.service

import kotlinx.coroutines.CoroutineScope
import kotlinx.coroutines.Dispatchers
import kotlinx.coroutines.SupervisorJob
import kotlinx.coroutines.launch
import org.slf4j.LoggerFactory
import org.springframework.stereotype.Service
import org.springframework.transaction.reactive.TransactionalOperator
import party.qwer.twentyq.logging.debugL
import party.qwer.twentyq.model.GameLogEntity
import party.qwer.twentyq.model.UserStatsEntity
import party.qwer.twentyq.redis.session.UserStatsStore
import party.qwer.twentyq.repository.GameLogRepository
import party.qwer.twentyq.repository.UserNicknameMapRepository
import party.qwer.twentyq.repository.UserStatsRepository
import party.qwer.twentyq.util.common.executeWithOptimisticRetry
import party.qwer.twentyq.util.common.safeRedisOperation
import java.time.Instant

@Service
class UserStatsRecorder(
    private val userStatsRepository: UserStatsRepository,
    private val statsStore: UserStatsStore,
    private val gameLogRepository: GameLogRepository,
    private val userNicknameMapRepository: UserNicknameMapRepository,
    private val categoryStatsManager: CategoryStatsManager,
    private val transactionalOperator: TransactionalOperator,
) {
    companion object {
        private val log = LoggerFactory.getLogger(UserStatsRecorder::class.java)
    }

    private val statsScope = CoroutineScope(Dispatchers.IO + SupervisorJob())

    fun recordGameCompletion(
        chatId: String,
        userId: String,
        sender: String = "",
        category: String,
        questionCount: Int,
        hintCount: Int,
        wrongGuessCount: Int = 0,
        result: GameResult,
        target: String? = null,
        totalGameQuestionCount: Int = questionCount,
    ) {
        val p =
            GameCompletionParams(
                chatId,
                userId,
                sender,
                category,
                questionCount,
                hintCount,
                wrongGuessCount,
                result,
                target,
                totalGameQuestionCount,
            )
        statsScope.launch {
            logRecordStart(chatId, userId, category, result)
            executeRecordTransaction(p)
        }
    }

    private suspend fun executeRecordTransaction(p: GameCompletionParams) {
        transactionalOperator
            .executeWithOptimisticRetry(logTag = "STATS_RECORD_TX", log = log) {
                val updated = updateOrCreateStats(p)
                userStatsRepository.save(updated)
                saveNicknameMapping(p)
                saveGameLogEntity(p)
                updated
            }.onSuccess { handleRecordSuccess(p.chatId, p.userId, it) }
            .onFailure { handleRecordFailure(p.chatId, p.userId, it) }
    }

    private suspend fun handleRecordSuccess(
        chatId: String,
        userId: String,
        updated: UserStatsEntity,
    ) {
        writeToCache(chatId, userId, updated)
        logRecordSuccess(userId, updated.bestScoreQuestionCount)
    }

    private fun handleRecordFailure(
        chatId: String,
        userId: String,
        e: Throwable,
    ) {
        log.error("STATS_RECORD_FAILED chatId={}, userId={}, error={}", chatId, userId, e.message, e)
    }

    fun recordGameStart(
        chatId: String,
        userId: String,
    ) {
        statsScope.launch {
            log.info("STATS_GAME_START chatId={}, userId={}", chatId, userId)
            transactionalOperator
                .executeWithOptimisticRetry(logTag = "STATS_GAME_START_TX", log = log) {
                    val existing = userStatsRepository.findById(UserStatsCalculator.compositeId(chatId, userId))
                    val updated =
                        existing?.copy(
                            totalGamesStarted = existing.totalGamesStarted + 1,
                            updatedAt = Instant.now(),
                        ) ?: UserStatsEntity(
                            id = UserStatsCalculator.compositeId(chatId, userId),
                            chatId = chatId,
                            userId = userId,
                            totalGamesStarted = 1,
                            totalGamesCompleted = 0,
                            createdAt = Instant.now(),
                            updatedAt = Instant.now(),
                        ).markAsNew()
                    userStatsRepository.save(updated)
                    updated
                }.onSuccess {
                    writeToCache(chatId, userId, it)
                    log.info("STATS_GAME_START_SUCCESS userId={}, totalStarted={}", userId, it.totalGamesStarted)
                }.onFailure {
                    log.error("STATS_GAME_START_FAILED chatId={}, userId={}, error={}", chatId, userId, it.message, it)
                }
        }
    }

    private suspend fun writeToCache(
        chatId: String,
        userId: String,
        entity: UserStatsEntity,
    ) {
        safeRedisOperation("STATS_CACHE_WRITE_FAILED", log, Unit, "chatId" to chatId, "userId" to userId) {
            val categoryStats = UserStatsCalculator.parseCategoryStatsJson(entity.categoryStatsJson)
            val stats = entity.toUserStats(categoryStats)
            statsStore.set(chatId, userId, stats)
            log.debugL { "STATS_CACHE_WRITE chatId=$chatId, userId=$userId" }
        }
    }

    private fun logRecordStart(
        chatId: String,
        userId: String,
        category: String,
        result: GameResult,
    ) {
        log.info("STATS_RECORD_START chatId={}, userId={}, category={}, result={}", chatId, userId, category, result)
    }

    private fun logRecordSuccess(
        userId: String,
        bestScore: Int?,
    ) {
        log.info("STATS_RECORD_SUCCESS userId={}, bestScore={}", userId, bestScore)
    }

    private suspend fun updateOrCreateStats(p: GameCompletionParams): UserStatsEntity {
        val existing = userStatsRepository.findById(UserStatsCalculator.compositeId(p.chatId, p.userId))
        return if (existing != null) {
            UserStatsFactory.updateExistingStats(
                existing,
                p.category,
                p.questionCount,
                p.hintCount,
                p.wrongGuessCount,
                p.result,
                p.target,
                categoryStatsManager,
                p.totalGameQuestionCount,
            )
        } else {
            UserStatsFactory.createNewStats(
                p.chatId,
                p.userId,
                p.category,
                p.questionCount,
                p.hintCount,
                p.wrongGuessCount,
                p.result,
                p.target,
                categoryStatsManager,
                p.totalGameQuestionCount,
            )
        }
    }

    private suspend fun saveGameLogEntity(p: GameCompletionParams) {
        val resolvedSender = p.sender.ifBlank { gameLogRepository.findLatestSender(p.chatId, p.userId) ?: "" }
        val gameLog =
            GameLogEntity(
                chatId = p.chatId,
                userId = p.userId,
                sender = resolvedSender,
                category = p.category,
                questionCount = p.questionCount,
                hintCount = p.hintCount,
                wrongGuessCount = p.wrongGuessCount,
                result = p.result.name,
                target = p.target,
            )
        gameLogRepository.save(gameLog)
    }

    private suspend fun saveNicknameMapping(p: GameCompletionParams) {
        if (p.sender.isBlank()) {
            return
        }
        userNicknameMapRepository.upsertNickname(
            chatId = p.chatId,
            userId = p.userId,
            lastSender = p.sender,
            lastSeenAt = Instant.now(),
        )
    }

    private data class GameCompletionParams(
        val chatId: String,
        val userId: String,
        val sender: String,
        val category: String,
        val questionCount: Int,
        val hintCount: Int,
        val wrongGuessCount: Int,
        val result: GameResult,
        val target: String?,
        val totalGameQuestionCount: Int,
    )
}
