package party.qwer.twentyq.bridge.handlers

import org.slf4j.LoggerFactory
import org.springframework.stereotype.Component
import party.qwer.twentyq.logging.debugL
import party.qwer.twentyq.service.RiddleService
import party.qwer.twentyq.util.common.security.requireSessionOrThrow
import party.qwer.twentyq.util.game.GameMessageProvider

@Component
class HintsHandler(
    private val riddleService: RiddleService,
    private val messageProvider: GameMessageProvider,
) {
    companion object {
        private val log = LoggerFactory.getLogger(HintsHandler::class.java)
    }

    suspend fun handle(
        chatId: String,
        count: Int,
    ): String {
        log.info("HANDLE_HINTS chatId={}, requestedCount={}, fixedCount=1", chatId, count)
        riddleService.requireSessionOrThrow(chatId)

        val hints = riddleService.generateHints(chatId, 1)

        if (hints.isEmpty()) {
            return messageProvider.get("error.hint_not_available")
        }

        val status = riddleService.getStatus(chatId)
        val hintNumber = status.hints.lastOrNull()?.hintNumber ?: status.hintCount

        return messageProvider.get(
            "hint.generated",
            "hintNumber" to hintNumber,
            "content" to hints.first(),
        )
    }

    suspend fun isHintAvailable(chatId: String): Boolean =
        runCatching { riddleService.isHintAvailable(chatId) }
            .onFailure { ex ->
                log.debugL { "HINT_AVAILABILITY_CHECK_FAILED chatId=$chatId, error=${ex.message}" }
            }.getOrElse { false }

    suspend fun isFirstHint(chatId: String): Boolean =
        runCatching {
            riddleService.getStatus(chatId).hintCount == 0
        }.onFailure { ex ->
            log.debugL { "FIRST_HINT_CHECK_FAILED chatId=$chatId, error=${ex.message}" }
        }.getOrElse { false }
}
