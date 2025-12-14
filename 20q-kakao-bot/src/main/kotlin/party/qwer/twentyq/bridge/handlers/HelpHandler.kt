package party.qwer.twentyq.bridge.handlers

import org.springframework.stereotype.Component
import party.qwer.twentyq.util.game.GameMessageProvider

@Component
class HelpHandler(
    private val messageProvider: GameMessageProvider,
) {
    fun handle(): String = messageProvider.get("help.message")
}
