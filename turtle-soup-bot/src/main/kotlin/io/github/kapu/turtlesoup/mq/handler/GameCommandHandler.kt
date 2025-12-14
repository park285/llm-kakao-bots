package io.github.kapu.turtlesoup.mq.handler

import io.github.kapu.turtlesoup.bridge.SurrenderHandler
import io.github.kapu.turtlesoup.config.GameConstants
import io.github.kapu.turtlesoup.config.PuzzleConstants
import io.github.kapu.turtlesoup.models.Command
import io.github.kapu.turtlesoup.models.GameState
import io.github.kapu.turtlesoup.mq.MessageBuilder
import io.github.kapu.turtlesoup.mq.ValkeyMQReplyPublisher
import io.github.kapu.turtlesoup.mq.models.InboundMessage
import io.github.kapu.turtlesoup.mq.models.OutboundMessage
import io.github.kapu.turtlesoup.rest.LlmRestClient
import io.github.kapu.turtlesoup.service.GameService
import io.github.kapu.turtlesoup.utils.GameAlreadyStartedException
import io.github.kapu.turtlesoup.utils.MessageKeys
import io.github.kapu.turtlesoup.utils.MessageProvider
import io.github.oshai.kotlinlogging.KotlinLogging

/** 게임 명령어 처리 핸들러 */
@Suppress("TooManyFunctions")
class GameCommandHandler(
    private val gameService: GameService,
    private val surrenderHandler: SurrenderHandler,
    private val messageProvider: MessageProvider,
    private val publisher: ValkeyMQReplyPublisher,
    private val restClient: LlmRestClient,
) {
    private val messageBuilder = MessageBuilder(messageProvider)

    suspend fun processCommand(
        message: InboundMessage,
        command: Command,
    ): String {
        if (shouldRegisterPlayer(command)) {
            gameService.registerPlayer(message.chatId, message.userId)
        }

        val handlers: Map<Class<out Command>, suspend () -> String> =
            mapOf(
                Command.Hint::class.java to { handleHint(message) },
                Command.Problem::class.java to { handleProblem(message) },
                Command.Surrender::class.java to {
                    surrenderHandler.handleConsensus(message.chatId, message.userId)
                },
                Command.Agree::class.java to { surrenderHandler.handleAgree(message.chatId, message.userId) },
                Command.Summary::class.java to { handleSummary(message) },
                Command.Help::class.java to { messageProvider.get(MessageKeys.HELP_MESSAGE) },
                Command.Unknown::class.java to { messageProvider.get(MessageKeys.ERROR_UNKNOWN_COMMAND) },
            )

        return when (command) {
            is Command.Start -> handleStart(message, command)
            is Command.Ask -> handleAsk(message, command.question)
            is Command.Answer -> handleAnswer(message, command.answer)
            else ->
                handlers[command::class.java]?.invoke()
                    ?: messageProvider.get(MessageKeys.ERROR_UNKNOWN_COMMAND)
        }
    }

    private suspend fun handleStart(
        message: InboundMessage,
        command: Command.Start,
    ): String {
        val difficulty = resolveDifficulty(command)
        val startState = startOrResumeGame(message, difficulty.value)

        publishScenario(message, startState.state, startState.isResuming)

        val instructionMessage = buildInstructionMessage(startState.state, startState.isResuming)
        return difficulty.warning?.let { "$it\n\n$instructionMessage" } ?: instructionMessage
    }

    private suspend fun handleAsk(
        message: InboundMessage,
        question: String,
    ): String {
        logger.info { "handleAsk_start session_id=${message.chatId} question=$question" }
        val (state, result) = gameService.askQuestion(sessionId = message.chatId, question = question)
        logger.info { "handleAsk_complete session_id=${message.chatId}" }
        return messageProvider.get(MessageKeys.ANSWER_RESPONSE_SINGLE, "answer" to result.answer)
    }

    private suspend fun handleAnswer(
        message: InboundMessage,
        answer: String,
    ): String {
        val result = gameService.submitAnswer(sessionId = message.chatId, answer = answer)

        return when {
            result.isCorrect ->
                messageProvider.get(
                    MessageKeys.ANSWER_CORRECT,
                    "explanation" to result.explanation,
                    "questionCount" to result.questionCount.toString(),
                    "hintCount" to result.hintCount.toString(),
                    "maxHints" to result.maxHints.toString(),
                    "hintBlock" to messageBuilder.buildHintBlock(result.hintsUsed),
                )
            result.isClose -> messageProvider.get(MessageKeys.ANSWER_CLOSE_CALL)
            else -> messageProvider.get(MessageKeys.ANSWER_INCORRECT)
        }
    }

    private suspend fun handleHint(message: InboundMessage): String {
        val (state, hint) = gameService.requestHint(sessionId = message.chatId)
        return messageProvider.get(
            MessageKeys.HINT_GENERATED,
            "hintNumber" to state.hintsUsed.toString(),
            "content" to hint,
        )
    }

    private suspend fun handleProblem(message: InboundMessage): String {
        val state = gameService.getGameState(sessionId = message.chatId)
        val scenario = state.puzzle?.scenario ?: messageProvider.get(MessageKeys.FALLBACK_PUZZLE_NOT_FOUND)
        return messageProvider.get(
            MessageKeys.PROBLEM_DISPLAY,
            "scenario" to scenario,
            "questionCount" to state.questionCount.toString(),
            "hintCount" to state.hintsUsed.toString(),
            "maxHints" to GameConstants.MAX_HINTS.toString(),
        )
    }

    private suspend fun handleSummary(message: InboundMessage): String {
        val state = gameService.getGameState(sessionId = message.chatId)
        return messageBuilder.buildSummary(state.history)
    }

    companion object {
        private val logger = KotlinLogging.logger {}
    }

    private fun resolveDifficulty(command: Command.Start): DifficultySelection {
        if (command.hasInvalidInput) {
            return DifficultySelection(
                value = null,
                warning =
                    messageProvider.get(
                        MessageKeys.START_INVALID_DIFFICULTY,
                        "min" to PuzzleConstants.MIN_DIFFICULTY,
                        "max" to PuzzleConstants.MAX_DIFFICULTY,
                    ),
            )
        }

        val desired = command.difficulty ?: return DifficultySelection(null, null)
        return if (desired in PuzzleConstants.MIN_DIFFICULTY..PuzzleConstants.MAX_DIFFICULTY) {
            DifficultySelection(desired, null)
        } else {
            DifficultySelection(
                value = null,
                warning =
                    messageProvider.get(
                        MessageKeys.START_INVALID_DIFFICULTY,
                        "min" to PuzzleConstants.MIN_DIFFICULTY,
                        "max" to PuzzleConstants.MAX_DIFFICULTY,
                    ),
            )
        }
    }

    private suspend fun startOrResumeGame(
        message: InboundMessage,
        difficulty: Int?,
    ): StartState {
        return runCatching {
            val state =
                gameService.startGame(
                    sessionId = message.chatId,
                    userId = message.userId,
                    chatId = message.chatId,
                    difficulty = difficulty,
                    category = null,
                    theme = null,
                )
            StartState(state = state, isResuming = false)
        }.getOrElse { error ->
            if (error is GameAlreadyStartedException) {
                logger.debug { "game_already_started session_id=${message.chatId} resuming" }
                StartState(state = gameService.getGameState(message.chatId), isResuming = true)
            } else {
                throw error
            }
        }
    }

    private suspend fun publishScenario(
        message: InboundMessage,
        state: GameState,
        isResuming: Boolean,
    ) {
        val puzzle = state.puzzle
        val scenario = puzzle?.scenario ?: messageProvider.get(MessageKeys.FALLBACK_PUZZLE_NOT_FOUND)
        val difficulty = puzzle?.difficulty ?: PuzzleConstants.DEFAULT_DIFFICULTY
        val scenarioMessage =
            if (isResuming) {
                messageProvider.get(MessageKeys.START_RESUME, "scenario" to scenario)
            } else {
                messageProvider.get(
                    MessageKeys.START_SCENARIO,
                    "scenario" to scenario,
                    "difficulty" to buildDifficultyStars(difficulty),
                )
            }

        publisher.publish(
            OutboundMessage.Final(
                chatId = message.chatId,
                text = scenarioMessage,
                threadId = message.threadId,
            ),
        )
    }

    private fun buildInstructionMessage(
        state: GameState,
        isResuming: Boolean,
    ): String =
        if (isResuming) {
            messageProvider.get(
                MessageKeys.START_RESUME_STATUS,
                "questionCount" to state.questionCount,
                "hintCount" to state.hintsUsed,
            )
        } else {
            messageProvider.get(MessageKeys.START_INSTRUCTION)
        }

    private fun buildDifficultyStars(difficulty: Int): String {
        val clamped = difficulty.coerceIn(PuzzleConstants.MIN_DIFFICULTY, PuzzleConstants.MAX_DIFFICULTY)
        return "★".repeat(clamped) + "☆".repeat(PuzzleConstants.MAX_DIFFICULTY - clamped)
    }

    private fun shouldRegisterPlayer(command: Command): Boolean =
        when (command) {
            is Command.Help, is Command.Unknown -> false
            else -> true
        }
}

private data class DifficultySelection(
    val value: Int?,
    val warning: String?,
)

private data class StartState(
    val state: GameState,
    val isResuming: Boolean,
)
