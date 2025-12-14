package party.qwer.twentyq.bridge

import kotlinx.coroutines.async
import kotlinx.coroutines.coroutineScope
import kotlinx.coroutines.delay
import kotlinx.coroutines.launch
import org.slf4j.LoggerFactory
import org.springframework.stereotype.Component
import party.qwer.twentyq.bridge.handlers.MessageHandlers
import party.qwer.twentyq.logging.debugL
import party.qwer.twentyq.model.Command
import party.qwer.twentyq.model.OutboundMessage
import party.qwer.twentyq.util.game.GameMessageProvider
import kotlin.time.Duration.Companion.seconds

/**
 * 대기 메시지 타이밍 조정 담당 클래스
 */
@Component
class WaitingMessageCoordinator(
    private val handlers: MessageHandlers,
    private val messageProvider: GameMessageProvider,
) {
    companion object {
        private val log = LoggerFactory.getLogger(WaitingMessageCoordinator::class.java)
    }

    suspend fun <T> withDelayedWaitingMessage(
        chatId: String,
        threadId: String?,
        delaySeconds: Long,
        emit: suspend (OutboundMessage) -> Unit,
        block: suspend () -> T,
    ): T =
        coroutineScope {
            val resultDeferred = async { block() }

            val timerJob =
                launch {
                    delay(delaySeconds.seconds)
                    if (resultDeferred.isActive) {
                        val waitingMsg = messageProvider.get("processing.waiting")
                        emit(OutboundMessage.Waiting(chatId, waitingMsg, threadId))
                        log.debugL { "WAITING_MESSAGE_SENT chatId=$chatId, delay=${delaySeconds}s" }
                    }
                }

            try {
                resultDeferred.await()
            } finally {
                timerJob.cancel()
            }
        }

    suspend fun sendWaitingMessageIfNeeded(
        command: Command?,
        chatId: String,
        threadId: String?,
        emit: suspend (OutboundMessage) -> Unit,
    ) {
        when (command) {
            is Command.Start -> {
                val sessionExists =
                    runCatching { handlers.startHandler.hasExistingSession(chatId) }
                        .onFailure { ex -> log.debugL { "SESSION_CHECK_FAILED chatId=$chatId, error=${ex.message}" } }
                        .getOrDefault(false)

                if (!sessionExists) {
                    val waitingMessage = messageProvider.get("start.waiting")
                    emit(OutboundMessage.Waiting(chatId, waitingMessage, threadId))
                }
            }

            is Command.Hints -> {
                val shouldShowWaiting =
                    runCatching {
                        handlers.hintsHandler.isHintAvailable(chatId) &&
                            handlers.hintsHandler.isFirstHint(chatId)
                    }.onFailure { ex ->
                        log.debugL { "HINT_WAITING_CHECK_FAILED chatId=$chatId, error=${ex.message}" }
                    }.getOrDefault(false)

                if (shouldShowWaiting) {
                    log.debugL { "BEFORE_SEND_WAITING chatId=$chatId, type=HINTS" }
                    val waitingMessage = messageProvider.get("hint.waiting")
                    emit(OutboundMessage.Waiting(chatId, waitingMessage, threadId))
                    log.debugL { "WAITING_MESSAGE_SENT chatId=$chatId, type=HINTS" }
                }
            }

            else -> Unit
        }
    }
}
