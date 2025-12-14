package party.qwer.twentyq.bridge.handlers

import org.springframework.stereotype.Component

@Component
data class MessageHandlers(
    val startHandler: StartHandler,
    val hintsHandler: HintsHandler,
    val askHandler: AskHandler,
    val surrenderHandler: SurrenderHandler,
    val adminHandler: AdminHandler,
    val helpHandler: HelpHandler,
    val healthCheckHandler: HealthCheckHandler,
    val statusHandler: StatusHandler,
    val chainedQuestionHandler: ChainedQuestionHandler,
    val userStatsHandler: UserStatsHandler,
    val usageHandler: UsageHandler,
    val modelInfoHandler: ModelInfoHandler,
)
