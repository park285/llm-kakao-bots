package party.qwer.twentyq.repository

import org.springframework.data.r2dbc.repository.Query
import org.springframework.data.repository.kotlin.CoroutineCrudRepository
import org.springframework.stereotype.Repository
import party.qwer.twentyq.model.UserNicknameMapEntity
import java.time.Instant

@Repository
interface UserNicknameMapRepository : CoroutineCrudRepository<UserNicknameMapEntity, Long> {
    @Query(
        """
        INSERT INTO user_nickname_map (chat_id, user_id, last_sender, last_seen_at, created_at)
        VALUES (:chatId, :userId, :lastSender, :lastSeenAt, NOW())
        ON CONFLICT (chat_id, user_id)
        DO UPDATE SET
            last_sender = EXCLUDED.last_sender,
            last_seen_at = EXCLUDED.last_seen_at
        """,
    )
    suspend fun upsertNickname(
        chatId: String,
        userId: String,
        lastSender: String,
        lastSeenAt: Instant,
    )

    @Query(
        """
        SELECT * FROM user_nickname_map
        WHERE chat_id = :chatId
        AND LOWER(last_sender) = LOWER(:nickname)
        ORDER BY last_seen_at DESC
        LIMIT 1
        """,
    )
    suspend fun findLatestByNickname(
        chatId: String,
        nickname: String,
    ): UserNicknameMapEntity?
}
