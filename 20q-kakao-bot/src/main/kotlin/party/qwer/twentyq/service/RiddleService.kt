package party.qwer.twentyq.service

import org.slf4j.LoggerFactory
import org.springframework.stereotype.Service
import party.qwer.twentyq.api.dto.HintHistory
import party.qwer.twentyq.api.dto.QuestionHistory
import party.qwer.twentyq.api.dto.RiddleStatusResponse
import party.qwer.twentyq.model.RiddleCategory
import party.qwer.twentyq.redis.RiddleSessionRepository
import party.qwer.twentyq.service.dto.AnswerResult
import party.qwer.twentyq.service.exception.SessionNotFoundException
import party.qwer.twentyq.service.riddle.HintGenerator
import party.qwer.twentyq.service.riddle.HintPolicy
import party.qwer.twentyq.service.riddle.RiddleAnswerProcessor
import party.qwer.twentyq.service.riddle.RiddleCreator
import party.qwer.twentyq.service.riddle.RiddleTerminator

/** 수수께끼 게임 통합 관리 서비스 */
@Service
class RiddleService(
    private val sessionRepo: RiddleSessionRepository,
    private val riddleCreator: RiddleCreator,
    private val riddleAnswerProcessor: RiddleAnswerProcessor,
    private val hintGenerator: HintGenerator,
    private val riddleTerminator: RiddleTerminator,
) {
    companion object {
        private val log = LoggerFactory.getLogger(RiddleService::class.java)
    }

    suspend fun createRiddle(
        chatId: String,
        category: String? = null,
    ): String = riddleCreator.createRiddle(chatId, category)

    suspend fun generateHints(
        chatId: String,
        count: Int = 2,
    ): List<String> = hintGenerator.generateHints(chatId, count)

    suspend fun answer(
        chatId: String,
        question: String,
        userId: String,
        isChain: Boolean = false,
    ): AnswerResult = riddleAnswerProcessor.processAnswer(chatId, question, userId, isChain)

    suspend fun surrender(chatId: String): String = riddleTerminator.surrender(chatId)

    private fun buildHintHistories(history: List<QuestionHistory>): List<HintHistory> =
        history
            .filter { it.questionNumber < 0 }
            .mapIndexed { idx, h ->
                HintHistory(
                    hintNumber = idx + 1,
                    content = h.answer,
                )
            }

    suspend fun getStatus(chatId: String): RiddleStatusResponse {
        log.info("getStatus START chatId={}", chatId)

        if (sessionRepo.getQuiz(chatId = chatId) == null) throw SessionNotFoundException()

        val history = sessionRepo.getHistory(chatId)
        val questions = history.filter { it.questionNumber > 0 }
        val hintHistories = buildHintHistories(history)

        val questionCount = questions.size
        val hintCount = sessionRepo.getHintCount(chatId)
        val maxHints = HintPolicy.computeMaxHints(hintCount)

        val selectedCategoryRaw = sessionRepo.getSelectedCategory(chatId)
        val selectedCategoryKo = toKoreanCategory(selectedCategoryRaw)

        log.info(
            "getStatus SUCCESS chatId={}, questionCount={}, hintCount={}, maxHints={}",
            chatId,
            questionCount,
            hintCount,
            maxHints,
        )

        return RiddleStatusResponse(
            questionCount = questionCount,
            questions = questions,
            hints = hintHistories,
            hintCount = hintCount,
            maxHints = maxHints,
            selectedCategory = selectedCategoryKo,
        )
    }

    // 틀린 정답 조회 래퍼
    suspend fun getWrongGuesses(chatId: String): List<String> = sessionRepo.getWrongGuesses(chatId)

    private fun toKoreanCategory(selectedCategoryRaw: String?): String? =
        selectedCategoryRaw?.let {
            val cat = RiddleCategory.fromString(it)
            if (cat == RiddleCategory.ANY) null else cat.koreanName
        }

    suspend fun hasSession(chatId: String): Boolean = sessionRepo.getQuiz(chatId) != null

    suspend fun isHintAvailable(chatId: String): Boolean {
        if (!hasSession(chatId)) return false

        val hintCount = sessionRepo.getHintCount(chatId)
        val maxHints = HintPolicy.computeMaxHints(hintCount)

        val remaining = (maxHints - hintCount).coerceAtLeast(0)
        return remaining > 0
    }
}
