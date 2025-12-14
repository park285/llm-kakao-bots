package io.github.kapu.turtlesoup.service

import io.github.kapu.turtlesoup.config.GameConstants
import io.github.kapu.turtlesoup.models.AnswerResult
import io.github.kapu.turtlesoup.models.GameState
import io.github.kapu.turtlesoup.models.HistoryEntry
import io.github.kapu.turtlesoup.models.Puzzle
import io.github.kapu.turtlesoup.models.PuzzleCategory
import io.github.kapu.turtlesoup.models.SurrenderResult
import io.github.kapu.turtlesoup.models.ValidationResult
import io.github.kapu.turtlesoup.rest.AnswerQuestionResult
import io.github.kapu.turtlesoup.rest.LlmRestClient
import io.github.kapu.turtlesoup.security.McpInjectionGuard
import io.github.kapu.turtlesoup.utils.GameAlreadySolvedException
import io.github.kapu.turtlesoup.utils.GameNotStartedException
import io.github.kapu.turtlesoup.utils.InvalidQuestionException
import io.github.kapu.turtlesoup.utils.MaxHintsReachedException
import io.github.kapu.turtlesoup.utils.canUseHint
import io.github.kapu.turtlesoup.utils.isValidAnswer
import io.github.kapu.turtlesoup.utils.isValidQuestion
import io.github.oshai.kotlinlogging.KotlinLogging
import kotlin.math.max

class GameService(
    private val restClient: LlmRestClient,
    private val sessionManager: GameSessionManager,
    private val setupService: GameSetupService,
    private val injectionGuard: McpInjectionGuard,
) {
    suspend fun startGame(
        sessionId: String,
        userId: String,
        chatId: String,
        difficulty: Int? = null,
        category: PuzzleCategory? = null,
        theme: String? = null,
    ): GameState {
        return sessionManager.withLock(sessionId, holderName = userId) {
            val setupResult =
                setupService.prepareNewGame(
                    sessionId = sessionId,
                    userId = userId,
                    chatId = chatId,
                    difficulty = difficulty,
                    category = category,
                    theme = theme,
                )
            logGameStarted(sessionId, userId, setupResult.puzzle)
            setupResult.state
        }
    }

    suspend fun registerPlayer(
        sessionId: String,
        userId: String,
    ) {
        sessionManager.withLock(sessionId, holderName = userId) {
            val state = sessionManager.load(sessionId) ?: return@withLock

            val baseState =
                if (state.players.isEmpty()) {
                    state.copy(players = setOf(state.userId))
                } else {
                    state
                }

            if (userId in baseState.players) return@withLock

            val updated = baseState.addPlayer(userId)
            sessionManager.save(updated)
        }
    }

    suspend fun askQuestion(
        sessionId: String,
        question: String,
    ): Pair<GameState, AnswerQuestionResult> {
        if (!question.isValidQuestion()) {
            throw InvalidQuestionException("Invalid question format")
        }

        // 보안 검증 (로컬 InjectionGuard + MCP guard_evaluate)
        val sanitizedQuestion = injectionGuard.validateOrThrow(question)

        return sessionManager.withOwnerLock(sessionId) {
            val state = sessionManager.loadOrThrow(sessionId)

            if (state.isSolved) {
                throw GameAlreadySolvedException(sessionId)
            }

            val puzzle = state.puzzle ?: throw GameNotStartedException(sessionId)

            // MCP를 통한 LLM 호출 (보안 검사 포함, question_count 포함)
            val result =
                restClient.answerQuestion(
                    chatId = sessionId,
                    scenario = puzzle.scenario,
                    solution = puzzle.solution,
                    question = sanitizedQuestion,
                )

            val resolvedHistory =
                result.history.map { HistoryEntry(question = it.question, answer = it.answer) }
            val (mergedHistory, mergedQuestionCount) = mergeHistory(state, resolvedHistory, result.questionCount)

            // LLM에서 관리하는 question_count와 히스토리를 상태에 반영 (단, 축소 응답은 보호)
            val newState =
                state.copy(
                    questionCount = mergedQuestionCount,
                    history = mergedHistory,
                    lastActivityAt = java.time.Instant.now(),
                )
            sessionManager.save(newState)
            sessionManager.refresh(sessionId)

            logger.info {
                "question_answered session_id=$sessionId question_count=${newState.questionCount}"
            }

            newState to result
        }
    }

    private fun mergeHistory(
        state: GameState,
        resolvedHistory: List<HistoryEntry>,
        resolvedQuestionCount: Int,
    ): Pair<List<HistoryEntry>, Int> {
        val lastEntry = resolvedHistory.lastOrNull()
        val shouldAppend = lastEntry != null && state.history.lastOrNull() != lastEntry

        val mergedHistory =
            when {
                resolvedHistory.size >= state.history.size -> resolvedHistory
                shouldAppend -> state.history + checkNotNull(lastEntry)
                else -> state.history
            }
        val mergedQuestionCount =
            max(
                resolvedQuestionCount,
                max(
                    state.questionCount + if (shouldAppend) 1 else 0,
                    mergedHistory.size,
                ),
            )
        return mergedHistory to mergedQuestionCount
    }

    suspend fun submitSolution(
        sessionId: String,
        playerAnswer: String,
    ): Pair<GameState, ValidationResult> {
        if (!playerAnswer.isValidAnswer()) {
            throw InvalidQuestionException("Invalid answer format")
        }

        // 보안 검증
        val sanitizedAnswer = injectionGuard.validateOrThrow(playerAnswer)

        return sessionManager.withOwnerLock(sessionId) {
            val state = sessionManager.loadOrThrow(sessionId)

            if (state.isSolved) {
                throw GameAlreadySolvedException(sessionId)
            }

            val puzzle = state.puzzle ?: throw GameNotStartedException(sessionId)

            // MCP를 통한 정답 검증
            val result =
                restClient.validateSolution(
                    chatId = sessionId,
                    solution = puzzle.solution,
                    playerAnswer = sanitizedAnswer,
                )

            val newState =
                if (result == ValidationResult.YES) {
                    state.markSolved()
                } else {
                    state.updateActivity()
                }

            sessionManager.save(newState)

            logger.info { "solution_submitted session_id=$sessionId result=$result" }

            if (result == ValidationResult.YES) {
                sessionManager.delete(sessionId)
                runCatching { restClient.endSessionByChat(sessionId) }
                    .onFailure { logger.warn(it) { "llm_session_end_failed session_id=$sessionId" } }
                logger.info {
                    "game_ended session_id=$sessionId reason=solved question_count=${newState.questionCount} " +
                        "hints_used=${newState.hintsUsed}"
                }
            }

            newState to result
        }
    }

    suspend fun submitAnswer(
        sessionId: String,
        answer: String,
    ): AnswerResult {
        val (state, result) = submitSolution(sessionId, answer)
        return AnswerResult(
            result = result,
            questionCount = state.questionCount,
            hintCount = state.hintsUsed,
            maxHints = GameConstants.MAX_HINTS,
            hintsUsed = state.hintContents,
            explanation = if (result == ValidationResult.YES) state.puzzle?.solution ?: "" else "",
        )
    }

    suspend fun requestHint(sessionId: String): Pair<GameState, String> {
        return sessionManager.withOwnerLock(sessionId) {
            val state = sessionManager.loadOrThrow(sessionId)

            if (state.isSolved) {
                throw GameAlreadySolvedException(sessionId)
            }

            if (!state.canUseHint()) {
                throw MaxHintsReachedException()
            }

            val puzzle = state.puzzle ?: throw GameNotStartedException(sessionId)

            // MCP를 통한 힌트 생성
            val hint =
                restClient.generateHint(
                    chatId = sessionId,
                    scenario = puzzle.scenario,
                    solution = puzzle.solution,
                    level = state.hintsUsed + 1,
                )

            val newState = state.useHint(hint)
            sessionManager.save(newState)

            logger.info {
                "hint_requested session_id=$sessionId hints_used=${newState.hintsUsed}"
            }

            newState to hint
        }
    }

    suspend fun surrender(sessionId: String): SurrenderResult {
        return sessionManager.withOwnerLock(sessionId) {
            val state = sessionManager.loadOrThrow(sessionId)
            val puzzle = state.puzzle ?: throw GameNotStartedException(sessionId)

            sessionManager.delete(sessionId)
            runCatching { restClient.endSessionByChat(sessionId) }
                .onFailure { logger.warn(it) { "llm_session_end_failed session_id=$sessionId" } }

            logger.info {
                "game_surrendered session_id=$sessionId " +
                    "question_count=${state.questionCount} hints_used=${state.hintsUsed}"
            }

            SurrenderResult(
                solution = puzzle.solution,
                hintsUsed = state.hintContents,
            )
        }
    }

    suspend fun getGameState(sessionId: String): GameState {
        return sessionManager.loadOrThrow(sessionId)
    }

    suspend fun endGame(sessionId: String) {
        sessionManager.withOwnerLock(sessionId) {
            val state = sessionManager.load(sessionId)

            sessionManager.delete(sessionId)
            runCatching { restClient.endSessionByChat(sessionId) }
                .onFailure { logger.warn(it) { "llm_session_end_failed session_id=$sessionId" } }
            logger.info { "game_ended session_id=$sessionId" }
        }
    }

    private fun logGameStarted(
        sessionId: String,
        userId: String,
        puzzle: Puzzle,
    ) {
        logger.info {
            "game_started session_id=$sessionId user_id=$userId " +
                "puzzle_title=${puzzle.title} difficulty=${puzzle.difficulty} solution=${puzzle.solution}"
        }
    }

    companion object {
        private val logger = KotlinLogging.logger {}
    }
}
