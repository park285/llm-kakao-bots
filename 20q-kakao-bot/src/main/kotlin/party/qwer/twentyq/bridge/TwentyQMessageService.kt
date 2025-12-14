package party.qwer.twentyq.bridge

import kotlinx.coroutines.CoroutineScope
import kotlinx.coroutines.Dispatchers
import kotlinx.coroutines.SupervisorJob
import org.slf4j.LoggerFactory
import org.springframework.stereotype.Component
import party.qwer.twentyq.bridge.handlers.MessageHandlers
import party.qwer.twentyq.config.AppProperties
import party.qwer.twentyq.logging.LoggingExtensions.sampled
import party.qwer.twentyq.model.Command
import party.qwer.twentyq.model.InboundMessage
import party.qwer.twentyq.model.OutboundMessage
import party.qwer.twentyq.model.requiresWriteLock
import party.qwer.twentyq.mq.MessageQueueCoordinator
import party.qwer.twentyq.redis.LockCoordinator
import party.qwer.twentyq.redis.ProcessingLockService
import party.qwer.twentyq.redis.tracking.PlayerSetStore
import party.qwer.twentyq.rest.LlmAvailabilityGuard
import party.qwer.twentyq.rest.NlpRestClient
import party.qwer.twentyq.service.UserStatsService
import party.qwer.twentyq.service.exception.GameMessageKeys
import party.qwer.twentyq.util.common.security.resolveAccessDenialReason
import party.qwer.twentyq.util.game.GameMessageProvider
import party.qwer.twentyq.util.game.constants.GameConstants
import party.qwer.twentyq.util.game.extensions.displayName
import party.qwer.twentyq.util.logging.LoggingConstants

/**
 * 카카오 메시지 처리 서비스
 */
@Component
class TwentyQMessageService(
    private val appProperties: AppProperties,
    private val lockCoordinator: LockCoordinator,
    private val processingLockService: ProcessingLockService,
    private val queueCoordinator: MessageQueueCoordinator,
    private val playerSetStore: PlayerSetStore,
    private val handlers: MessageHandlers,
    private val messageProvider: GameMessageProvider,
    private val exceptionHandler: MessageExceptionHandler,
    private val nlpRestClient: NlpRestClient,
    private val userStatsService: UserStatsService,
    private val chainedQuestionProcessor: ChainedQuestionProcessor,
    private val waitingMessageCoordinator: WaitingMessageCoordinator,
    private val messageEmitter: MessageEmitter,
    private val llmAvailabilityGuard: LlmAvailabilityGuard,
) {
    companion object {
        private val log = LoggerFactory.getLogger(TwentyQMessageService::class.java)

        private const val MESSAGE_TRUNCATE_LENGTH = 50
        private const val PLAYER_SET_PARALLELISM = 32
    }

    private val commandParser by lazy {
        CommandParser(appProperties.commands.prefix, nlpRestClient)
    }

    private val playerSetExceptionHandler =
        kotlinx.coroutines.CoroutineExceptionHandler { _, ex ->
            log.error("PLAYERSET_BACKGROUND_TASK_FAILED error={}", ex.message, ex)
        }
    private val playerSetScope =
        CoroutineScope(
            Dispatchers.IO.limitedParallelism(PLAYER_SET_PARALLELISM) + SupervisorJob() + playerSetExceptionHandler,
        )

    private val commandExecutor by lazy {
        MessageCommandExecutor(
            handlers,
            messageProvider,
            playerSetStore,
            playerSetScope,
            commandParser,
            GameConstants.WAITING_MESSAGE_DELAY_SECONDS,
            userStatsService,
            exceptionHandler,
            chainedQuestionProcessor,
            waitingMessageCoordinator,
            messageEmitter,
            llmAvailabilityGuard,
        )
    }

    private val lockingSupport by lazy {
        LockingSupport(lockCoordinator, processingLockService)
    }

    private val messagingSupport by lazy {
        MessagingSupport(messageProvider, exceptionHandler)
    }

    private val queueNotifier by lazy {
        MessageQueueNotifier(messageProvider, exceptionHandler)
    }

    private val queueProcessor by lazy {
        MessageQueueProcessor(
            queueCoordinator,
            lockingSupport,
            messagingSupport,
            commandParser,
            commandExecutor,
            queueNotifier,
            appProperties.mq.maxQueueProcessIterations,
        )
    }

    /**
     * 메시지 처리 메인 진입점: validation → access control → lock → execution
     * - 모든 명령어 실행 오류를 사용자 메시지로 변환 (resilience 보장)
     */
    suspend fun handle(
        message: InboundMessage,
        emit: suspend (OutboundMessage) -> Unit,
    ) {
        val context = validateMessageInput(message) ?: return

        if (!canProceed(context, emit)) return

        // pending/processing 체크 → 즉시 enqueue
        if (shouldEnqueueImmediately(context, emit)) return

        // Lock 획득 및 비즈니스 로직 실행
        val requiresWrite = context.command?.requiresWriteLock() ?: true
        val executed =
            lockCoordinator.withLock(context.chatId, context.userId, requiresWrite) {
                processingLockService.startProcessing(context.chatId)

                val executionResult =
                    runCatching {
                        commandExecutor.execute(context, emit)
                    }

                executionResult.exceptionOrNull()?.let { ex ->
                    val exception = ex as? Exception ?: Exception(ex.message)
                    val errorMessage = exceptionHandler.getErrorMessage(exception, context.chatId, context.userId)
                    sendErrorResponse(errorMessage, context.chatId, context.threadId, emit)
                }

                processingLockService.finishProcessing(context.chatId)
            }

        // Lock 획득 실패 → enqueue
        if (executed == null) {
            log.warn("MESSAGE_REJECTED_LOCKED chatId={}, userId={}", context.chatId, context.userId)
            queueProcessor.enqueueAndNotify(
                context.chatId,
                context.userId,
                context.content,
                context.threadId,
                context.sender,
                emit,
            )
        }

        // 타임아웃/실패 포함 항상 큐 처리 시도
        queueProcessor.processQueuedMessages(context.chatId, emit)
    }

    private suspend fun canProceed(
        context: MessageContext,
        emit: suspend (OutboundMessage) -> Unit,
    ): Boolean {
        val denialReason = checkAccessControl(context)
        if (denialReason != null) {
            log.warn("MESSAGE_REJECTED chatId={}, userId={}, reason={}", context.chatId, context.userId, denialReason)
            val message =
                if (denialReason == "error.user_blocked") {
                    val nickname = context.displayName(messageProvider.get("user.anonymous"))
                    messageProvider.get(denialReason, "nickname" to nickname)
                } else {
                    messageProvider.get(denialReason)
                }
            sendErrorResponse(message, context.chatId, context.threadId, emit)
            return false
        }

        if (requiresExistingSession(context.command) && !hasExistingSession(context)) {
            log.warn("MESSAGE_REJECTED_NO_SESSION chatId={}, userId={}", context.chatId, context.userId)
            val message = messageProvider.get(GameMessageKeys.NO_SESSION)
            sendErrorResponse(message, context.chatId, context.threadId, emit)
            return false
        }

        return true
    }

    private suspend fun hasExistingSession(context: MessageContext): Boolean =
        runCatching { handlers.startHandler.hasExistingSession(context.chatId) }
            .onFailure { ex ->
                log.error(
                    "SESSION_CHECK_FAILED chatId={}, userId={}, error={}",
                    context.chatId,
                    context.userId,
                    ex.message,
                    ex,
                )
            }.getOrDefault(true)

    // validation + 메시지 수신 로깅
    private suspend fun validateMessageInput(message: InboundMessage): MessageContext? {
        val chatId = message.chatId
        val userId = message.userId.takeIf { it.isNotBlank() } ?: chatId
        val content = message.content

        log.sampled(
            key = "message.received",
            limit = LoggingConstants.LOG_SAMPLE_LIMIT_HIGH,
            windowMillis = LoggingConstants.LOG_SAMPLE_WINDOW_MS,
        ) {
            val truncated = content.take(MESSAGE_TRUNCATE_LENGTH)
            it.info(
                "MESSAGE_RECEIVED chatId={}, userId={}, msg='{}'",
                chatId,
                userId,
                truncated,
            )
        }

        if (chatId.isBlank() || content.isBlank()) {
            log.warn("MESSAGE_REJECTED chatId={}, userId={}, reason=MISSING_CONTENT", chatId, userId)
            return null
        }

        val command = commandParser.parse(content)

        return MessageContext(
            chatId = chatId,
            userId = userId,
            content = content,
            threadId = message.threadId,
            sender = message.sender,
            command = command,
        )
    }

    // 접근 권한 검증: 관리자 명령어는 allowlist 우회
    // 반환값: null이면 허용, String이면 차단
    private fun checkAccessControl(context: MessageContext): String? {
        val isAdminCommand =
            context.command is Command.AdminForceEnd ||
                context.command is Command.AdminClearAll ||
                context.command is Command.AdminRefreshCache ||
                context.command is Command.AdminRestartAll

        if (isAdminCommand) return null

        log.debug(
            "ACCESS_CHECK userId={}, blockedUserIds={}, enabled={}",
            context.userId,
            appProperties.access.blockedUserIds,
            appProperties.access.enabled,
        )

        return resolveAccessDenialReason(
            access = appProperties.access,
            userId = context.userId,
            chatId = context.chatId,
        )
    }

    private fun requiresExistingSession(command: Command?): Boolean =
        when (command) {
            null,
            is Command.Start,
            is Command.Help,
            is Command.HealthCheck,
            is Command.UserStats,
            is Command.AdminForceEnd,
            is Command.AdminClearAll,
            is Command.AdminRefreshCache,
            is Command.AdminRestartAll,
            is Command.AdminUsage,
            is Command.ModelInfo,
            -> false
            else -> true
        }

    private suspend fun shouldEnqueueImmediately(
        context: MessageContext,
        emit: suspend (OutboundMessage) -> Unit,
    ): Boolean {
        val hasPending =
            runCatching { queueCoordinator.hasPending(context.chatId) }
                .onFailure {
                    log.error(
                        "QUEUE_PENDING_CHECK_FAILED chatId={}, error={}",
                        context.chatId,
                        it.message,
                        it,
                    )
                }.getOrDefault(false)

        return when {
            hasPending -> {
                log.warn("MESSAGE_REJECTED_PENDING chatId={}, userId={}", context.chatId, context.userId)
                queueProcessor.enqueueAndNotify(
                    context.chatId,
                    context.userId,
                    context.content,
                    context.threadId,
                    context.sender,
                    emit,
                )
                true
            }
            processingLockService.isProcessing(context.chatId) -> {
                log.warn("MESSAGE_REJECTED_PROCESSING chatId={}, userId={}", context.chatId, context.userId)
                queueProcessor.enqueueAndNotify(
                    context.chatId,
                    context.userId,
                    context.content,
                    context.threadId,
                    context.sender,
                    emit,
                )
                true
            }
            else -> false
        }
    }
}

private suspend fun sendErrorResponse(
    message: String,
    chatId: String,
    threadId: String?,
    emit: suspend (OutboundMessage) -> Unit,
) {
    emit(OutboundMessage.Error(chatId, message, threadId))
}
