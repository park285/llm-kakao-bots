package party.qwer.twentyq.model

import org.springframework.data.annotation.Id
import org.springframework.data.relational.core.mapping.Column
import org.springframework.data.relational.core.mapping.Table
import java.time.Instant

@Table("user_nickname_map")
data class UserNicknameMapEntity(
    @Id
    @Column("id")
    val id: Long? = null,
    @Column("chat_id")
    val chatId: String,
    @Column("user_id")
    val userId: String,
    @Column("last_sender")
    val lastSender: String,
    @Column("last_seen_at")
    val lastSeenAt: Instant = Instant.now(),
    @Column("created_at")
    val createdAt: Instant = Instant.now(),
)
