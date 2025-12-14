package party.qwer.twentyq.bridge.handlers

import org.springframework.stereotype.Component
import party.qwer.twentyq.util.game.GameMessageProvider

@Component
class HealthCheckHandler(
    private val messageProvider: GameMessageProvider,
) {
    fun handle(nickname: String): String = messageProvider.get("health.alive", "nickname" to nickname)
}
