package party.qwer.twentyq.model

import org.springframework.data.annotation.Id
import org.springframework.data.relational.core.mapping.Column
import org.springframework.data.relational.core.mapping.Table
import java.time.Instant

/**
 * 게임 세션 단위 로그 (판당 1건)
 */
@Table("game_sessions")
data class GameSessionEntity(
    @Id
    @Column("id")
    val id: Long? = null,
    @Column("session_id")
    val sessionId: String,
    @Column("chat_id")
    val chatId: String,
    @Column("category")
    val category: String,
    @Column("result")
    val result: String,
    @Column("participant_count")
    val participantCount: Int,
    @Column("question_count")
    val questionCount: Int = 0,
    @Column("hint_count")
    val hintCount: Int = 0,
    @Column("completed_at")
    val completedAt: Instant = Instant.now(),
    @Column("created_at")
    val createdAt: Instant = Instant.now(),
)
