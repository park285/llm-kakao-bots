package party.qwer.twentyq.repository

import kotlinx.coroutines.flow.Flow
import org.springframework.data.r2dbc.repository.Query
import org.springframework.data.repository.kotlin.CoroutineCrudRepository
import org.springframework.stereotype.Repository
import party.qwer.twentyq.model.GameLogEntity
import java.time.Instant

/**
 * 게임 로그 Repository
 */
@Repository
interface GameLogRepository : CoroutineCrudRepository<GameLogEntity, Long> {
    /**
     * 특정 방의 기간별 게임 로그 조회
     */
    @Query(
        """
        SELECT * FROM game_logs
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
    ): Flow<GameLogEntity>

    /**
     * 특정 방의 전체 게임 로그 조회
     */
    @Query(
        """
        SELECT * FROM game_logs
        WHERE chat_id = :chatId
        ORDER BY completed_at DESC
        LIMIT :limit
        """,
    )
    fun findByChatId(
        chatId: String,
        limit: Int = 100,
    ): Flow<GameLogEntity>

    /**
     * 동일 사용자 최신 sender 조회 (비어있지 않은 값만)
     */
    @Query(
        """
        SELECT sender FROM game_logs
        WHERE chat_id = :chatId AND user_id = :userId AND sender <> ''
        ORDER BY completed_at DESC
        LIMIT 1
        """,
    )
    suspend fun findLatestSender(
        chatId: String,
        userId: String,
    ): String?

    /**
     * 닉네임으로 userId 조회 (전적 조회용 폴백)
     */
    @Query(
        """
        SELECT * FROM game_logs
        WHERE chat_id = :chatId AND LOWER(sender) = LOWER(:sender)
        ORDER BY completed_at DESC
        LIMIT 1
        """,
    )
    suspend fun findByChatIdAndSender(
        chatId: String,
        sender: String,
    ): GameLogEntity?
}
