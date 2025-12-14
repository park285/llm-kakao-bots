package party.qwer.twentyq.service

import org.springframework.stereotype.Component
import party.qwer.twentyq.model.CategoryStat
import java.time.Instant

/**
 * 카테고리 통계 관리 담당 클래스
 */
@Component
class CategoryStatsManager(
    private val calculator: UserStatsCalculator,
) {
    /**
     * 카테고리 통계 업데이트
     */
    fun updateCategoryStats(
        categoryStatsJson: String?,
        category: String,
        questionCount: Int,
        hintCount: Int,
        wrongGuessCount: Int,
        result: GameResult,
        target: String?,
        totalGameQuestionCount: Int = questionCount,
    ): Map<String, CategoryStat> {
        val stats = UserStatsCalculator.parseCategoryStatsJson(categoryStatsJson).toMutableMap()
        val current = stats[category] ?: CategoryStat()

        // 카테고리별 베스트 스코어 계산: 전체 게임 질문 수 사용
        val newBest =
            UserStatsCalculator.computeCategoryBestScore(
                current,
                totalGameQuestionCount,
                wrongGuessCount,
                result,
                target,
            )

        stats[category] =
            when (result) {
                GameResult.CORRECT ->
                    current.copy(
                        gamesCompleted = current.gamesCompleted + 1,
                        questionsAsked = current.questionsAsked + questionCount,
                        hintsUsed = current.hintsUsed + hintCount,
                        wrongGuesses = current.wrongGuesses + wrongGuessCount,
                        bestQuestionCount = newBest?.questionCount ?: current.bestQuestionCount,
                        bestWrongGuessCount = newBest?.wrongGuessCount ?: current.bestWrongGuessCount,
                        bestTarget = newBest?.target ?: current.bestTarget,
                        bestAchievedAt = newBest?.achievedAt ?: current.bestAchievedAt,
                    )
                GameResult.SURRENDER ->
                    current.copy(
                        gamesCompleted = current.gamesCompleted + 1,
                        surrenders = current.surrenders + 1,
                        questionsAsked = current.questionsAsked + questionCount,
                        hintsUsed = current.hintsUsed + hintCount,
                        wrongGuesses = current.wrongGuesses + wrongGuessCount,
                    )
            }
        return stats
    }

    /**
     * 카테고리 통계 생성
     */
    fun createCategoryStat(
        questionCount: Int,
        hintCount: Int,
        wrongGuessCount: Int,
        result: GameResult,
        target: String?,
        totalGameQuestionCount: Int = questionCount,
    ): CategoryStat =
        when (result) {
            GameResult.CORRECT ->
                CategoryStat(
                    gamesCompleted = 1,
                    questionsAsked = questionCount,
                    hintsUsed = hintCount,
                    wrongGuesses = wrongGuessCount,
                    // 베스트 스코어: 전체 게임 질문 수 사용
                    bestQuestionCount = totalGameQuestionCount,
                    bestWrongGuessCount = wrongGuessCount,
                    bestTarget = target,
                    bestAchievedAt = Instant.now(),
                )
            GameResult.SURRENDER ->
                CategoryStat(
                    gamesCompleted = 1,
                    surrenders = 1,
                    questionsAsked = questionCount,
                    hintsUsed = hintCount,
                    wrongGuesses = wrongGuessCount,
                )
        }
}
