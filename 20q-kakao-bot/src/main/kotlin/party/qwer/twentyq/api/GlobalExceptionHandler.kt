package party.qwer.twentyq.api

import kotlinx.coroutines.TimeoutCancellationException
import org.slf4j.LoggerFactory
import org.springframework.http.HttpStatus
import org.springframework.http.ResponseEntity
import org.springframework.web.bind.annotation.ExceptionHandler
import org.springframework.web.bind.annotation.RestControllerAdvice
import party.qwer.twentyq.api.dto.ErrorResponse
import party.qwer.twentyq.service.exception.DuplicateQuestionException
import party.qwer.twentyq.service.exception.ExceptionMessageResolver
import party.qwer.twentyq.service.exception.GameAlreadyExistsException
import party.qwer.twentyq.service.exception.InvalidQuestionException
import party.qwer.twentyq.service.exception.SessionNotFoundException
import party.qwer.twentyq.util.game.GameMessageProvider
import java.util.concurrent.TimeoutException

/** 전역 예외 처리 핸들러 */
@RestControllerAdvice
class GlobalExceptionHandler(
    private val messageProvider: GameMessageProvider,
) {
    companion object {
        private val log = LoggerFactory.getLogger(GlobalExceptionHandler::class.java)
    }

    @ExceptionHandler(SessionNotFoundException::class)
    fun handleSessionNotFound(ex: SessionNotFoundException): ResponseEntity<ErrorResponse> =
        ResponseEntity.status(HttpStatus.BAD_REQUEST).body(
            ErrorResponse(
                error = "SESSION_NOT_FOUND",
                message = ExceptionMessageResolver.resolve(ex, messageProvider),
            ),
        )

    @ExceptionHandler(GameAlreadyExistsException::class)
    fun handleGameAlreadyExists(ex: GameAlreadyExistsException): ResponseEntity<ErrorResponse> =
        ResponseEntity.status(HttpStatus.CONFLICT).body(
            ErrorResponse(
                error = "GAME_ALREADY_EXISTS",
                message = ExceptionMessageResolver.resolve(ex, messageProvider),
            ),
        )

    @ExceptionHandler(InvalidQuestionException::class)
    fun handleInvalidQuestion(ex: InvalidQuestionException): ResponseEntity<ErrorResponse> =
        ResponseEntity.status(HttpStatus.BAD_REQUEST).body(
            ErrorResponse(
                error = "INVALID_QUESTION",
                message = ExceptionMessageResolver.resolve(ex, messageProvider),
            ),
        )

    @ExceptionHandler(DuplicateQuestionException::class)
    fun handleDuplicateQuestion(ex: DuplicateQuestionException): ResponseEntity<ErrorResponse> =
        ResponseEntity.status(HttpStatus.BAD_REQUEST).body(
            ErrorResponse(
                error = "INVALID_QUESTION_DUPLICATE",
                message = ExceptionMessageResolver.resolve(ex, messageProvider),
            ),
        )

    @ExceptionHandler(IllegalArgumentException::class)
    fun handleIllegalArgument(ex: IllegalArgumentException): ResponseEntity<ErrorResponse> =
        ResponseEntity.status(HttpStatus.BAD_REQUEST).body(
            ErrorResponse(error = "BAD_REQUEST", message = ex.message ?: "Invalid request"),
        )

    @ExceptionHandler(IllegalStateException::class)
    fun handleIllegalState(ex: IllegalStateException): ResponseEntity<ErrorResponse> =
        ResponseEntity.status(HttpStatus.BAD_GATEWAY).body(
            ErrorResponse(error = "UPSTREAM_ERROR", message = ex.message ?: "Upstream processing failed"),
        )

    @ExceptionHandler(TimeoutCancellationException::class)
    fun handleTimeout(ex: TimeoutCancellationException): ResponseEntity<ErrorResponse> {
        log.warn("AI timeout: {}", ex.message)
        return ResponseEntity.status(HttpStatus.GATEWAY_TIMEOUT).body(
            ErrorResponse(
                error = "AI_TIMEOUT",
                message = ExceptionMessageResolver.resolve(ex, messageProvider),
            ),
        )
    }

    @ExceptionHandler(TimeoutException::class)
    fun handleJavaTimeout(ex: TimeoutException): ResponseEntity<ErrorResponse> {
        log.warn("AI timeout (java): {}", ex.message)
        return ResponseEntity.status(HttpStatus.GATEWAY_TIMEOUT).body(
            ErrorResponse(
                error = "AI_TIMEOUT",
                message = ExceptionMessageResolver.resolve(ex, messageProvider),
            ),
        )
    }

    @ExceptionHandler(Exception::class)
    fun handleGeneric(ex: Exception): ResponseEntity<ErrorResponse> {
        log.warn("Unhandled exception: {}", ex.message)
        return ResponseEntity.status(HttpStatus.INTERNAL_SERVER_ERROR).body(
            ErrorResponse(
                error = "INTERNAL_ERROR",
                message = ExceptionMessageResolver.resolve(ex, messageProvider),
            ),
        )
    }
}
