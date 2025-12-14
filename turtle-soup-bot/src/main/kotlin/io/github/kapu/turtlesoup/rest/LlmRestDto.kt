package io.github.kapu.turtlesoup.rest

import kotlinx.serialization.SerialName
import kotlinx.serialization.Serializable

// =============================================================================
// Request DTOs
// =============================================================================

@Serializable
data class AnswerRequest(
    @SerialName("session_id") val sessionId: String? = null,
    @SerialName("chat_id") val chatId: String? = null,
    val namespace: String? = null,
    val scenario: String,
    val solution: String,
    val question: String,
)

@Serializable
data class HintRequest(
    @SerialName("session_id") val sessionId: String? = null,
    @SerialName("chat_id") val chatId: String? = null,
    val namespace: String? = null,
    val scenario: String,
    val solution: String,
    val level: Int,
)

@Serializable
data class ValidateRequest(
    @SerialName("session_id") val sessionId: String? = null,
    @SerialName("chat_id") val chatId: String? = null,
    val namespace: String? = null,
    val solution: String,
    @SerialName("player_answer") val playerAnswer: String,
)

@Serializable
data class RewriteRequest(
    val title: String,
    val scenario: String,
    val solution: String,
    val difficulty: Int,
)

@Serializable
data class GuardRequest(
    @SerialName("input_text") val inputText: String,
)

@Serializable
data class SessionCreateRequest(
    @SerialName("session_id") val sessionId: String? = null,
    @SerialName("chat_id") val chatId: String? = null,
    val namespace: String? = null,
)

// =============================================================================
// Response DTOs
// =============================================================================

@Serializable
data class AnswerResponse(
    val answer: String,
    @SerialName("raw_text") val rawText: String = "",
    @SerialName("question_count") val questionCount: Int,
    val history: List<HistoryItem> = emptyList(),
)

@Serializable
data class HistoryItem(
    val question: String,
    val answer: String,
)

@Serializable
data class HintResponse(
    val hint: String,
    val level: Int,
)

@Serializable
data class ValidateResponse(
    val result: String,
    @SerialName("raw_text") val rawText: String = "",
)

@Serializable
data class RewriteResponse(
    val scenario: String,
    val solution: String,
    @SerialName("original_scenario") val originalScenario: String = "",
    @SerialName("original_solution") val originalSolution: String = "",
)

@Serializable
data class GuardMaliciousResponse(
    val malicious: Boolean,
)

@Serializable
data class SessionCreateResponse(
    @SerialName("session_id") val sessionId: String,
    val model: String = "",
    val created: Boolean = false,
)

@Serializable
data class SessionEndResponse(
    val removed: Boolean,
)

// =============================================================================
// Puzzle API DTOs
// =============================================================================

@Serializable
data class PuzzlePresetResponse(
    val id: Int? = null,
    val title: String? = null,
    val question: String? = null,
    val answer: String? = null,
    val difficulty: Int? = null,
)
