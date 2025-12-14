package io.github.kapu.turtlesoup.mq

import io.github.kapu.turtlesoup.bridge.CommandParser
import io.github.kapu.turtlesoup.config.MQConstants
import io.github.kapu.turtlesoup.mq.models.DequeueResult
import io.github.kapu.turtlesoup.mq.models.EnqueueResult
import io.github.kapu.turtlesoup.mq.models.OutboundMessage
import io.github.kapu.turtlesoup.mq.models.PendingMessage
import io.github.kapu.turtlesoup.redis.LockManager
import io.github.kapu.turtlesoup.redis.ProcessingLockService
import io.github.kapu.turtlesoup.utils.MessageKeys
import io.github.kapu.turtlesoup.utils.MessageProvider
import io.github.oshai.kotlinlogging.KotlinLogging

/** 큐잉된 메시지 처리기 */
class MessageQueueProcessor(
    private val queueCoordinator: MessageQueueCoordinator,
    private val lockManager: LockManager,
    private val processingLockService: ProcessingLockService,
    private val messageProvider: MessageProvider,
    private val notifier: MessageQueueNotifier,
    private val commandParser: CommandParser,
    private val commandExecutor: CommandExecutor,
) {
    companion object {
        private val logger = KotlinLogging.logger {}
    }

    /** 메시지 큐잉 및 알림 */
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

        val userName =
            pendingMessage.displayName(
                chatId,
                messageProvider.get(MessageKeys.USER_ANONYMOUS),
            )
        val message = buildQueueMessage(result, chatId, userName, content)

        emit(OutboundMessage.Waiting(chatId, message, threadId))
    }

    private suspend fun buildQueueMessage(
        result: EnqueueResult,
        chatId: String,
        userName: String,
        content: String,
    ): String {
        return when (result) {
            EnqueueResult.SUCCESS -> {
                val rawDetails = queueCoordinator.getQueueDetails(chatId)
                val queueDetails =
                    rawDetails.takeIf { it.isNotBlank() }
                        ?: messageProvider.get(MessageKeys.QUEUE_EMPTY)
                messageProvider.get(
                    MessageKeys.QUEUE_MESSAGE_QUEUED,
                    "user" to userName,
                    "queueDetails" to queueDetails,
                )
            }
            EnqueueResult.QUEUE_FULL -> messageProvider.get(MessageKeys.QUEUE_FULL)
            EnqueueResult.DUPLICATE ->
                messageProvider.get(
                    MessageKeys.QUEUE_ALREADY_QUEUED,
                    "user" to userName,
                    "content" to content,
                )
        }
    }

    /** 큐에 있는 메시지 순차 처리 */
    suspend fun processQueuedMessages(
        chatId: String,
        emit: suspend (OutboundMessage) -> Unit,
    ) {
        var iterations = 0
        while (iterations < MQConstants.MAX_QUEUE_PROCESS_ITERATIONS) {
            iterations++
            val dequeueResult =
                runCatching { queueCoordinator.dequeue(chatId) }
                    .onFailure { ex ->
                        logger.warn(ex) {
                            "queue_dequeue_failed chat_id=$chatId iteration=$iterations"
                        }
                    }
                    .getOrNull() ?: return

            when (dequeueResult) {
                is DequeueResult.Empty -> return
                is DequeueResult.Exhausted -> {
                    // 루프 제한 도달: 즉시 다음 iteration에서 재시도
                    logger.debug { "dequeue_exhausted chat_id=$chatId iteration=$iterations" }
                    continue
                }
                is DequeueResult.Success -> {
                    val continueProcessing =
                        processSingleQueuedMessage(chatId, dequeueResult.message, emit)
                    if (!continueProcessing) return
                }
            }
        }

        if (iterations >= MQConstants.MAX_QUEUE_PROCESS_ITERATIONS) {
            logger.warn {
                "queue_processing_limit_reached chat_id=$chatId max_iterations=$iterations"
            }
        }
    }

    private suspend fun processSingleQueuedMessage(
        chatId: String,
        pending: PendingMessage,
        emit: suspend (OutboundMessage) -> Unit,
    ): Boolean {
        logger.debug { "processing_queued_message chat_id=$chatId user_id=${pending.userId}" }

        notifier.notifyProcessingStart(chatId, pending, emit)

        val holderName = pending.sender ?: pending.userId

        val lockAcquired =
            runCatching {
                lockManager.withLock(chatId, holderName) {
                    processingLockService.startProcessing(chatId)
                    executeWithErrorHandling(chatId, pending, emit)
                    processingLockService.finishProcessing(chatId)
                }
            }.getOrNull()

        if (lockAcquired == null) {
            return handleLockAcquisitionFailure(chatId, pending, emit)
        }

        return true
    }

    private suspend fun executeWithErrorHandling(
        chatId: String,
        pending: PendingMessage,
        emit: suspend (OutboundMessage) -> Unit,
    ) {
        val executionResult =
            runCatching {
                commandExecutor.execute(
                    chatId,
                    pending.userId,
                    pending.content,
                    pending.threadId,
                    pending.sender,
                    emit,
                )
            }
        executionResult.exceptionOrNull()?.let { ex ->
            notifier.notifyError(chatId, pending, ex, emit)
        }
    }

    private suspend fun handleLockAcquisitionFailure(
        chatId: String,
        pending: PendingMessage,
        emit: suspend (OutboundMessage) -> Unit,
    ): Boolean {
        logger.warn { "queue_processing_lock_failed chat_id=$chatId user_id=${pending.userId}" }

        when (val reEnqueueResult = queueCoordinator.enqueue(chatId, pending)) {
            EnqueueResult.SUCCESS -> {
                notifier.notifyRetry(chatId, pending, emit)
                logger.info { "queue_requeue_success chat_id=$chatId user_id=${pending.userId}" }
            }
            EnqueueResult.DUPLICATE -> {
                notifier.notifyDuplicate(chatId, pending, emit)
                logger.info {
                    "queue_requeue_duplicate chat_id=$chatId user_id=${pending.userId}"
                }
            }
            EnqueueResult.QUEUE_FULL -> {
                notifier.notifyFailed(chatId, pending, emit)
                logger.warn { "queue_requeue_full chat_id=$chatId user_id=${pending.userId}" }
            }
        }
        return false
    }

    /** 명령어 실행 인터페이스 */
    fun interface CommandExecutor {
        suspend fun execute(
            chatId: String,
            userId: String,
            content: String,
            threadId: String?,
            sender: String?,
            emit: suspend (OutboundMessage) -> Unit,
        )
    }
}
