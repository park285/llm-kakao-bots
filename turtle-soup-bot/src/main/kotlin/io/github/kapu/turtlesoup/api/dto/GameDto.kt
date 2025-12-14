package io.github.kapu.turtlesoup.api.dto

import io.github.kapu.turtlesoup.models.PuzzleCategory
import kotlinx.serialization.Serializable

@Serializable
data class StartGameRequest(
    val sessionId: String,
    val userId: String,
    val chatId: String,
    val category: PuzzleCategory? = null,
    val difficulty: Int? = null,
    val theme: String? = null,
)

@Serializable
data class AskQuestionRequest(
    val sessionId: String,
    val question: String,
)

@Serializable
data class SubmitSolutionRequest(
    val sessionId: String,
    val answer: String,
)

@Serializable
data class HintRequest(
    val sessionId: String,
)

@Serializable
data class GameStateResponse(
    val sessionId: String,
    val userId: String,
    val chatId: String,
    val scenarioTitle: String,
    val scenario: String,
    val questionCount: Int,
    val hintsUsed: Int,
    val isSolved: Boolean,
    val elapsedSeconds: Long,
)

@Serializable
data class QuestionResponse(
    val answer: String,
    val questionCount: Int,
)

@Serializable
data class SolutionResponse(
    val result: String,
    val solution: String? = null,
)

@Serializable
data class HintResponse(
    val hint: String,
    val hintsUsed: Int,
    val hintsRemaining: Int,
)

@Serializable
data class ErrorResponse(
    val error: String,
    val message: String,
)
