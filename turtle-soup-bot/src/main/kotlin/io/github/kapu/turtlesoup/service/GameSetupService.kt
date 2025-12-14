package io.github.kapu.turtlesoup.service

import io.github.kapu.turtlesoup.models.GameState
import io.github.kapu.turtlesoup.models.Puzzle
import io.github.kapu.turtlesoup.models.PuzzleCategory
import io.github.kapu.turtlesoup.models.PuzzleGenerationRequest
import io.github.kapu.turtlesoup.rest.LlmRestClient
import io.github.kapu.turtlesoup.utils.GameAlreadyStartedException
import io.github.oshai.kotlinlogging.KotlinLogging

class GameSetupService(
    private val restClient: LlmRestClient,
    private val puzzleService: PuzzleService,
    private val sessionManager: GameSessionManager,
) {
    suspend fun prepareNewGame(
        sessionId: String,
        userId: String,
        chatId: String,
        difficulty: Int?,
        category: PuzzleCategory?,
        theme: String?,
    ): GameSetupResult {
        val existing = sessionManager.load(sessionId)
        handleExistingGame(sessionId, existing)

        val puzzle =
            puzzleService.generatePuzzle(
                request =
                    PuzzleGenerationRequest(
                        category = category,
                        difficulty = difficulty,
                        theme = theme,
                    ),
                chatId = chatId,
            )
        restClient.createSession(chatId)
        logger.info { "llm_session_created" }

        val state = buildInitialState(sessionId, userId, chatId, puzzle)
        sessionManager.save(state)

        return GameSetupResult(state, puzzle)
    }

    private suspend fun handleExistingGame(
        sessionId: String,
        existing: GameState?,
    ) {
        if (existing == null) return

        if (existing.isSolved) {
            sessionManager.delete(sessionId)
        } else {
            throw GameAlreadyStartedException(sessionId)
        }
    }

    private fun buildInitialState(
        sessionId: String,
        userId: String,
        chatId: String,
        puzzle: Puzzle,
    ): GameState =
        GameState(
            sessionId = sessionId,
            userId = userId,
            chatId = chatId,
            puzzle = puzzle,
            players = setOf(userId),
        )

    companion object {
        private val logger = KotlinLogging.logger {}
    }
}

data class GameSetupResult(
    val state: GameState,
    val puzzle: Puzzle,
)
