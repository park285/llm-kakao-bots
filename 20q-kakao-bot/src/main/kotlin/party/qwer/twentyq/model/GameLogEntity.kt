package party.qwer.twentyq.model

import org.springframework.data.annotation.Id
import org.springframework.data.relational.core.mapping.Column
import org.springframework.data.relational.core.mapping.Table
import java.time.Instant

/**
 * 게임 완료 로그 Entity (기간별 통계용)
 */
@Table("game_logs")
data class GameLogEntity(
    @Id
    @Column("id")
    val id: Long? = null,
    @Column("chat_id")
    val chatId: String,
    @Column("user_id")
    val userId: String,
    @Column("sender")
    val sender: String = "",
    @Column("category")
    val category: String,
    @Column("question_count")
    val questionCount: Int = 0,
    @Column("hint_count")
    val hintCount: Int = 0,
    @Column("wrong_guess_count")
    val wrongGuessCount: Int = 0,
    @Column("result")
    val result: String,
    @Column("target")
    val target: String? = null,
    @Column("completed_at")
    val completedAt: Instant = Instant.now(),
    @Column("created_at")
    val createdAt: Instant = Instant.now(),
)
