package party.qwer.twentyq.bridge.handlers

import org.springframework.stereotype.Component
import party.qwer.twentyq.model.ChainCondition
import party.qwer.twentyq.model.Command
import party.qwer.twentyq.model.FiveScaleKo
import party.qwer.twentyq.model.PendingMessage
import party.qwer.twentyq.mq.MessageQueueCoordinator
import party.qwer.twentyq.service.RiddleService
import party.qwer.twentyq.service.dto.AnswerResult
import party.qwer.twentyq.service.exception.GameMessageKeys
import party.qwer.twentyq.service.exception.InvalidQuestionException
import party.qwer.twentyq.util.common.formatting.UserIdFormatter
import party.qwer.twentyq.util.game.GameMessageProvider

/**
 * 체인 질문 핸들러
 *
 * 쉼표로 구분된 여러 질문을 순차적으로 처리
 * - 첫 질문: 즉시 실행
 * - 나머지 질문: PendingMessageStore에 큐잉
 */
@Component
class ChainedQuestionHandler(
    private val riddleService: RiddleService,
    private val queueCoordinator: MessageQueueCoordinator,
    private val messageProvider: GameMessageProvider,
) {
    // 체인 질문 대기열 사전 준비 (즉시 큐잉 + 안내 메시지 반환)
    suspend fun prepareChainQueue(
        chatId: String,
        userId: String,
        sender: String?,
        questions: List<String>,
    ): String? {
        if (questions.size <= 1) {
            return null
        }

        val displaySender =
            UserIdFormatter.displayName(userId, sender, chatId, messageProvider.get("user.anonymous"))

        val remainingQuestions = questions.drop(1)

        // 나머지 질문들을 즉시 큐에 추가 (optimistic enqueuing)
        val chainMessage =
            PendingMessage(
                userId = userId,
                content = "", // 체인 메시지는 content 사용 안 함
                threadId = null,
                sender = displaySender,
                isChainBatch = true,
                batchQuestions = remainingQuestions,
            )

        queueCoordinator.enqueue(chatId, chainMessage)

        // lock.message_queued 템플릿 형식으로 큐 안내 메시지 생성
        val queueDetails =
            remainingQuestions
                .mapIndexed { index, question ->
                    messageProvider.get(
                        "chain.queue_item",
                        "index" to (index + 1),
                        "question" to question,
                    )
                }.joinToString("\n")

        return messageProvider.get(
            "lock.message_queued",
            "user" to displaySender,
            "queueDetails" to queueDetails,
        )
    }

    suspend fun handle(
        chatId: String,
        command: Command.ChainedQuestion,
        userId: String,
        sender: String?,
    ): String {
        val questions = command.questions
        if (questions.isEmpty()) {
            throw InvalidQuestionException(
                messageProvider.get("error.invalid_question"),
            )
        }

        val firstQuestion = questions.first()
        val result = riddleService.answer(chatId, firstQuestion, userId)

        val shouldContinue = evaluateCondition(command.condition, result.scale)
        val hasRemainingQuestions = questions.size > 1

        // Optimistic queueing: 조건 불만족 시 skip flag 설정
        if (!shouldContinue && hasRemainingQuestions) {
            queueCoordinator.setChainSkipFlag(chatId, userId)
        }

        return buildResponseWithSkipNotification(result, questions, shouldContinue, chatId, userId, sender)
    }

    private fun evaluateCondition(
        condition: ChainCondition,
        scale: FiveScaleKo,
    ): Boolean =
        when (condition) {
            ChainCondition.ALWAYS -> true
            ChainCondition.IF_TRUE ->
                scale in
                    listOf(
                        FiveScaleKo.ALWAYS_YES,
                        FiveScaleKo.MOSTLY_YES,
                    )
        }

    private fun buildResponseWithSkipNotification(
        result: AnswerResult,
        questions: List<String>,
        shouldContinue: Boolean,
        chatId: String,
        userId: String,
        sender: String?,
    ): String {
        val baseResponse = buildBaseResponse(result, chatId, userId, sender)
        if (!shouldContinue && questions.size > 1) {
            val skippedQuestions = questions.drop(1)
            val skippedList = skippedQuestions.joinToString(", ")
            val skipNotification =
                messageProvider.get(
                    GameMessageKeys.CHAIN_CONDITION_NOT_MET,
                    "questions" to skippedList,
                )
            return "$baseResponse\n\n$skipNotification"
        }
        return baseResponse
    }

    private fun buildBaseResponse(
        result: AnswerResult,
        chatId: String,
        userId: String,
        sender: String?,
    ): String =
        when {
            result.isCorrect -> {
                result.successMessage ?: messageProvider.get("answer.correct_default")
            }
            result.isWrongGuess -> {
                val displayName =
                    UserIdFormatter.displayName(
                        userId,
                        sender,
                        chatId,
                        messageProvider.get("user.anonymous"),
                    )
                messageProvider.get(
                    GameMessageKeys.ANSWER_WRONG_GUESS,
                    "nickname" to displayName,
                    "guess" to (result.guessedAnswer ?: ""),
                )
            }
            result.guardDegraded -> {
                messageProvider.get("error.invalid_question.default")
            }
            else -> {
                FiveScaleKo
                    .token(result.scale)
            }
        }
}
