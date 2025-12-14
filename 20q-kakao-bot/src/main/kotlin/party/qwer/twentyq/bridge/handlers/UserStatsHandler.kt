package party.qwer.twentyq.bridge.handlers

import org.slf4j.LoggerFactory
import org.springframework.stereotype.Component
import party.qwer.twentyq.model.CategoryStat
import party.qwer.twentyq.model.RiddleCategory
import party.qwer.twentyq.model.RoomStatsResult
import party.qwer.twentyq.model.UserStats
import party.qwer.twentyq.redis.RiddleSessionRepository
import party.qwer.twentyq.service.StatsPeriod
import party.qwer.twentyq.service.UserStatsService
import party.qwer.twentyq.util.game.GameMessageProvider

/**
 * 사용자 전적 조회 핸들러
 */
@Component
class UserStatsHandler(
    private val userStatsService: UserStatsService,
    private val messageProvider: GameMessageProvider,
    private val sessionRepo: RiddleSessionRepository,
) {
    companion object {
        private val log = LoggerFactory.getLogger(UserStatsHandler::class.java)
        private const val TOP_CATEGORIES_LIMIT = 5
        private const val PERCENTAGE_MULTIPLIER = 100
    }

    suspend fun handle(
        chatId: String,
        userId: String,
        sender: String?,
        targetNickname: String? = null,
    ): String {
        // 다른 사용자 전적 조회
        if (targetNickname != null) {
            val targetPlayer = sessionRepo.getPlayerByNickname(chatId, targetNickname)
            if (targetPlayer == null) {
                return messageProvider.get(
                    "stats.user_not_found",
                    "nickname" to targetNickname,
                )
            }
            val stats = userStatsService.getUserStats(chatId, targetPlayer.userId)
            return if (stats != null) {
                formatStats(stats, targetPlayer.sender)
            } else {
                messageProvider.get(
                    "stats.no_stats",
                    "nickname" to targetPlayer.sender,
                )
            }
        }

        // 본인 전적 조회
        val stats = userStatsService.getUserStats(chatId, userId)
        return if (stats != null) {
            formatStats(stats, sender)
        } else {
            messageProvider.get("stats.not_found")
        }
    }

    /**
     * 방 전체 통계 조회 핸들러
     */
    suspend fun handleRoomStats(
        chatId: String,
        period: StatsPeriod,
    ): String {
        val roomStats = userStatsService.getRoomStats(chatId, period)

        // 게임이 없는 경우
        if (roomStats.totalGames == 0) {
            return messageProvider.get(
                "stats.room.no_games",
                "period" to getPeriodName(period),
            )
        }

        return formatRoomStats(roomStats)
    }

    /**
     * 방 통계 포맷팅
     */
    private fun formatRoomStats(roomStats: RoomStatsResult): String {
        val parts = mutableListOf<String>()

        // 헤더
        parts.add(
            messageProvider.get(
                "stats.room.header",
                "period" to getPeriodName(roomStats.period),
            ),
        )

        // 전체 요약
        parts.add("")
        parts.add(
            messageProvider.get(
                "stats.room.summary",
                "totalGames" to roomStats.totalGames,
                "totalParticipants" to roomStats.totalParticipants,
                "completionRate" to roomStats.completionRate,
            ),
        )

        // 참여 액티비티
        if (roomStats.participantActivities.isNotEmpty()) {
            parts.add("")
            parts.add(messageProvider.get("stats.room.activity_header"))

            roomStats.participantActivities.forEach { activity ->
                parts.add(
                    messageProvider.get(
                        "stats.room.activity_item",
                        "sender" to activity.sender,
                        "games" to activity.gamesPlayed,
                    ),
                )
            }
        }

        return parts.joinToString("\n")
    }

    /**
     * 기간명 반환
     */
    private fun getPeriodName(period: StatsPeriod): String =
        when (period) {
            StatsPeriod.DAILY -> messageProvider.get("stats.period.daily")
            StatsPeriod.WEEKLY -> messageProvider.get("stats.period.weekly")
            StatsPeriod.MONTHLY -> messageProvider.get("stats.period.monthly")
            StatsPeriod.ALL -> messageProvider.get("stats.period.all")
        }

    private fun formatStats(
        stats: UserStats,
        sender: String?,
    ): String {
        val parts = mutableListOf<String>()
        addStatsHeader(parts, stats, sender)

        if (stats.categoryStats.isNotEmpty()) {
            addCategoryStats(parts, stats.categoryStats)
        }

        return parts.joinToString("\n")
    }

    // 통계 헤더 추가
    private fun addStatsHeader(
        parts: MutableList<String>,
        stats: UserStats,
        sender: String?,
    ) {
        val nickname = sender ?: messageProvider.get("user.anonymous")
        parts.add(
            messageProvider.get(
                "stats.header",
                "nickname" to nickname,
                "totalGames" to stats.totalGamesCompleted,
            ),
        )
    }

    // 카테고리별 통계 추가
    private fun addCategoryStats(
        parts: MutableList<String>,
        categoryStats: Map<String, CategoryStat>,
    ) {
        val topCategories =
            categoryStats
                .entries
                .sortedByDescending { it.value.gamesCompleted }
                .take(TOP_CATEGORIES_LIMIT)

        parts.add("") // 첫 카테고리 앞 빈 줄

        topCategories.forEachIndexed { index, (category, stat) ->
            if (index > 0) parts.add("") // 카테고리 간 빈 줄
            addSingleCategoryStats(parts, category, stat)
        }
    }

    // 단일 카테고리 통계 추가
    private fun addSingleCategoryStats(
        parts: MutableList<String>,
        category: String,
        stat: CategoryStat,
    ) {
        // 카테고리 헤더 (영어→한국어 변환)
        val categoryName = RiddleCategory.fromString(category).koreanName
        parts.add(
            messageProvider.get(
                "stats.category.header",
                "category" to categoryName,
                "games" to stat.gamesCompleted,
            ),
        )

        // 완주율 계산
        val completionRate = calculateCompletionRate(stat)
        parts.add(
            messageProvider.get(
                "stats.category.results",
                "completed" to stat.gamesCompleted,
                "surrender" to stat.surrenders,
                "completionRate" to completionRate,
            ),
        )

        // 평균 질문/힌트
        val avgQuestions = calculateAverage(stat.questionsAsked, stat.gamesCompleted)
        val avgHints = calculateAverage(stat.hintsUsed, stat.gamesCompleted)
        parts.add(
            messageProvider.get(
                "stats.category.averages",
                "avgQuestions" to String.format("%.1f", avgQuestions),
                "avgHints" to String.format("%.1f", avgHints),
            ),
        )

        // 베스트 스코어
        addBestScore(parts, stat)
    }

    // 완주율 계산
    private fun calculateCompletionRate(stat: CategoryStat): Int {
        if (stat.gamesCompleted == 0) return 0
        val completed = stat.gamesCompleted - stat.surrenders
        return (completed.toDouble() / stat.gamesCompleted * PERCENTAGE_MULTIPLIER).toInt()
    }

    // 평균 계산
    private fun calculateAverage(
        total: Int,
        count: Int,
    ): Double = if (count > 0) total.toDouble() / count else 0.0

    // 베스트 스코어 추가
    private fun addBestScore(
        parts: MutableList<String>,
        stat: CategoryStat,
    ) {
        if (stat.bestQuestionCount != null && stat.bestTarget != null) {
            parts.add(
                messageProvider.get(
                    "stats.category.best",
                    "count" to stat.bestQuestionCount,
                    "target" to stat.bestTarget,
                ),
            )
        } else {
            parts.add(messageProvider.get("stats.category.no_best"))
        }
    }
}
