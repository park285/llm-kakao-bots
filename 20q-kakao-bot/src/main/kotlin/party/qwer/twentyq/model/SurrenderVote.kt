package party.qwer.twentyq.model

import party.qwer.twentyq.util.common.extensions.nowMillis

// 항복 투표
class SurrenderVote(
    val initiator: String,
    val eligiblePlayers: Set<String>,
    val approvals: Set<String> = emptySet(),
    val createdAt: Long = nowMillis(),
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

    fun copy(
        initiator: String = this.initiator,
        eligiblePlayers: Set<String> = this.eligiblePlayers,
        approvals: Set<String> = this.approvals,
        createdAt: Long = this.createdAt,
    ): SurrenderVote = SurrenderVote(initiator, eligiblePlayers, approvals, createdAt)

    fun approve(userId: String): SurrenderVote {
        require(canVote(userId)) { "User $userId is not eligible to vote" }
        return copy(approvals = approvals + userId)
    }

    companion object {
        private const val REQUIRED_APPROVALS_MANY = 3
    }
}
