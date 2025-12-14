package party.qwer.twentyq.model

import org.springframework.data.annotation.Id
import org.springframework.data.annotation.Transient
import org.springframework.data.annotation.Version
import org.springframework.data.domain.Persistable
import org.springframework.data.relational.core.mapping.Column
import org.springframework.data.relational.core.mapping.Table
import java.time.Instant

/**
 * 사용자 스탯 Entity 
 */
@Table("user_stats")
data class UserStatsEntity(
    @Id
    @Column("id")
    private val id: String,
    @Column("chat_id")
    val chatId: String,
    @Column("user_id")
    val userId: String,
    @Column("total_games_started")
    val totalGamesStarted: Int = 0,
    @Column("total_games_completed")
    val totalGamesCompleted: Int = 0,
    @Column("total_surrenders")
    val totalSurrenders: Int = 0,
    @Column("total_questions_asked")
    val totalQuestionsAsked: Int = 0,
    @Column("total_hints_used")
    val totalHintsUsed: Int = 0,
    @Column("total_wrong_guesses")
    val totalWrongGuesses: Int = 0,
    @Column("best_score_question_count")
    val bestScoreQuestionCount: Int? = null,
    @Column("best_score_wrong_guess_count")
    val bestScoreWrongGuessCount: Int? = null,
    @Column("best_score_target")
    val bestScoreTarget: String? = null,
    @Column("best_score_category")
    val bestScoreCategory: String? = null,
    @Column("best_score_achieved_at")
    val bestScoreAchievedAt: Instant? = null,
    @Column("category_stats_json")
    val categoryStatsJson: String? = null,
    @Column("created_at")
    val createdAt: Instant = Instant.now(),
    @Column("updated_at")
    val updatedAt: Instant = Instant.now(),
    @Version
    @Column("version")
    val version: Long = 0,
) : Persistable<String> {
    @Transient
    private var new: Boolean = false

    override fun getId(): String = id

    override fun isNew(): Boolean = new

    /**
     * INSERT용으로 마킹
     */
    fun markAsNew() = apply { new = true }

    /**
     * Entity를 DTO로 변환
     */
    fun toUserStats(categoryStats: Map<String, CategoryStat> = emptyMap()): UserStats =
        UserStats(
            userId = userId,
            totalGamesStarted = totalGamesStarted,
            totalGamesCompleted = totalGamesCompleted,
            totalSurrenders = totalSurrenders,
            totalQuestionsAsked = totalQuestionsAsked,
            totalHintsUsed = totalHintsUsed,
            totalWrongGuesses = totalWrongGuesses,
            bestScore = buildBestScore(),
            categoryStats = categoryStats,
        )

    // 베스트 스코어 필수 필드 존재 여부 확인
    private fun hasAllRequiredBestScoreFields(): Boolean =
        bestScoreQuestionCount != null &&
            bestScoreTarget != null &&
            bestScoreCategory != null &&
            bestScoreAchievedAt != null

    /**
     * BestScore 생성
     */
    private fun buildBestScore(): BestScore? {
        if (!hasAllRequiredBestScoreFields()) {
            return null
        }

        return BestScore(
            questionCount = bestScoreQuestionCount!!, // null check already done
            wrongGuessCount = bestScoreWrongGuessCount ?: 0,
            target = bestScoreTarget!!, // null check already done
            category = bestScoreCategory!!, // null check already done
            achievedAt = bestScoreAchievedAt!!, // null check already done
        )
    }
}
