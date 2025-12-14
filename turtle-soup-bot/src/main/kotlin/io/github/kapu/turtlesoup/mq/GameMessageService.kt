package io.github.kapu.turtlesoup.mq

import io.github.kapu.turtlesoup.bridge.CommandParser
import io.github.kapu.turtlesoup.config.TimeConstants
import io.github.kapu.turtlesoup.models.Command
import io.github.kapu.turtlesoup.models.requiresLock
import io.github.kapu.turtlesoup.models.waitingMessageKey
import io.github.kapu.turtlesoup.mq.handler.GameCommandHandler
import io.github.kapu.turtlesoup.mq.models.InboundMessage
import io.github.kapu.turtlesoup.mq.models.OutboundMessage
import io.github.kapu.turtlesoup.redis.ProcessingLockService
import io.github.kapu.turtlesoup.rest.LlmRestClient
import io.github.kapu.turtlesoup.utils.AccessControl
import io.github.kapu.turtlesoup.utils.LockException
import io.github.kapu.turtlesoup.utils.MessageKeys
import io.github.kapu.turtlesoup.utils.MessageProvider
import io.github.kapu.turtlesoup.utils.TurtleSoupException
import io.github.kapu.turtlesoup.utils.isExpectedUserBehavior
import io.github.oshai.kotlinlogging.KLogger
import io.github.oshai.kotlinlogging.KotlinLogging
import kotlinx.coroutines.TimeoutCancellationException
import kotlinx.coroutines.withTimeout

/** MQ 메시지 처리 코디네이터 */
class GameMessageService(
    private val commandHandler: GameCommandHandler,
    private val messageSender: MessageSender,
    private val messageProvider: MessageProvider,
    private val publisher: ValkeyMQReplyPublisher,
    private val accessControl: AccessControl,
    private val commandParser: CommandParser,
    private val processingLockService: ProcessingLockService,
    private val queueProcessor: MessageQueueProcessor,
    private val restClient: LlmRestClient,
) {
    companion object {
        private val logger = KotlinLogging.logger {}
    }

    private fun shouldBlockWhenAiUnavailable(command: Command): Boolean =
        command !is Command.Unknown && command !is Command.Surrender && command !is Command.Agree

    /** 입력 메시지 처리 */
    suspend fun handleMessage(message: InboundMessage) {
        if (!isAccessAllowed(message)) return

        val command = parseCommand(message, commandParser, logger) ?: return
        if (shouldBlockWhenAiUnavailable(command) && !restClient.isHealthy()) {
            val unavailableText = messageProvider.get(MessageKeys.ERROR_AI_UNAVAILABLE)
            messageSender.sendFinal(message, unavailableText)
            return
        }
        dispatchCommand(message, command)
    }

    private suspend fun dispatchCommand(
        message: InboundMessage,
        command: Command,
    ) {
        val chatId = message.chatId

        when {
            !command.requiresLock -> handleSimpleCommand(message, command)
            processingLockService.isProcessing(chatId) -> enqueueMessage(message)
            else -> executeWithQueue(message, command, chatId)
        }
    }

    private suspend fun enqueueMessage(message: InboundMessage) {
        logger.info { "message_enqueued chat_id=${message.chatId} user_id=${message.userId}" }
        queueProcessor.enqueueAndNotify(
            chatId = message.chatId,
            userId = message.userId,
            content = message.content,
            threadId = message.threadId,
            sender = message.sender,
            emit = ::publish,
        )
    }

    /** emit function for queue processor */
    private suspend fun publish(outbound: OutboundMessage) {
        publisher.publish(outbound)
    }

    private fun isAccessAllowed(message: InboundMessage): Boolean {
        val denialReason = accessControl.getDenialReason(message.userId, message.chatId)
        if (denialReason != null) {
            logger.debug { "access_denied user_id=${message.userId} chat_id=${message.chatId}" }
            return false
        }
        return true
    }

    private suspend fun executeWithQueue(
        message: InboundMessage,
        command: Command,
        chatId: String,
    ) {
        try {
            processingLockService.startProcessing(chatId)
            executeCommand(message, command, chatId)
        } finally {
            processingLockService.finishProcessing(chatId)
            // 큐에 있는 메시지 처리
            queueProcessor.processQueuedMessages(chatId, ::publish)
        }
    }

    private suspend fun executeCommand(
        message: InboundMessage,
        command: Command,
        sessionId: String,
    ) {
        messageSender.sendWaiting(message, command)
        val result = runCommandWithTimeout(message, command) { msg, cmd -> processCommand(msg, cmd) }

        result.fold(
            onSuccess = { response -> messageSender.sendFinal(message, response) },
            onFailure = { error -> handleDirectFailure(message, sessionId, error) },
        )
    }

    private suspend fun processCommand(
        message: InboundMessage,
        command: Command,
    ): String = commandHandler.processCommand(message, command)

    private suspend fun handleSimpleCommand(
        message: InboundMessage,
        command: Command,
    ) {
        val response =
            when (command) {
                is Command.Help -> messageProvider.get(MessageKeys.HELP_MESSAGE)
                is Command.Unknown -> messageProvider.get(MessageKeys.ERROR_UNKNOWN_COMMAND)
                else -> return
            }
        messageSender.sendFinal(message, response)
    }

    /** 큐에서 처리된 명령 실행 (processQueuedMessages에서 호출) */
    suspend fun handleQueuedCommand(
        message: InboundMessage,
        command: Command,
        emit: suspend (OutboundMessage) -> Unit,
    ) {
        if (shouldBlockWhenAiUnavailable(command) && !restClient.isHealthy()) {
            val unavailableText = messageProvider.get(MessageKeys.ERROR_AI_UNAVAILABLE)
            emit(OutboundMessage.Final(message.chatId, unavailableText, message.threadId))
            return
        }

        command.waitingMessageKey?.let { key ->
            val waitingText = messageProvider.get(key)
            emit(OutboundMessage.Waiting(message.chatId, waitingText, message.threadId))
        }

        val result = runCommandWithTimeout(message, command) { msg, cmd -> processCommand(msg, cmd) }

        result.fold(
            onSuccess = { response ->
                emit(OutboundMessage.Final(message.chatId, response, message.threadId))
            },
            onFailure = { error ->
                handleQueuedFailure(message, error, emit)
            },
        )
    }

    private suspend fun handleDirectFailure(
        message: InboundMessage,
        sessionId: String,
        error: Throwable,
    ) {
        handleErrorByType(
            error = error,
            onTimeout = {
                logger.warn(it) { "ai_timeout session_id=$sessionId" }
                messageSender.sendError(message, ErrorMapping(MessageKeys.ERROR_AI_TIMEOUT))
            },
            onLock = {
                logger.warn(it) { "lock_failed session_id=$sessionId holder=${it.holderName}" }
                messageSender.sendLockError(message, it.holderName)
            },
            onDomain = {
                logDomainException(logger, sessionId, it)
                messageSender.sendError(message, ExceptionMapper.getErrorMapping(it))
            },
            onStateOrRedis = {
                logger.error(it) { "state_or_redis_error session_id=$sessionId" }
                messageSender.sendError(message, ErrorMapping(MessageKeys.ERROR_INTERNAL))
            },
            onUnexpected = {
                logger.error(it) { "unexpected_exception session_id=$sessionId type=${it::class.simpleName}" }
                messageSender.sendError(message, ErrorMapping(MessageKeys.ERROR_INTERNAL))
            },
        )
    }

    private suspend fun handleQueuedFailure(
        message: InboundMessage,
        error: Throwable,
        emit: suspend (OutboundMessage) -> Unit,
    ) {
        handleErrorByType(
            error = error,
            onTimeout = {
                logger.warn(it) { "ai_timeout session_id=${message.chatId}" }
                val errorText = messageProvider.get(MessageKeys.ERROR_AI_TIMEOUT)
                emit(OutboundMessage.Error(message.chatId, errorText, message.threadId))
            },
            onLock = {
                logger.warn(it) { "lock_failed session_id=${message.chatId} holder=${it.holderName}" }
                val errorText = messageProvider.get(MessageKeys.ERROR_INTERNAL)
                emit(OutboundMessage.Error(message.chatId, errorText, message.threadId))
            },
            onDomain = {
                val mapping = ExceptionMapper.getErrorMapping(it)
                val errorText = messageProvider.get(mapping.key, *mapping.params)
                emit(OutboundMessage.Error(message.chatId, errorText, message.threadId))
            },
            onStateOrRedis = {
                logger.error(it) { "queued_command_failed chat_id=${message.chatId}" }
                val errorText = messageProvider.get(MessageKeys.ERROR_INTERNAL)
                emit(OutboundMessage.Error(message.chatId, errorText, message.threadId))
            },
            onUnexpected = {
                logger.error(it) { "queued_command_failed chat_id=${message.chatId}" }
                val errorText = messageProvider.get(MessageKeys.ERROR_INTERNAL)
                emit(OutboundMessage.Error(message.chatId, errorText, message.threadId))
            },
        )
    }
}

private fun parseCommand(
    message: InboundMessage,
    commandParser: CommandParser,
    logger: KLogger,
): Command? {
    val command = commandParser.parse(message.content)
    if (command == null) {
        logger.debug { "message_ignored content='${message.content}'" }
    } else {
        logger.info { "command_parsed command=${command::class.simpleName} chat_id=${message.chatId}" }
    }
    return command
}

private inline fun handleErrorByType(
    error: Throwable,
    onTimeout: (TimeoutCancellationException) -> Unit,
    onLock: (LockException) -> Unit,
    onDomain: (TurtleSoupException) -> Unit,
    onStateOrRedis: (Throwable) -> Unit,
    onUnexpected: (Throwable) -> Unit,
) {
    when (error) {
        is TimeoutCancellationException -> onTimeout(error)
        is LockException -> onLock(error)
        is TurtleSoupException -> onDomain(error)
        is org.redisson.client.RedisException, is IllegalStateException -> onStateOrRedis(error)
        else -> onUnexpected(error)
    }
}

private suspend fun runCommandWithTimeout(
    message: InboundMessage,
    command: Command,
    executor: suspend (InboundMessage, Command) -> String,
): Result<String> {
    return runCatching {
        withTimeout(TimeConstants.AI_TIMEOUT_MILLIS) {
            executor(message, command)
        }
    }
}

private fun logDomainException(
    logger: KLogger,
    sessionId: String,
    error: TurtleSoupException,
) {
    if (error.isExpectedUserBehavior()) {
        logger.warn { "game_warning session_id=$sessionId error=${error::class.simpleName}" }
    } else {
        logger.error(error) { "game_exception session_id=$sessionId error=${error::class.simpleName}" }
    }
}
