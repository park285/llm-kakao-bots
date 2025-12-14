package party.qwer.twentyq.repository

import kotlinx.coroutines.flow.Flow
import org.springframework.data.r2dbc.repository.Query
import org.springframework.data.repository.kotlin.CoroutineCrudRepository
import org.springframework.stereotype.Repository
import party.qwer.twentyq.model.GameSessionEntity
import java.time.Instant

/**
 * 게임 세션 로그 Repository
 */
@Repository
interface GameSessionRepository : CoroutineCrudRepository<GameSessionEntity, Long> {
    @Query(
        """
        SELECT * FROM game_sessions
        WHERE chat_id = :chatId
        AND completed_at >= :startTime
        AND completed_at < :endTime
        ORDER BY completed_at DESC
        """,
    )
    fun findByChatIdAndPeriod(
        chatId: String,
        startTime: Instant,
        endTime: Instant,
    ): Flow<GameSessionEntity>

    @Query(
        """
        SELECT * FROM game_sessions
        WHERE chat_id = :chatId
        ORDER BY completed_at DESC
        LIMIT :limit
        """,
    )
    fun findByChatId(
        chatId: String,
        limit: Int = 100,
    ): Flow<GameSessionEntity>

    @Query(
        """
        SELECT * FROM game_sessions
        WHERE session_id = :sessionId
        LIMIT 1
        """,
    )
    suspend fun findBySessionId(sessionId: String): GameSessionEntity?
}
