package io.github.kapu.turtlesoup.models

import kotlinx.serialization.Serializable
import java.time.Duration
import java.time.Instant

@Serializable
data class GameState(
    val sessionId: String,
    val userId: String,
    val chatId: String,
    val puzzle: Puzzle? = null,
    val questionCount: Int = 0,
    val history: List<HistoryEntry> = emptyList(),
    val hintsUsed: Int = 0,
    val hintContents: List<String> = emptyList(),
    val players: Set<String> = emptySet(),
    val isSolved: Boolean = false,
    @Serializable(with = InstantSerializer::class)
    val startedAt: Instant = Instant.now(),
    @Serializable(with = InstantSerializer::class)
    val lastActivityAt: Instant = Instant.now(),
) {
    init {
        require(hintsUsed >= 0) { "Hints used cannot be negative" }
        require(questionCount >= 0) { "Question count cannot be negative" }
    }

    val elapsedSeconds: Long
        get() = Duration.between(startedAt, Instant.now()).seconds

    fun useHint(hintContent: String): GameState =
        copy(
            hintsUsed = hintsUsed + 1,
            hintContents = hintContents + hintContent,
            lastActivityAt = Instant.now(),
        )

    fun appendHistory(
        question: String,
        answer: String,
    ): GameState =
        copy(
            questionCount = questionCount + 1,
            history = history + HistoryEntry(question = question, answer = answer),
            lastActivityAt = Instant.now(),
        )

    fun addPlayer(playerId: String): GameState =
        copy(
            players = players + playerId,
            lastActivityAt = Instant.now(),
        )

    fun markSolved(): GameState =
        copy(
            isSolved = true,
            lastActivityAt = Instant.now(),
        )

    fun updateActivity(): GameState =
        copy(
            lastActivityAt = Instant.now(),
        )
}

@Serializable
data class HistoryEntry(
    val question: String,
    val answer: String,
)
