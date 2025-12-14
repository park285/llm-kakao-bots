package party.qwer.twentyq.service.riddle

import org.slf4j.LoggerFactory
import org.springframework.stereotype.Component
import party.qwer.twentyq.model.FiveScaleKo
import party.qwer.twentyq.model.RiddleSecret
import party.qwer.twentyq.redis.RiddleSessionRepository
import party.qwer.twentyq.service.dto.AnswerResult
import party.qwer.twentyq.service.dto.AnswerSource

/** 정답 검증: 정규화/LLM 순차 검증 및 성공 처리 */
@Component
class AnswerVerificationService(
    private val sessionRepo: RiddleSessionRepository,
    private val similarityVerifier: SimilarityVerifier,
    private val answerValidator: AnswerValidator,
    private val answerSuccessHandler: AnswerSuccessHandler,
) {
    companion object {
        private val log = LoggerFactory.getLogger(AnswerVerificationService::class.java)
    }

    /** 정답 시도 체크: "정답 X" 형식의 명시적 정답 시도 감지 */
    suspend fun checkAnswerAttempts(
        chatId: String,
        normalizedQuestion: String,
        secret: RiddleSecret,
        userId: String,
    ): AnswerResult? {
        val answerMatch = answerValidator.matchExplicitAnswer(normalizedQuestion)
        if (answerMatch != null) {
            log.info("EXPLICIT_ANSWER_ATTEMPT chatId={}", chatId)
            return handleAnswerAttempt(chatId, answerMatch, secret, userId)
        }

        return null
    }

    /** 정답 시도 검증: 정규화 → LLM 순차 검증 후 실패 시 오답 처리 */
    suspend fun handleAnswerAttempt(
        chatId: String,
        answerMatch: MatchResult,
        secret: RiddleSecret,
        userId: String,
    ): AnswerResult {
        val guess = answerMatch.groupValues[1].trim()
        val verifiers =
            listOf<suspend () -> AnswerResult?>(
                { verifyExactMatch(chatId, guess, secret, userId) },
                { verifyWithLlm(chatId, guess, secret, userId) },
            )

        var verified: AnswerResult? = null
        for (verifier in verifiers) {
            val candidate = verifier()
            if (candidate != null) {
                verified = candidate
                break
            }
        }

        return verified ?: handleWrongGuess(chatId, guess, secret.target, userId)
    }

    // 1단계: 정규화 exact 매칭
    private suspend fun verifyExactMatch(
        chatId: String,
        guess: String,
        secret: RiddleSecret,
        userId: String,
    ): AnswerResult? {
        val normalizedGuess = answerValidator.normalizeForEquality(guess)
        val normalizedTarget = answerValidator.normalizeForEquality(secret.target)

        return if (normalizedGuess == normalizedTarget) {
            log.info(
                "EXACT_MATCH chatId={}, guess='{}', target='{}', normalized='{}'",
                chatId,
                guess,
                secret.target,
                normalizedGuess,
            )
            createSuccessResult(chatId, secret, userId)
        } else {
            null
        }
    }

    // 2단계: LLM 검증
    private suspend fun verifyWithLlm(
        chatId: String,
        guess: String,
        secret: RiddleSecret,
        userId: String,
    ): AnswerResult? {
        val status = similarityVerifier.verifyHighSimilarity(secret.target, guess)

        return when (status) {
            VerifyAnswerResponse.RESPONSE_ACCEPT -> {
                log.info("LLM_VERIFIED chatId={}, guess='{}', target='{}'", chatId, guess, secret.target)
                createSuccessResult(chatId, secret, userId)
            }
            VerifyAnswerResponse.RESPONSE_CLOSE -> {
                log.info("LLM_CLOSE_CALL chatId={}, guess='{}', target='{}'", chatId, guess, secret.target)
                handleCloseCall(chatId, guess, secret.target, userId)
            }
            else -> null
        }
    }

    // 3단계: 오답 처리
    private suspend fun handleWrongGuess(
        chatId: String,
        guess: String,
        target: String,
        userId: String,
    ): AnswerResult {
        val normalizedGuess = answerValidator.normalizeForEquality(guess)
        val normalizedTarget = answerValidator.normalizeForEquality(target)

        log.info(
            "WRONG_GUESS chatId={}, guess='{}', target='{}', norm_guess='{}', norm_target='{}'",
            chatId,
            guess,
            target,
            normalizedGuess,
            normalizedTarget,
        )

        sessionRepo.addWrongGuess(chatId, guess, userId)

        return AnswerResult(
            scale = FiveScaleKo.ALWAYS_NO,
            source = AnswerSource.FALLBACK_DEFAULT,
            guardDegraded = false,
            isWrongGuess = true,
            guessedAnswer = guess,
        )
    }

    // Close call 처리: 거의 정답 (1단계 상위/하위 카테고리)
    private suspend fun handleCloseCall(
        chatId: String,
        guess: String,
        target: String,
        userId: String,
    ): AnswerResult {
        sessionRepo.addWrongGuess(chatId, guess, userId)

        log.info("CLOSE_CALL chatId={}, guess='{}', target='{}'", chatId, guess, target)

        return AnswerResult(
            scale = FiveScaleKo.ALWAYS_NO,
            source = AnswerSource.FALLBACK_DEFAULT,
            guardDegraded = false,
            isCloseCall = true,
            guessedAnswer = guess,
        )
    }

    suspend fun createSuccessResult(
        chatId: String,
        secret: RiddleSecret,
        userId: String,
    ): AnswerResult {
        val successMessage = answerSuccessHandler.handleSuccess(chatId, secret, userId)

        return AnswerResult(
            scale = FiveScaleKo.ALWAYS_YES,
            source = AnswerSource.FALLBACK_DEFAULT,
            guardDegraded = false,
            isCorrect = true,
            successMessage = successMessage,
        )
    }
}
