package io.github.kapu.turtlesoup.api

import io.github.kapu.turtlesoup.api.dto.AskQuestionRequest
import io.github.kapu.turtlesoup.api.dto.ErrorResponse
import io.github.kapu.turtlesoup.api.dto.GameStateResponse
import io.github.kapu.turtlesoup.api.dto.HintRequest
import io.github.kapu.turtlesoup.api.dto.HintResponse
import io.github.kapu.turtlesoup.api.dto.QuestionResponse
import io.github.kapu.turtlesoup.api.dto.SolutionResponse
import io.github.kapu.turtlesoup.api.dto.StartGameRequest
import io.github.kapu.turtlesoup.api.dto.SubmitSolutionRequest
import io.github.kapu.turtlesoup.config.ApiErrorCodes
import io.github.kapu.turtlesoup.models.ValidationResult
import io.github.kapu.turtlesoup.service.GameService
import io.github.kapu.turtlesoup.utils.GameAlreadyStartedException
import io.github.kapu.turtlesoup.utils.GameNotStartedException
import io.github.kapu.turtlesoup.utils.MaxHintsReachedException
import io.github.kapu.turtlesoup.utils.SessionNotFoundException
import io.github.kapu.turtlesoup.utils.TurtleSoupException
import io.github.kapu.turtlesoup.utils.remainingHints
import io.github.kapu.turtlesoup.utils.safeMessage
import io.github.oshai.kotlinlogging.KotlinLogging
import io.ktor.http.HttpStatusCode
import io.ktor.server.application.Application
import io.ktor.server.application.ApplicationCall
import io.ktor.server.request.receive
import io.ktor.server.response.respond
import io.ktor.server.routing.Route
import io.ktor.server.routing.delete
import io.ktor.server.routing.get
import io.ktor.server.routing.post
import io.ktor.server.routing.route
import io.ktor.server.routing.routing
import org.koin.ktor.ext.inject

private val logger = KotlinLogging.logger {}

fun Application.configureGameRoutes() {
    val gameService: GameService by inject()

    routing {
        route("/api/game") {
            startGameRoute(gameService)
            askQuestionRoute(gameService)
            submitSolutionRoute(gameService)
            requestHintRoute(gameService)
            getStatusRoute(gameService)
            endGameRoute(gameService)
        }
    }
}

private fun Route.startGameRoute(gameService: GameService) {
    post("/start") {
        try {
            val req = call.receive<StartGameRequest>()
            val state =
                gameService.startGame(
                    sessionId = req.sessionId,
                    userId = req.userId,
                    chatId = req.chatId,
                    difficulty = req.difficulty,
                    category = req.category,
                    theme = req.theme,
                )

            call.respond(HttpStatusCode.OK, state.toResponse())
        } catch (e: TurtleSoupException) {
            call.respondGameError(e, "start_game_failed")
        }
    }
}

private suspend fun ApplicationCall.respondGameError(
    error: TurtleSoupException,
    logEvent: String,
) {
    when (error) {
        is SessionNotFoundException,
        is GameAlreadyStartedException,
        is MaxHintsReachedException,
        -> {}
        else -> logger.error(error) { logEvent }
    }

    val (status, code) =
        when (error) {
            is SessionNotFoundException -> HttpStatusCode.NotFound to ApiErrorCodes.SESSION_NOT_FOUND
            is GameAlreadyStartedException -> HttpStatusCode.Conflict to ApiErrorCodes.GAME_ALREADY_STARTED
            is MaxHintsReachedException -> HttpStatusCode.BadRequest to ApiErrorCodes.MAX_HINTS_REACHED
            else -> HttpStatusCode.BadRequest to ApiErrorCodes.GAME_ERROR
        }

    respond(status, ErrorResponse(code, error.safeMessage))
}

private fun io.github.kapu.turtlesoup.models.GameState.toResponse(): GameStateResponse {
    val puzzle = this.puzzle ?: throw GameNotStartedException(sessionId)
    return GameStateResponse(
        sessionId = sessionId,
        userId = userId,
        chatId = chatId,
        scenarioTitle = puzzle.title,
        scenario = puzzle.scenario,
        questionCount = questionCount,
        hintsUsed = hintsUsed,
        isSolved = isSolved,
        elapsedSeconds = elapsedSeconds,
    )
}

private fun Route.askQuestionRoute(gameService: GameService) {
    post("/question") {
        try {
            val req = call.receive<AskQuestionRequest>()

            val (state, result) =
                gameService.askQuestion(
                    sessionId = req.sessionId,
                    question = req.question,
                )

            call.respond(
                HttpStatusCode.OK,
                QuestionResponse(
                    answer = result.answer,
                    questionCount = state.questionCount,
                ),
            )
        } catch (e: TurtleSoupException) {
            call.respondGameError(e, "ask_question_failed")
        }
    }
}

private fun Route.submitSolutionRoute(gameService: GameService) {
    post("/solution") {
        try {
            val req = call.receive<SubmitSolutionRequest>()

            val (state, result) =
                gameService.submitSolution(
                    sessionId = req.sessionId,
                    playerAnswer = req.answer,
                )

            call.respond(
                HttpStatusCode.OK,
                SolutionResponse(
                    result = result.name,
                    solution = if (result == ValidationResult.YES) state.puzzle?.solution else null,
                ),
            )
        } catch (e: TurtleSoupException) {
            call.respondGameError(e, "submit_solution_failed")
        }
    }
}

private fun Route.requestHintRoute(gameService: GameService) {
    post("/hint") {
        try {
            val req = call.receive<HintRequest>()

            val (state, hint) = gameService.requestHint(req.sessionId)

            call.respond(
                HttpStatusCode.OK,
                HintResponse(
                    hint = hint,
                    hintsUsed = state.hintsUsed,
                    hintsRemaining = state.remainingHints(),
                ),
            )
        } catch (e: TurtleSoupException) {
            call.respondGameError(e, "request_hint_failed")
        }
    }
}

private fun Route.getStatusRoute(gameService: GameService) {
    get("/status/{sessionId}") {
        try {
            val sessionId =
                call.parameters["sessionId"]
                    ?: return@get call.respond(
                        HttpStatusCode.BadRequest,
                        ErrorResponse(ApiErrorCodes.INVALID_REQUEST, "Session ID required"),
                    )

            val state = gameService.getGameState(sessionId)
            call.respond(HttpStatusCode.OK, state.toResponse())
        } catch (e: TurtleSoupException) {
            call.respondGameError(e, "get_status_failed")
        }
    }
}

private fun Route.endGameRoute(gameService: GameService) {
    delete("/{sessionId}") {
        try {
            val sessionId =
                call.parameters["sessionId"]
                    ?: return@delete call.respond(
                        HttpStatusCode.BadRequest,
                        ErrorResponse(ApiErrorCodes.INVALID_REQUEST, "Session ID required"),
                    )

            gameService.endGame(sessionId)
            call.respond(HttpStatusCode.NoContent)
        } catch (e: TurtleSoupException) {
            call.respondGameError(e, "end_game_failed")
        }
    }
}
