package party.qwer.twentyq.bridge

import org.springframework.stereotype.Component
import party.qwer.twentyq.util.game.GameMessageProvider

/**
 * 메시징 관련 의존성 묶음
 */
@Component
internal class MessagingSupport(
    val messageProvider: GameMessageProvider,
    val exceptionHandler: MessageExceptionHandler,
)
