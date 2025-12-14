package party.qwer.twentyq.bridge

import kotlinx.coroutines.CoroutineScope
import kotlinx.coroutines.async
import kotlinx.coroutines.coroutineScope
import kotlinx.coroutines.delay
import kotlinx.coroutines.launch
import org.slf4j.Logger
import org.slf4j.LoggerFactory
import party.qwer.twentyq.bridge.handlers.MessageHandlers
import party.qwer.twentyq.logging.LoggingExtensions.sampled
import party.qwer.twentyq.logging.debugL
import party.qwer.twentyq.logging.warnL
import party.qwer.twentyq.model.Command
import party.qwer.twentyq.model.OutboundMessage
import party.qwer.twentyq.model.PendingMessage
import party.qwer.twentyq.model.requiresLlm
import party.qwer.twentyq.redis.tracking.PlayerSetStore
import party.qwer.twentyq.rest.LlmAvailabilityGuard
import party.qwer.twentyq.service.StatsPeriod
import party.qwer.twentyq.service.UserStatsService
import party.qwer.twentyq.service.exception.GameMessageKeys
import party.qwer.twentyq.util.common.extensions.chunkedByLines
import party.qwer.twentyq.util.common.extensions.isAnswerCommand
import party.qwer.twentyq.util.common.formatting.UserIdFormatter
import party.qwer.twentyq.util.game.GameMessageProvider
import party.qwer.twentyq.util.game.extensions.displayName
import party.qwer.twentyq.util.logging.LoggingConstants
import kotlin.time.Duration.Companion.seconds

/**
 * 메시지 명령 실행자
 */
internal class MessageCommandExecutor(
    private val handlers: MessageHandlers,
    private val messageProvider: GameMessageProvider,
    private val playerSetStore: PlayerSetStore,
    private val playerSetScope: CoroutineScope,
    private val commandParser: CommandParser,
    private val waitingMessageDelaySeconds: Long,
    private val userStatsService: UserStatsService,
    private val exceptionHandler: MessageExceptionHandler,
    private val chainedQuestionProcessor: ChainedQuestionProcessor,
    private val waitingMessageCoordinator: WaitingMessageCoordinator,
    private val messageEmitter: MessageEmitter,
    private val llmAvailabilityGuard: LlmAvailabilityGuard,
) {
    companion object {
        private val log = LoggerFactory.getLogger(MessageCommandExecutor::class.java)
    }

    private val generalDispatcher = GeneralCommandDispatcher(handlers, messageProvider, log)

    suspend fun execute(
        context: MessageContext,
        emit: suspend (OutboundMessage) -> Unit,
    ) {
        val command = context.command
        if (command != null && command.requiresLlm() && !llmAvailabilityGuard.isAvailable()) {
            val unavailableMessage = messageProvider.get(GameMessageKeys.AI_UNAVAILABLE)
            sendChunkedReply(unavailableMessage, context.chatId, context.threadId, emit)
            log.warn("LLM_UNAVAILABLE_BLOCKED chatId={}, command={}", context.chatId, command)
            return
        }

        registerPlayerAsync(context)
        if (context.command is Command.ChainedQuestion && context.command.questions.size > 1) {
            handlers.chainedQuestionHandler
                .prepareChainQueue(
                    context.chatId,
                    context.userId,
                    context.sender,
                    context.command.questions,
                )?.let { queueInfo ->
                    emit(OutboundMessage.Waiting(context.chatId, queueInfo, context.threadId))
                    val remaining = context.command.questions.size - 1
                    log.debugL { "CHAIN_QUEUE_PREPARED chatId=${context.chatId}, remaining=$remaining" }
                }
        }

        waitingMessageCoordinator.sendWaitingMessageIfNeeded(context.command, context.chatId, context.threadId, emit)

        val response =
            waitingMessageCoordinator.withDelayedWaitingMessage(
                context.chatId,
                context.threadId,
                waitingMessageDelaySeconds,
                emit,
            ) {
                dispatch(context.command, context.chatId, context.userId, context.sender)
            }

        handleCommandResponse(context, response, emit)
    }

    // 명령 응답 처리
    private suspend fun handleCommandResponse(
        context: MessageContext,
        response: String,
        emit: suspend (OutboundMessage) -> Unit,
    ) {
        if (context.command is Command.Ask || context.command is Command.ChainedQuestion) {
            handleAskCommandResponse(context, response, emit)
        } else {
            sendChunkedReply(response, context.chatId, context.threadId, emit)
        }
        // Success 로깅 (inline)
        log.sampled(
            key = "message.success",
            limit = LoggingConstants.LOG_SAMPLE_LIMIT_HIGH,
            windowMillis = LoggingConstants.LOG_SAMPLE_WINDOW_MS,
        ) {
            val senderName = context.displayName(messageProvider.get("user.anonymous"))
            it.info(
                "MESSAGE_SUCCESS chatId={}, userId={}, senderName={}, command={}",
                context.chatId,
                context.userId,
                senderName,
                context.command,
            )
        }
    }

    // Ask/ChainedQuestion 명령 응답 처리
    private suspend fun handleAskCommandResponse(
        context: MessageContext,
        response: String,
        emit: suspend (OutboundMessage) -> Unit,
    ) {
        sendAskResponse(context.command!!, response, context.chatId, context.threadId, emit)
    }

    // Ask/ChainedQuestion 응답 전송 공통 헬퍼
    private suspend fun sendAskResponse(
        command: Command,
        response: String,
        chatId: String,
        threadId: String?,
        emit: suspend (OutboundMessage) -> Unit,
    ) {
        val isAnswerAttempt = command is Command.Ask && command.question.isAnswerCommand()

        if (isAnswerAttempt) {
            sendChunkedReply(response, chatId, threadId, emit)
        } else {
            val statusMessages =
                runCatching {
                    handlers.statusHandler.handleSeparated(chatId)
                }.getOrElse { ex ->
                    log.warnL { "STATUS_SEPARATED_FAILED chatId=$chatId reason=${ex.message}" }
                    emptyList()
                }

            statusMessages.forEach { message ->
                sendChunkedReply(message, chatId, threadId, emit)
            }
        }
    }

    // 대기열 명령 처리
    suspend fun processQueuedCommand(
        content: String,
        chatId: String,
        userId: String,
        threadId: String?,
        sender: String?,
        emit: suspend (OutboundMessage) -> Unit,
    ) {
        val command = commandParser.parse(content)
        if (command != null && command.requiresLlm() && !llmAvailabilityGuard.isAvailable()) {
            val unavailableMessage = messageProvider.get(GameMessageKeys.AI_UNAVAILABLE)
            sendChunkedReply(unavailableMessage, chatId, threadId, emit)
            log.warn("QUEUE_BLOCKED_LLM_UNAVAILABLE chatId={}, userId={}, command={}", chatId, userId, command)
            return
        }
        val response = dispatch(command, chatId, userId, sender)

        // Ask/ChainedQuestion 명령은 status 반환
        if (command is Command.Ask || command is Command.ChainedQuestion) {
            sendAskResponse(command, response, chatId, threadId, emit)
        } else {
            sendChunkedReply(response, chatId, threadId, emit)
        }

        log.sampled(
            key = "queue.success",
            limit = LoggingConstants.LOG_SAMPLE_LIMIT_LOW,
            windowMillis = LoggingConstants.LOG_SAMPLE_WINDOW_LONG,
        ) {
            it.debug("QUEUE_MESSAGE_PROCESSED chatId={}, userId={}", chatId, userId)
        }
    }

    suspend fun processChainQuestions(
        chatId: String,
        pending: PendingMessage,
        emit: suspend (OutboundMessage) -> Unit,
    ) {
        chainedQuestionProcessor.processChainQuestions(chatId, pending, emit)
    }

    private fun registerPlayerAsync(context: MessageContext) {
        if (context.command == null) return

        playerSetScope.launch {
            runCatching {
                val isNewPlayer =
                    playerSetStore.addAsync(
                        context.chatId,
                        context.userId,
                        context.sender ?: "",
                    )

                if (isNewPlayer) {
                    userStatsService.recordGameStart(
                        chatId = context.chatId,
                        userId = context.userId,
                    )
                    log.info("GAME_START_RECORDED chatId={}, userId={}", context.chatId, context.userId)
                }
            }.onFailure { ex ->
                log.error(
                    "PLAYERSET_ADD_FAILED chatId={}, userId={}, error={}",
                    context.chatId,
                    context.userId,
                    ex.message,
                    ex,
                )
            }
        }
    }

    private suspend fun dispatch(
        command: Command?,
        chatId: String,
        userId: String,
        sender: String?,
    ): String =
        when (command) {
            null -> messageProvider.get(GameMessageKeys.UNKNOWN_COMMAND)
            is Command.AdminForceEnd,
            is Command.AdminClearAll,
            is Command.AdminRefreshCache,
            is Command.AdminRestartAll,
            is Command.AdminUsage,
            -> dispatchAdmin(command, chatId, userId)
            else -> generalDispatcher.dispatchGeneral(command, chatId, userId, sender)
        }

    private suspend fun dispatchAdmin(
        command: Command,
        chatId: String,
        userId: String,
    ): String =
        when (command) {
            is Command.AdminForceEnd -> handlers.adminHandler.forceEnd(chatId, userId)
            is Command.AdminClearAll -> handlers.adminHandler.clearAll(chatId, userId)
            is Command.AdminRefreshCache -> handlers.adminHandler.refreshCache(chatId, userId)
            is Command.AdminRestartAll -> handlers.adminHandler.restartAllBots(chatId, userId)
            is Command.AdminUsage -> handlers.usageHandler.handle(chatId, userId, command.period, command.modelOverride)
            else -> messageProvider.get(GameMessageKeys.UNKNOWN_COMMAND)
        }

    private suspend fun sendChunkedReply(
        response: String,
        chatId: String,
        threadId: String?,
        emit: suspend (OutboundMessage) -> Unit,
    ) {
        messageEmitter.sendChunkedReply(response, chatId, threadId, emit)
    }
}

private class GeneralCommandDispatcher(
    private val handlers: MessageHandlers,
    private val messageProvider: GameMessageProvider,
    private val log: Logger,
) {
    suspend fun dispatchGeneral(
        command: Command,
        chatId: String,
        userId: String,
        sender: String?,
    ): String =
        when (command) {
            is Command.Start -> handlers.startHandler.handle(chatId, command.categories)
            is Command.Hints -> handlers.hintsHandler.handle(chatId, command.count)
            is Command.UserStats -> handleUserStats(command, chatId, userId, sender)
            is Command.Ask -> handlers.askHandler.handle(chatId, command.question, userId, sender)
            is Command.ChainedQuestion -> handlers.chainedQuestionHandler.handle(chatId, command, userId, sender)
            is Command.Surrender, is Command.Agree, is Command.Reject ->
                handleSurrenderCommand(command, chatId, userId)
            is Command.Help, is Command.Status, is Command.HealthCheck, is Command.ModelInfo ->
                handleQueryCommands(command, chatId, userId, sender)
            else -> messageProvider.get(GameMessageKeys.UNKNOWN_COMMAND)
        }

    // Query 커맨드 처리 (Help, Status, HealthCheck)
    private suspend fun handleQueryCommands(
        command: Command,
        chatId: String,
        userId: String,
        sender: String?,
    ): String =
        when (command) {
            is Command.Help -> handlers.helpHandler.handle()
            is Command.Status -> handlers.statusHandler.handle(chatId)
            is Command.HealthCheck ->
                handlers.healthCheckHandler.handle(
                    UserIdFormatter.displayName(userId, sender, chatId, messageProvider.get("user.anonymous")),
                )
            is Command.ModelInfo -> handlers.modelInfoHandler.handle()
            else -> messageProvider.get(GameMessageKeys.UNKNOWN_COMMAND)
        }

    // 사용자 통계 처리
    private suspend fun handleUserStats(
        command: Command.UserStats,
        chatId: String,
        userId: String,
        sender: String?,
    ): String =
        if (command.roomPeriod != null) {
            val period = StatsPeriod.fromString(command.roomPeriod)
            handlers.userStatsHandler.handleRoomStats(chatId, period)
        } else {
            handlers.userStatsHandler.handle(chatId, userId, sender, command.targetNickname)
        }

    private suspend fun withStatusFallback(
        chatId: String,
        primary: suspend () -> String,
    ): String {
        val primaryResult = primary()
        return runCatching { handlers.statusHandler.handle(chatId) }
            .getOrElse { ex ->
                log.warnL { "STATUS_FALLBACK chatId=$chatId reason=${ex.message}" }
                primaryResult
            }
    }

    private suspend fun handleSurrenderCommand(
        command: Command,
        chatId: String,
        userId: String,
    ): String =
        when (command) {
            is Command.Surrender -> handlers.surrenderHandler.handleConsensus(chatId, userId)
            is Command.Agree -> handlers.surrenderHandler.handleAgree(chatId, userId)
            is Command.Reject -> handlers.surrenderHandler.handleReject(chatId, userId)
            else -> messageProvider.get(GameMessageKeys.UNKNOWN_COMMAND)
        }
}
