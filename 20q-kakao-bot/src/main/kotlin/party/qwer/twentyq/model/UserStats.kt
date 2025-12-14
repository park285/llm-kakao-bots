package party.qwer.twentyq.model

import java.time.Instant

/**
 * 사용자 게임 통계 (응답 DTO)
 */
data class UserStats(
    val userId: String,
    val totalGamesStarted: Int = 0,
    val totalGamesCompleted: Int = 0,
    val totalSurrenders: Int = 0,
    val totalQuestionsAsked: Int = 0,
    val totalHintsUsed: Int = 0,
    val totalWrongGuesses: Int = 0,
    val bestScore: BestScore? = null,
    val categoryStats: Map<String, CategoryStat> = emptyMap(),
)

/**
 * 최고 기록 (가장 적은 질문으로 정답 맞춘 기록)
 */
data class BestScore(
    val questionCount: Int,
    val wrongGuessCount: Int = 0,
    val target: String,
    val category: String,
    val achievedAt: Instant,
)

/**
 * 카테고리별 통계
 */
data class CategoryStat(
    val gamesCompleted: Int = 0,
    val surrenders: Int = 0,
    val questionsAsked: Int = 0,
    val hintsUsed: Int = 0,
    val wrongGuesses: Int = 0,
    val bestQuestionCount: Int? = null,
    val bestWrongGuessCount: Int? = null,
    val bestTarget: String? = null,
    val bestAchievedAt: Instant? = null,
)
