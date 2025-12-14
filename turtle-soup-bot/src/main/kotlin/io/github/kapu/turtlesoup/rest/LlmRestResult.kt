package io.github.kapu.turtlesoup.rest

import io.github.kapu.turtlesoup.config.PuzzleConstants
import io.github.kapu.turtlesoup.models.Puzzle
import io.github.kapu.turtlesoup.models.PuzzleCategory

// Result 클래스 (내부 사용, 직렬화 불필요)

data class AnswerQuestionResult(
    val answer: String,
    val questionCount: Int,
    val history: List<QuestionHistoryItem> = emptyList(),
)

data class QuestionHistoryItem(
    val question: String,
    val answer: String,
)

data class RewriteResult(
    val scenario: String,
    val solution: String,
)

data class SessionCreateResult(
    val sessionId: String,
    val model: String = "",
    val created: Boolean = false,
)

// Puzzle API Result
data class PuzzlePresetResult(
    val id: Int,
    val title: String,
    val question: String,
    val answer: String,
    val difficulty: Int,
) {
    fun toPuzzle(): Puzzle =
        Puzzle(
            title = title,
            scenario = question,
            solution = answer,
            category = PuzzleCategory.MYSTERY,
            difficulty = difficulty.coerceIn(PuzzleConstants.MIN_DIFFICULTY, PuzzleConstants.MAX_DIFFICULTY),
        )
}
