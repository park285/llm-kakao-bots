package party.qwer.twentyq.service

import kotlinx.coroutines.flow.Flow
import kotlinx.coroutines.flow.collect
import org.slf4j.LoggerFactory
import org.springframework.stereotype.Service
import party.qwer.twentyq.model.GameSessionEntity
import party.qwer.twentyq.model.RoomStatsResult
import party.qwer.twentyq.model.UserStats
import party.qwer.twentyq.redis.session.UserStatsStore
import party.qwer.twentyq.repository.GameLogRepository
import party.qwer.twentyq.repository.GameSessionRepository
import party.qwer.twentyq.repository.UserStatsRepository
import party.qwer.twentyq.service.StatsPeriod
import party.qwer.twentyq.util.game.constants.ValidationConstants
import java.time.Instant

/**
 * 사용자 스탯 서비스
 */
@Service
class UserStatsService(
    private val userStatsRepository: UserStatsRepository,
    private val statsStore: UserStatsStore,
    private val gameLogRepository: GameLogRepository,
    private val gameSessionRepository: GameSessionRepository,
    private val gameLogAggregator: GameLogAggregator,
    private val userStatsRecorder: UserStatsRecorder,
) {
    companion object {
        private val log = LoggerFactory.getLogger(UserStatsService::class.java)
    }

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
    ) = userStatsRecorder.recordGameCompletion(
        chatId = chatId,
        userId = userId,
        sender = sender,
        category = category,
        questionCount = questionCount,
        hintCount = hintCount,
        wrongGuessCount = wrongGuessCount,
        result = result,
        target = target,
        totalGameQuestionCount = totalGameQuestionCount,
    )

    fun recordGameStart(
        chatId: String,
        userId: String,
    ) = userStatsRecorder.recordGameStart(chatId, userId)

    /**
     * 방별 사용자 스탯 조회 (캐시 우선)
     */
    suspend fun getUserStats(
        chatId: String,
        userId: String,
    ): UserStats? {
        // Valkey 캐시 확인
        statsStore.get(chatId, userId)?.let {
            log.debug("STATS_CACHE_HIT chatId={}, userId={}", chatId, userId)
            return it
        }

        // 캐시 미스 - DB 조회
        log.debug("STATS_CACHE_MISS chatId={}, userId={}", chatId, userId)
        val entity = userStatsRepository.findById(UserStatsCalculator.compositeId(chatId, userId)) ?: return null
        val categoryStats = UserStatsCalculator.parseCategoryStatsJson(entity.categoryStatsJson)
        val stats = entity.toUserStats(categoryStats)

        // 캐시 저장
        statsStore.set(chatId, userId, stats)

        return stats
    }

    /**
     * 방 전체 통계 조회 (기간별)
     */
    suspend fun getRoomStats(
        chatId: String,
        period: StatsPeriod,
    ): RoomStatsResult {
        val startTime = period.getStartTime()
        val now = Instant.now()

        // 세션 로그 조회 (총 판수, 완주율)
        val sessionLogs =
            if (startTime != null) {
                gameSessionRepository.findByChatIdAndPeriod(chatId, startTime, now)
            } else {
                gameSessionRepository.findByChatId(
                    chatId,
                    limit = ValidationConstants.MAX_STATS_QUERY_LIMIT,
                )
            }
        val sessionAggregation = aggregateSessions(sessionLogs)
        val completionRate =
            gameLogAggregator.calculateCompletionRate(
                sessionAggregation.totalSessions,
                sessionAggregation.totalCorrectSessions,
            )

        // 참여자 액티비티용 per-user 로그 유지
        val logs =
            if (startTime != null) {
                gameLogRepository.findByChatIdAndPeriod(chatId, startTime, now)
            } else {
                gameLogRepository.findByChatId(
                    chatId,
                    limit = ValidationConstants.MAX_STATS_QUERY_LIMIT,
                )
            }
        val aggregationResult = gameLogAggregator.aggregateGameLogs(logs)
        val participantActivities =
            gameLogAggregator.buildParticipantActivities(aggregationResult.senderGameCounts)

        return RoomStatsResult(
            period = period,
            totalGames = sessionAggregation.totalSessions,
            totalParticipants = aggregationResult.senderGameCounts.size,
            completionRate = completionRate,
            participantActivities = participantActivities,
        )
    }

    private suspend fun aggregateSessions(sessionLogs: Flow<GameSessionEntity>): SessionAggregation {
        var totalSessions = 0
        var totalCorrectSessions = 0

        sessionLogs.collect { session ->
            totalSessions++
            if (session.result == GameResult.CORRECT.name) {
                totalCorrectSessions++
            }
        }

        return SessionAggregation(
            totalSessions = totalSessions,
            totalCorrectSessions = totalCorrectSessions,
        )
    }

    private data class SessionAggregation(
        val totalSessions: Int,
        val totalCorrectSessions: Int,
    )
}

/**
 * 게임 결과 타입
 */
enum class GameResult {
    CORRECT,
    SURRENDER,
}
