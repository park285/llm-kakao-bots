package party.qwer.twentyq.service

import org.slf4j.LoggerFactory
import org.springframework.stereotype.Service
import party.qwer.twentyq.model.GameSessionEntity
import party.qwer.twentyq.repository.GameSessionRepository
import java.time.Instant
import java.util.UUID

/**
 * 게임 세션 단위 로그 기록기 (판당 1건)
 */
@Service
class GameSessionRecorder(
    private val gameSessionRepository: GameSessionRepository,
) {
    companion object {
        private val log = LoggerFactory.getLogger(GameSessionRecorder::class.java)
    }

    suspend fun recordSession(
        sessionId: String?,
        chatId: String,
        category: String,
        result: GameResult,
        participantCount: Int,
        questionCount: Int,
        hintCount: Int,
        completedAt: Instant = Instant.now(),
    ) {
        val resolvedSessionId = sessionId.orEmpty().ifBlank { generateFallbackSessionId(chatId) }
        val existing = gameSessionRepository.findBySessionId(resolvedSessionId)
        if (existing != null) {
            log.debug("SESSION_LOG_DUP_SKIP sessionId={}, chatId={}", resolvedSessionId, chatId)
            return
        }

        val entity =
            GameSessionEntity(
                sessionId = resolvedSessionId,
                chatId = chatId,
                category = category,
                result = result.name,
                participantCount = participantCount,
                questionCount = questionCount,
                hintCount = hintCount,
                completedAt = completedAt,
            )
        gameSessionRepository.save(entity)
        log.info(
            "SESSION_LOG_RECORDED chatId={}, sessionId={}, result={}, participants={}, questions={}, hints={}",
            chatId,
            resolvedSessionId,
            result,
            participantCount,
            questionCount,
            hintCount,
        )
    }

    private fun generateFallbackSessionId(chatId: String): String = "$chatId:${UUID.randomUUID()}"
}
