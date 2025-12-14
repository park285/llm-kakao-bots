package party.qwer.twentyq.service.riddle

import org.slf4j.LoggerFactory
import org.springframework.stereotype.Service
import party.qwer.twentyq.model.SecretForHint
import party.qwer.twentyq.redis.RiddleSessionRepository
import party.qwer.twentyq.rest.TwentyQRestClient
import party.qwer.twentyq.service.exception.HintLimitExceededException
import party.qwer.twentyq.service.exception.SessionNotFoundException
import party.qwer.twentyq.service.riddle.model.HintSessionContext
import party.qwer.twentyq.util.common.json.parseDescriptionOrNull
import tools.jackson.databind.ObjectMapper

/** 힌트 생성 서비스 */
@Service
class HintGenerator(
    private val sessionRepo: RiddleSessionRepository,
    private val restClient: TwentyQRestClient,
    private val objectMapper: ObjectMapper,
) {
    companion object {
        private val log = LoggerFactory.getLogger(HintGenerator::class.java)
    }

    suspend fun generateHints(
        chatId: String,
        count: Int = 1,
    ): List<String> {
        log.info("generateHints START chatId={}, count={}", chatId, count)

        val context = loadSessionContext(chatId)
        ensureRemainingOrThrow(context)

        // REST 서버에서 힌트 생성 (TOON 변환, 프롬프트 구성, JSON 파싱 모두 서버에서 처리)
        val response =
            restClient.generateHints(
                target = context.secretForHint.target,
                category = context.selectedCategory,
                details = context.secretForHint.details,
            )

        if (response.isError) {
            log.error("generateHints MCP_ERROR chatId={}, error={}", chatId, response.errorMessage)
            return emptyList()
        }

        val hints = response.hints.take(count)
        if (hints.isEmpty()) {
            log.warn("generateHints EMPTY_HINTS chatId={}", chatId)
            return emptyList()
        }

        return saveHints(chatId, context, hints, response.thoughtSignature)
    }

    private suspend fun loadSessionContext(chatId: String): HintSessionContext {
        val secret =
            sessionRepo.getSecret(chatId = chatId)
                ?: throw SessionNotFoundException()

        val currentHintCount = sessionRepo.getHintCount(chatId)
        val maxHints = HintPolicy.computeMaxHints(currentHintCount)
        val history = sessionRepo.getHistory(chatId)
        val questionCount = history.count { it.questionNumber > 0 }

        val effectiveCategory = sessionRepo.getSelectedCategory(chatId) ?: secret.category
        val details = objectMapper.parseDescriptionOrNull(secret.description)

        val secretForHint =
            SecretForHint(
                target = secret.target,
                category = secret.category,
                details = details,
            )

        return HintSessionContext(
            chatId = chatId,
            secretForHint = secretForHint,
            secretTarget = secret.target,
            currentHintCount = currentHintCount,
            maxHints = maxHints,
            questionCount = questionCount,
            selectedCategory = effectiveCategory,
        )
    }

    private fun ensureRemainingOrThrow(context: HintSessionContext) {
        val remaining = (context.maxHints - context.currentHintCount).coerceAtLeast(0)
        if (remaining <= 0) {
            log.warn(
                "generateHints LIMIT_EXCEEDED chatId={}, currentCount={}, maxHints={}",
                context.chatId,
                context.currentHintCount,
                context.maxHints,
            )
            throw HintLimitExceededException(
                maxHints = context.maxHints,
                hintCount = context.currentHintCount,
                remaining = remaining,
            )
        }
        log.info("generateHints HINT_BUDGET chatId={}, remaining={}", context.chatId, remaining)
    }

    private suspend fun saveHints(
        chatId: String,
        context: HintSessionContext,
        hints: List<String>,
        thoughtSignature: String?,
    ): List<String> {
        hints.forEachIndexed { index, hint ->
            val hintNumber = context.currentHintCount + index + 1
            sessionRepo.incrementHintCount(chatId)
            sessionRepo.addHistory(
                chatId = chatId,
                questionNumber = -hintNumber,
                question = "힌트 #$hintNumber",
                answer = hint,
                thoughtSignature = thoughtSignature,
                userId = null,
            )
            log.info("HINT_SAVE chatId={}, count={}, max={}", chatId, hintNumber, context.maxHints)
        }

        log.info("generateHints SUCCESS chatId={}, hintsGenerated={}", chatId, hints.size)
        return hints
    }
}
