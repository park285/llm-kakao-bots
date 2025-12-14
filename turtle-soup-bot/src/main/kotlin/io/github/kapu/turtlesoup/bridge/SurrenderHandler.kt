package io.github.kapu.turtlesoup.bridge

import io.github.kapu.turtlesoup.models.SurrenderVote
import io.github.kapu.turtlesoup.service.GameService
import io.github.kapu.turtlesoup.service.SurrenderVoteService
import io.github.kapu.turtlesoup.utils.MessageKeys
import io.github.kapu.turtlesoup.utils.MessageProvider
import io.github.oshai.kotlinlogging.KotlinLogging

/** 포기/투표 핸들러 */
class SurrenderHandler(
    private val gameService: GameService,
    private val voteService: SurrenderVoteService,
    private val messageProvider: MessageProvider,
) {
    /**
     * 항복 합의 처리 (플레이어 수에 따라 자동 조건 변경)
     * - 1인: 즉시 항복
     * - 2인: 2명 동의 필요
     * - 3인+: 3명 동의 필요
     */
    suspend fun handleConsensus(
        chatId: String,
        userId: String,
    ): String {
        logger.info { "handle_surrender_consensus chat_id=$chatId user_id=$userId" }
        voteService.requireSession(chatId)

        val players = voteService.resolvePlayers(chatId)
        val activeVote = voteService.activeVote(chatId)

        return activeVote
            ?.let { vote -> activeVoteResponse(chatId, players, vote) }
            ?: newVoteResponse(chatId, userId, players)
    }

    /**
     * 포기 투표 동의
     */
    suspend fun handleAgree(
        chatId: String,
        userId: String,
    ): String {
        logger.info { "handle_agree chat_id=$chatId user_id=$userId" }
        voteService.requireSession(chatId)

        return when (val result = voteService.approve(chatId, userId)) {
            SurrenderVoteService.VoteApprovalResult.NotFound ->
                messageProvider.get(MessageKeys.VOTE_NOT_FOUND)

            SurrenderVoteService.VoteApprovalResult.NotEligible ->
                messageProvider.get(MessageKeys.VOTE_NOT_FOUND)

            SurrenderVoteService.VoteApprovalResult.AlreadyVoted ->
                messageProvider.get(MessageKeys.VOTE_ALREADY_VOTED)

            SurrenderVoteService.VoteApprovalResult.PersistenceFailure ->
                messageProvider.get(MessageKeys.ERROR_INTERNAL)

            is SurrenderVoteService.VoteApprovalResult.Progress ->
                inProgressMessage(result.vote)

            is SurrenderVoteService.VoteApprovalResult.Completed ->
                votePassedMessage(chatId, result.vote)
        }
    }

    private suspend fun activeVoteResponse(
        chatId: String,
        players: Set<String>,
        vote: SurrenderVote,
    ): String =
        if (players.size == 1) {
            logger.info { "single_player_surrender chat_id=$chatId" }
            voteService.clear(chatId)
            executeSurrender(chatId)
        } else {
            inProgressMessage(vote)
        }

    private suspend fun newVoteResponse(
        chatId: String,
        userId: String,
        players: Set<String>,
    ): String {
        return when (val result = voteService.startVote(chatId, userId, players)) {
            is SurrenderVoteService.VoteStartResult.Immediate -> {
                logger.info { "vote_immediate_pass chat_id=$chatId" }
                executeSurrender(chatId)
            }

            is SurrenderVoteService.VoteStartResult.Started ->
                messageProvider.get(
                    MessageKeys.VOTE_START,
                    "required" to result.vote.requiredApprovals().toString(),
                    "current" to result.vote.approvals.size.toString(),
                )
        }
    }

    private fun inProgressMessage(vote: SurrenderVote): String {
        val remain = vote.requiredApprovals() - vote.approvals.size
        return messageProvider.get(
            MessageKeys.VOTE_IN_PROGRESS,
            "current" to vote.approvals.size.toString(),
            "required" to vote.requiredApprovals().toString(),
            "remain" to remain.toString(),
        )
    }

    private suspend fun votePassedMessage(
        chatId: String,
        updated: SurrenderVote,
    ): String {
        logger.info {
            "vote_passed chat_id=$chatId " +
                "approvals=${updated.approvals.size}/${updated.requiredApprovals()}"
        }
        return messageProvider.get(MessageKeys.VOTE_PASSED) + "\n\n" + executeSurrender(chatId)
    }

    private suspend fun executeSurrender(chatId: String): String {
        val result = gameService.surrender(chatId)

        val hintBlock =
            if (result.hintsUsed.isNotEmpty()) {
                val header =
                    messageProvider.get(
                        MessageKeys.SURRENDER_HINT_BLOCK_HEADER,
                        "hintCount" to result.hintsUsed.size.toString(),
                    )
                val items =
                    result.hintsUsed.mapIndexed { index, hint ->
                        messageProvider.get(
                            MessageKeys.SURRENDER_HINT_ITEM,
                            "hintNumber" to (index + 1).toString(),
                            "content" to hint,
                        )
                    }.joinToString("\n")
                header + items
            } else {
                ""
            }

        return messageProvider.get(
            MessageKeys.SURRENDER_RESULT,
            "solution" to result.solution,
            "hintBlock" to hintBlock,
        )
    }

    companion object {
        private val logger = KotlinLogging.logger {}
    }
}
