package io.github.kapu.turtlesoup.models

import kotlinx.serialization.Serializable
import java.time.Instant

/**
 * 항복 투표
 */
@Serializable
data class SurrenderVote(
    val initiator: String,
    val eligiblePlayers: Set<String>,
    val approvals: Set<String> = emptySet(),
    val createdAt: Long = Instant.now().toEpochMilli(),
) {
    // 필요 찬성 인원: 1명=1, 2명=2, 3+명=3
    fun requiredApprovals(): Int {
        val playerCount = eligiblePlayers.size
        return when {
            playerCount <= 1 -> 1
            playerCount == 2 -> 2
            else -> REQUIRED_APPROVALS_MANY
        }
    }

    fun isApproved(): Boolean = approvals.size >= requiredApprovals()

    fun canVote(userId: String): Boolean = userId in eligiblePlayers

    fun hasVoted(userId: String): Boolean = userId in approvals

    fun approve(userId: String): SurrenderVote {
        require(canVote(userId)) { "User $userId is not eligible to vote" }
        return copy(approvals = approvals + userId)
    }

    companion object {
        private const val REQUIRED_APPROVALS_MANY = 3
    }
}
