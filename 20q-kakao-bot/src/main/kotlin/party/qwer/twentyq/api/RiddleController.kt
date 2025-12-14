package party.qwer.twentyq.api

import org.slf4j.LoggerFactory
import org.springframework.http.ResponseEntity
import org.springframework.web.bind.annotation.GetMapping
import org.springframework.web.bind.annotation.PostMapping
import org.springframework.web.bind.annotation.RequestBody
import org.springframework.web.bind.annotation.RequestHeader
import org.springframework.web.bind.annotation.RequestMapping
import org.springframework.web.bind.annotation.RestController
import org.springframework.web.server.ServerWebExchange
import party.qwer.twentyq.api.dto.RiddleAnswerRequest
import party.qwer.twentyq.api.dto.RiddleAnswerResponse
import party.qwer.twentyq.api.dto.RiddleCreateRequest
import party.qwer.twentyq.api.dto.RiddleCreateResponse
import party.qwer.twentyq.api.dto.RiddleHintsRequest
import party.qwer.twentyq.api.dto.RiddleHintsResponse
import party.qwer.twentyq.api.dto.RiddleStatusResponse
import party.qwer.twentyq.model.FiveScaleKo
import party.qwer.twentyq.rest.LlmAvailabilityGuard
import party.qwer.twentyq.service.RiddleService
import party.qwer.twentyq.service.exception.GameMessageKeys
import party.qwer.twentyq.util.common.extensions.measureDurationMillis
import party.qwer.twentyq.util.game.GameMessageProvider
import party.qwer.twentyq.util.game.constants.GameConstants.MAX_HINT_REQUEST
import party.qwer.twentyq.util.game.constants.GameConstants.MIN_HINT_REQUEST
import party.qwer.twentyq.util.logging.LoggingConstants.LOG_TEXT_SHORT

/** 수수께끼 게임 API 컨트롤러 */
@RestController
@RequestMapping("/api/twentyq/riddles")
class RiddleController(
    private val riddleService: RiddleService,
    private val llmAvailabilityGuard: LlmAvailabilityGuard,
    private val messageProvider: GameMessageProvider,
) {
    companion object {
        private val log = LoggerFactory.getLogger(RiddleController::class.java)
        private const val INTERNAL_CHAT_PREFIX: String = "internal:"
        private const val FALLBACK_ADDR: String = "local"
        private const val DEFAULT_CATEGORY: String = "ANY"
        private const val HEADER_SESSION_ID: String = "X-Session-Id"
        private const val HEADER_USER_ID: String = "X-User-Id"
    }

    private fun resolveChatId(
        chatId: String?,
        remoteAddr: String?,
    ): String = chatId ?: "$INTERNAL_CHAT_PREFIX${remoteAddr ?: FALLBACK_ADDR}"

    @PostMapping
    suspend fun create(
        @RequestHeader(HEADER_SESSION_ID, required = false) chatId: String?,
        @RequestBody(required = false) request: RiddleCreateRequest?,
        exchange: ServerWebExchange,
    ): ResponseEntity<RiddleCreateResponse> {
        val category = request?.category
        val remoteAddr =
            exchange.request.remoteAddress
                ?.address
                ?.hostAddress
        val resolvedChatId = resolveChatId(chatId, remoteAddr)
        val resolvedCategory = category ?: DEFAULT_CATEGORY
        log.info("CREATE_REQUEST chatId={}, category={}", resolvedChatId, resolvedCategory)

        if (!llmAvailabilityGuard.isAvailable()) {
            val unavailableMessage = messageProvider.get(GameMessageKeys.AI_UNAVAILABLE)
            log.warn("API_BLOCKED_LLM_UNAVAILABLE chatId={}", resolvedChatId)
            return ResponseEntity.ok(RiddleCreateResponse(message = unavailableMessage))
        }

        val (message, duration) =
            measureDurationMillis {
                riddleService.createRiddle(
                    chatId = resolvedChatId,
                    category = category,
                )
            }
        log.info(
            "CREATE_SUCCESS chatId={}, category={}, duration={}ms",
            resolvedChatId,
            resolvedCategory,
            duration,
        )
        return ResponseEntity.ok(RiddleCreateResponse(message = message))
    }

    @PostMapping("/hints")
    suspend fun hints(
        @RequestHeader(HEADER_SESSION_ID, required = false) chatId: String?,
        @RequestBody req: RiddleHintsRequest,
        exchange: ServerWebExchange,
    ): ResponseEntity<RiddleHintsResponse> {
        val remoteAddr =
            exchange.request.remoteAddress
                ?.address
                ?.hostAddress
        val resolvedChatId = resolveChatId(chatId, remoteAddr)
        log.info("HINTS_REQUEST chatId={}, count={}", resolvedChatId, req.count)

        if (!llmAvailabilityGuard.isAvailable()) {
            val unavailableMessage = messageProvider.get(GameMessageKeys.AI_UNAVAILABLE)
            log.warn("API_BLOCKED_LLM_UNAVAILABLE chatId={}", resolvedChatId)
            return ResponseEntity.ok(RiddleHintsResponse(hints = listOf(unavailableMessage)))
        }

        require(req.count in MIN_HINT_REQUEST..MAX_HINT_REQUEST) {
            "count must be between 1 and 10"
        }
        val (hints, duration) =
            measureDurationMillis {
                riddleService.generateHints(chatId = resolvedChatId, count = req.count)
            }
        log.info(
            "HINTS_SUCCESS chatId={}, count={}, hintsGenerated={}, duration={}ms",
            resolvedChatId,
            req.count,
            hints.size,
            duration,
        )
        return ResponseEntity.ok(RiddleHintsResponse(hints = hints))
    }

    @PostMapping("/answers")
    suspend fun answer(
        @RequestHeader(HEADER_SESSION_ID, required = false) chatId: String?,
        @RequestHeader(HEADER_USER_ID, required = false) userId: String?,
        @RequestBody req: RiddleAnswerRequest,
        exchange: ServerWebExchange,
    ): ResponseEntity<RiddleAnswerResponse> {
        val remoteAddr =
            exchange.request.remoteAddress
                ?.address
                ?.hostAddress
        val resolvedChatId = resolveChatId(chatId, remoteAddr)
        val resolvedUserId = userId ?: remoteAddr ?: "unknown"
        log.info(
            "ANSWER_REQUEST chatId={}, userId={}, question='{}'",
            resolvedChatId,
            resolvedUserId,
            req.question.take(LOG_TEXT_SHORT),
        )

        if (!llmAvailabilityGuard.isAvailable()) {
            val unavailableMessage = messageProvider.get(GameMessageKeys.AI_UNAVAILABLE)
            log.warn("API_BLOCKED_LLM_UNAVAILABLE chatId={}, userId={}", resolvedChatId, resolvedUserId)
            return ResponseEntity.ok(RiddleAnswerResponse(scale = unavailableMessage))
        }

        val (result, duration) =
            measureDurationMillis {
                riddleService.answer(chatId = resolvedChatId, question = req.question, userId = resolvedUserId)
            }
        val scaleToken = FiveScaleKo.token(result.scale)
        log.info(
            "ANSWER_SUCCESS chatId={}, scale={}, source={}, duration={}ms",
            resolvedChatId,
            scaleToken,
            result.source,
            duration,
        )
        return ResponseEntity.ok(RiddleAnswerResponse(scale = scaleToken))
    }

    @GetMapping
    suspend fun status(
        @RequestHeader(HEADER_SESSION_ID, required = false) chatId: String?,
        exchange: ServerWebExchange,
    ): ResponseEntity<RiddleStatusResponse> {
        val remoteAddr =
            exchange.request.remoteAddress
                ?.address
                ?.hostAddress
        val resolvedChatId = resolveChatId(chatId, remoteAddr)
        log.info("STATUS_REQUEST chatId={}", resolvedChatId)

        val (status, duration) =
            measureDurationMillis {
                riddleService.getStatus(chatId = resolvedChatId)
            }
        log.info(
            "STATUS_SUCCESS chatId={}, questionCount={}, duration={}ms",
            resolvedChatId,
            status.questionCount,
            duration,
        )
        return ResponseEntity.ok(status)
    }
}
