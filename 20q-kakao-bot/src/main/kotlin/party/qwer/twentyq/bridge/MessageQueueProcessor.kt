package party.qwer.twentyq.bridge

import org.slf4j.LoggerFactory
import party.qwer.twentyq.logging.LoggingExtensions.sampled
import party.qwer.twentyq.model.OutboundMessage
import party.qwer.twentyq.model.PendingMessage
import party.qwer.twentyq.model.requiresWriteLock
import party.qwer.twentyq.mq.MessageQueueCoordinator
import party.qwer.twentyq.mq.queue.EnqueueResult
import party.qwer.twentyq.service.exception.GameMessageKeys
import party.qwer.twentyq.util.game.extensions.displayName
import party.qwer.twentyq.util.logging.LoggingConstants

internal class MessageQueueProcessor(
    private val queueCoordinator: MessageQueueCoordinator,
    private val lockingSupport: LockingSupport,
    private val messagingSupport: MessagingSupport,
    private val commandParser: CommandParser,
    private val commandExecutor: MessageCommandExecutor,
    private val notifier: MessageQueueNotifier,
    private val maxQueueProcessIterations: Int,
) {
    companion object {
        private val log = LoggerFactory.getLogger(MessageQueueProcessor::class.java)
    }

    suspend fun enqueueAndNotify(
        chatId: String,
        userId: String,
        content: String,
        threadId: String?,
        sender: String?,
        emit: suspend (OutboundMessage) -> Unit,
    ) {
        val pendingMessage = PendingMessage(userId, content, threadId, sender)
        val result = queueCoordinator.enqueue(chatId, pendingMessage)

        val userName = pendingMessage.displayName(chatId, messagingSupport.messageProvider.get("user.anonymous"))
        val message =
            when (result) {
                EnqueueResult.SUCCESS -> {
                    val rawDetails = queueCoordinator.getQueueDetails(chatId)
                    val queueDetails =
                        rawDetails.takeIf { it.isNotBlank() } ?: messagingSupport.messageProvider.get("queue.empty")
                    messagingSupport.messageProvider.get(
                        "lock.message_queued",
                        "user" to userName,
                        "queueDetails" to queueDetails,
                    )
                }

                EnqueueResult.QUEUE_FULL -> messagingSupport.messageProvider.get("lock.queue_full")
                EnqueueResult.DUPLICATE ->
                    messagingSupport.messageProvider.get(
                        "lock.already_queued",
                        "user" to userName,
                        "content" to content,
                    )
            }

        emit(OutboundMessage.Waiting(chatId, message, threadId))
    }

    suspend fun processQueuedMessages(
        chatId: String,
        emit: suspend (OutboundMessage) -> Unit,
    ) {
        var iterations = 0
        while (iterations < maxQueueProcessIterations) {
            iterations++
            val pending =
                runCatching { queueCoordinator.dequeue(chatId) }
                    .onFailure { ex ->
                        log.warn(
                            "QUEUE_DEQUEUE_FAILED chatId={}, iteration={}, error={}",
                            chatId,
                            iterations,
                            ex.message,
                            ex,
                        )
                    }.getOrNull() ?: return

            val continueProcessing = processSingleQueuedMessage(chatId, pending, emit)
            if (!continueProcessing) return
        }

        if (iterations >= maxQueueProcessIterations) {
            log.warn("QUEUE_PROCESSING_LIMIT_REACHED chatId={}, maxIterations={}", chatId, maxQueueProcessIterations)
        }
    }

    private suspend fun processSingleQueuedMessage(
        chatId: String,
        pending: PendingMessage,
        emit: suspend (OutboundMessage) -> Unit,
    ): Boolean {
        log.sampled(
            key = "queue.process",
            limit = LoggingConstants.LOG_SAMPLE_LIMIT_LOW,
            windowMillis = LoggingConstants.LOG_SAMPLE_WINDOW_LONG,
        ) {
            it.debug("PROCESSING_QUEUED_MESSAGE chatId={}, userId={}", chatId, pending.userId)
        }

        if (shouldSkipChainBatch(chatId, pending, emit)) return true

        notifier.notifyProcessingStart(chatId, pending, emit)

        val command = commandParser.parse(pending.content)
        val requiresWrite = command?.requiresWriteLock() ?: true

        val lockAcquired =
            lockingSupport.lockCoordinator.withLock(chatId, pending.userId, requiresWrite) {
                lockingSupport.processingLockService.startProcessing(chatId)

                val executionResult = runCatching { executeQueuedMessageProcessing(chatId, pending, emit) }
                executionResult.exceptionOrNull()?.let { ex ->
                    notifier.notifyError(chatId, pending, ex, emit)
                }

                lockingSupport.processingLockService.finishProcessing(chatId)
            }

        if (lockAcquired == null) {
            return handleLockAcquisitionFailure(chatId, pending, emit)
        }

        return true
    }

    // lock 획득 실패 시 재큐잉 처리
    private suspend fun handleLockAcquisitionFailure(
        chatId: String,
        pending: PendingMessage,
        emit: suspend (OutboundMessage) -> Unit,
    ): Boolean {
        log.warn("QUEUE_PROCESSING_LOCK_FAILED chatId={}, userId={}", chatId, pending.userId)

        when (val reEnqueueResult = queueCoordinator.enqueue(chatId, pending)) {
            EnqueueResult.SUCCESS -> {
                notifier.notifyRetry(chatId, pending, emit)
                log.info("QUEUE_REQUEUE_SUCCESS chatId={}, userId={}", chatId, pending.userId)
            }
            EnqueueResult.DUPLICATE -> {
                notifier.notifyDuplicate(chatId, pending, emit)
                log.info("QUEUE_REQUEUE_DUPLICATE chatId={}, userId={}", chatId, pending.userId)
            }
            EnqueueResult.QUEUE_FULL -> {
                notifier.notifyFailed(chatId, pending, emit)
                log.warn("QUEUE_REQUEUE_FULL chatId={}, userId={}", chatId, pending.userId)
            }
        }
        return false
    }

    private suspend fun shouldSkipChainBatch(
        chatId: String,
        pending: PendingMessage,
        emit: suspend (OutboundMessage) -> Unit,
    ): Boolean {
        if (!pending.isChainBatch || pending.batchQuestions == null) return false

        val shouldSkip = queueCoordinator.hasChainSkipFlag(chatId, pending.userId)
        if (!shouldSkip) return false

        log.info("CHAIN_BATCH_SKIPPED chatId={}, userId={} (condition not met)", chatId, pending.userId)
        val skipMessage =
            messagingSupport.messageProvider.get(
                GameMessageKeys.CHAIN_CONDITION_NOT_MET,
                "questions" to pending.batchQuestions.joinToString(", "),
            )
        emit(OutboundMessage.Final(chatId, skipMessage, pending.threadId))
        return true
    }

    private suspend fun executeQueuedMessageProcessing(
        chatId: String,
        pending: PendingMessage,
        emit: suspend (OutboundMessage) -> Unit,
    ) {
        if (pending.isChainBatch && pending.batchQuestions != null) {
            // skip flag는 processSingleQueuedMessage에서 이미 확인했으므로 바로 실행
            commandExecutor.processChainQuestions(chatId, pending, emit)
        } else {
            commandExecutor.processQueuedCommand(
                pending.content,
                chatId,
                pending.userId,
                pending.threadId,
                pending.sender,
                emit,
            )
        }
    }
}
