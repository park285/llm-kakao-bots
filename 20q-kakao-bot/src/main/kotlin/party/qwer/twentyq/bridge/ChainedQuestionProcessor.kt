package party.qwer.twentyq.bridge

import kotlinx.coroutines.flow.Flow
import org.slf4j.LoggerFactory
import org.springframework.stereotype.Component
import party.qwer.twentyq.bridge.handlers.MessageHandlers
import party.qwer.twentyq.logging.LoggingExtensions.sampled
import party.qwer.twentyq.logging.warnL
import party.qwer.twentyq.model.OutboundMessage
import party.qwer.twentyq.model.PendingMessage
import party.qwer.twentyq.rest.LlmAvailabilityGuard
import party.qwer.twentyq.service.exception.GameMessageKeys
import party.qwer.twentyq.util.common.extensions.chunkedByLines
import party.qwer.twentyq.util.game.GameMessageProvider
import party.qwer.twentyq.util.logging.LoggingConstants

/**
 * 체인 질문 처리 담당 클래스
 */
@Component
class ChainedQuestionProcessor(
    private val handlers: MessageHandlers,
    private val messageProvider: GameMessageProvider,
    private val exceptionHandler: MessageExceptionHandler,
    private val messageEmitter: MessageEmitter,
    private val llmAvailabilityGuard: LlmAvailabilityGuard,
) {
    private val log = LoggerFactory.getLogger(ChainedQuestionProcessor::class.java)

    suspend fun processChainQuestions(
        chatId: String,
        pending: PendingMessage,
        emit: suspend (OutboundMessage) -> Unit,
    ) {
        if (!llmAvailabilityGuard.isAvailable()) {
            val unavailableMessage = messageProvider.get(GameMessageKeys.AI_UNAVAILABLE)
            messageEmitter.sendChunkedReply(unavailableMessage, chatId, pending.threadId, emit)
            log.warn("CHAIN_BLOCKED_LLM_UNAVAILABLE chatId={}, userId={}", chatId, pending.userId)
            return
        }

        val questions =
            pending.batchQuestions ?: run {
                log.warn("CHAIN_QUESTIONS_NULL chatId={}, userId={}", chatId, pending.userId)
                return
            }

        // 각 질문 순차 처리
        questions.forEachIndexed { index, question ->
            processSingleChainQuestion(chatId, question, pending.userId, pending.sender, index)
        }

        // Status 메시지 조회 및 전송
        val statusMessages =
            runCatching {
                handlers.statusHandler.handleSeparated(chatId)
            }.getOrElse { ex ->
                log.warnL { "STATUS_FALLBACK chatId=$chatId reason=${ex.message}" }
                listOf(messageProvider.get("error.session_not_found"))
            }
        statusMessages.forEach { message ->
            messageEmitter.sendChunkedReply(message, chatId, pending.threadId, emit)
        }
    }

    // 단일 체인 질문 처리
    private suspend fun processSingleChainQuestion(
        chatId: String,
        question: String,
        userId: String,
        sender: String?,
        index: Int,
    ): String {
        val response =
            runCatching {
                handlers.askHandler.handle(chatId, question, userId, sender, isChain = true)
            }.fold(
                onSuccess = { it },
                onFailure = { ex ->
                    val exception = ex as? Exception ?: Exception(ex.message)
                    exceptionHandler.getMessageForException(exception)
                },
            )

        log.sampled(
            key = "chain.question.processed",
            limit = LoggingConstants.LOG_SAMPLE_LIMIT_LOW,
            windowMillis = LoggingConstants.LOG_SAMPLE_WINDOW_LONG,
        ) {
            it.debug(
                "CHAIN_QUESTION_PROCESSED chatId={}, userId={}, index={}, question={}",
                chatId,
                userId,
                index + 1,
                question,
            )
        }

        return response
    }
}
