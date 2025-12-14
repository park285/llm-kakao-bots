package io.github.kapu.turtlesoup.service

import io.github.kapu.turtlesoup.models.GameState
import io.github.kapu.turtlesoup.models.SurrenderVote
import io.github.kapu.turtlesoup.redis.SessionStore
import io.github.kapu.turtlesoup.redis.SurrenderVoteStore
import io.github.kapu.turtlesoup.utils.SessionNotFoundException

class SurrenderVoteService(
    private val sessionStore: SessionStore,
    private val voteStore: SurrenderVoteStore,
    private val sessionValidator: SessionValidator,
) {
    suspend fun requireSession(chatId: String) {
        sessionValidator.requireSession(chatId)
    }

    suspend fun resolvePlayers(chatId: String): Set<String> {
        val state = loadState(chatId)
        return playersOrFallback(state)
    }

    suspend fun activeVote(chatId: String): SurrenderVote? = voteStore.get(chatId)

    suspend fun startVote(
        chatId: String,
        initiator: String,
        players: Set<String>,
    ): VoteStartResult {
        val vote =
            SurrenderVote(
                initiator = initiator,
                eligiblePlayers = players,
                approvals = setOf(initiator),
            )

        return if (vote.isApproved()) {
            VoteStartResult.Immediate(vote)
        } else {
            voteStore.save(chatId, vote)
            VoteStartResult.Started(vote)
        }
    }

    suspend fun approve(
        chatId: String,
        userId: String,
    ): VoteApprovalResult {
        val vote = voteStore.get(chatId) ?: return VoteApprovalResult.NotFound
        val validation = validateApproval(vote, userId)
        if (validation != null) return validation

        val updated = voteStore.approve(chatId, userId)
        val result =
            when {
                updated == null -> VoteApprovalResult.PersistenceFailure
                updated.isApproved() -> {
                    voteStore.clear(chatId)
                    VoteApprovalResult.Completed(updated)
                }
                else -> VoteApprovalResult.Progress(updated)
            }

        return result
    }

    suspend fun clear(chatId: String) {
        voteStore.clear(chatId)
    }

    private suspend fun loadState(chatId: String): GameState {
        return sessionStore.loadGameState(chatId) ?: throw SessionNotFoundException(chatId)
    }

    private fun playersOrFallback(state: GameState): Set<String> {
        return if (state.players.isEmpty()) {
            setOf(state.userId)
        } else {
            state.players
        }
    }

    private fun validateApproval(
        vote: SurrenderVote,
        userId: String,
    ): VoteApprovalResult? {
        return when {
            !vote.canVote(userId) -> VoteApprovalResult.NotEligible
            vote.hasVoted(userId) -> VoteApprovalResult.AlreadyVoted
            else -> null
        }
    }

    sealed interface VoteStartResult {
        data class Immediate(val vote: SurrenderVote) : VoteStartResult

        data class Started(val vote: SurrenderVote) : VoteStartResult
    }

    sealed interface VoteApprovalResult {
        data class Completed(val vote: SurrenderVote) : VoteApprovalResult

        data class Progress(val vote: SurrenderVote) : VoteApprovalResult

        object NotFound : VoteApprovalResult

        object NotEligible : VoteApprovalResult

        object AlreadyVoted : VoteApprovalResult

        object PersistenceFailure : VoteApprovalResult
    }
}
