package party.qwer.twentyq.service

import party.qwer.twentyq.model.UserStatsEntity
import java.time.Instant

object UserStatsFactory {
    fun updateExistingStats(
        existing: UserStatsEntity,
        category: String,
        questionCount: Int,
        hintCount: Int,
        wrongGuessCount: Int,
        result: GameResult,
        target: String?,
        categoryStatsManager: CategoryStatsManager,
        totalGameQuestionCount: Int = questionCount,
    ): UserStatsEntity {
        val categoryStats =
            categoryStatsManager.updateCategoryStats(
                existing.categoryStatsJson,
                category,
                questionCount,
                hintCount,
                wrongGuessCount,
                result,
                target,
                totalGameQuestionCount,
            )
        val totalSurrender = UserStatsCalculator.computeTotalStats(existing.totalSurrenders, result)
        // 베스트 스코어 계산: 전체 게임 질문 수 사용
        val newBestScore =
            UserStatsCalculator.computeBestScore(
                existing.bestScoreQuestionCount,
                existing.bestScoreWrongGuessCount,
                totalGameQuestionCount,
                wrongGuessCount,
                result,
                target,
                category,
            )

        return existing.copy(
            totalGamesCompleted = existing.totalGamesCompleted + 1,
            totalSurrenders = totalSurrender,
            totalQuestionsAsked = existing.totalQuestionsAsked + questionCount,
            totalHintsUsed = existing.totalHintsUsed + hintCount,
            totalWrongGuesses = existing.totalWrongGuesses + wrongGuessCount,
            bestScoreQuestionCount = newBestScore?.questionCount ?: existing.bestScoreQuestionCount,
            bestScoreWrongGuessCount = newBestScore?.wrongGuessCount ?: existing.bestScoreWrongGuessCount,
            bestScoreTarget = newBestScore?.target ?: existing.bestScoreTarget,
            bestScoreCategory = newBestScore?.category ?: existing.bestScoreCategory,
            bestScoreAchievedAt = if (newBestScore != null) Instant.now() else existing.bestScoreAchievedAt,
            categoryStatsJson = UserStatsCalculator.serializeCategoryStats(categoryStats),
            updatedAt = Instant.now(),
        )
    }

    fun createNewStats(
        chatId: String,
        userId: String,
        category: String,
        questionCount: Int,
        hintCount: Int,
        wrongGuessCount: Int,
        result: GameResult,
        target: String?,
        categoryStatsManager: CategoryStatsManager,
        totalGameQuestionCount: Int = questionCount,
    ): UserStatsEntity {
        val categoryStat =
            categoryStatsManager.createCategoryStat(
                questionCount,
                hintCount,
                wrongGuessCount,
                result,
                target,
                totalGameQuestionCount,
            )
        val categoryStats = mapOf(category to categoryStat)
        val isCorrect = result == GameResult.CORRECT

        return UserStatsEntity(
            id = UserStatsCalculator.compositeId(chatId, userId),
            chatId = chatId,
            userId = userId,
            totalGamesStarted = 1,
            totalGamesCompleted = 1,
            totalSurrenders = if (result == GameResult.SURRENDER) 1 else 0,
            totalQuestionsAsked = questionCount,
            totalHintsUsed = hintCount,
            totalWrongGuesses = wrongGuessCount,
            // 베스트 스코어: 전체 게임 질문 수 사용
            bestScoreQuestionCount = if (isCorrect) totalGameQuestionCount else null,
            bestScoreWrongGuessCount = if (isCorrect) wrongGuessCount else null,
            bestScoreTarget = if (isCorrect) target else null,
            bestScoreCategory = if (isCorrect) category else null,
            bestScoreAchievedAt = if (isCorrect) Instant.now() else null,
            categoryStatsJson = UserStatsCalculator.serializeCategoryStats(categoryStats),
            createdAt = Instant.now(),
            updatedAt = Instant.now(),
        ).markAsNew()
    }
}
