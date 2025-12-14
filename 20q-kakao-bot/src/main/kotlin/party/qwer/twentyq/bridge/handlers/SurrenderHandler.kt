package party.qwer.twentyq.bridge.handlers

import org.slf4j.LoggerFactory
import org.springframework.stereotype.Component
import party.qwer.twentyq.config.AppProperties
import party.qwer.twentyq.logging.infoL
import party.qwer.twentyq.model.SurrenderVote
import party.qwer.twentyq.redis.RiddleSessionRepository
import party.qwer.twentyq.service.RiddleService
import party.qwer.twentyq.util.common.security.requireSessionOrThrow
import party.qwer.twentyq.util.game.GameMessageProvider

@Component
class SurrenderHandler(
    private val riddleService: RiddleService,
    private val sessionRepo: RiddleSessionRepository,
    private val messageProvider: GameMessageProvider,
    private val appProperties: AppProperties,
) {
    companion object {
        private val log = LoggerFactory.getLogger(SurrenderHandler::class.java)
        private const val MSG_PARAM_CURRENT = "current"
        private const val MSG_PARAM_REQUIRED = "required"
        private const val MSG_PARAM_REMAIN = "remain"
        private const val MSG_PARAM_PREFIX = "prefix"
    }

    private fun inProgressMessage(vote: SurrenderVote): String {
        val remain = vote.requiredApprovals() - vote.approvals.size
        return messageProvider.get(
            "vote.in_progress",
            MSG_PARAM_CURRENT to vote.approvals.size,
            MSG_PARAM_REQUIRED to vote.requiredApprovals(),
            MSG_PARAM_REMAIN to remain,
            MSG_PARAM_PREFIX to appProperties.commands.prefix,
        )
    }

    private suspend fun activeVoteResponse(
        chatId: String,
        players: Set<String>,
    ): String {
        val vote = sessionRepo.getSurrenderVote(chatId) ?: return messageProvider.get("vote.already_active")
        return if (players.size == 1) {
            log.infoL { "CONSENSUS_FALLBACK_IMMEDIATE chatId=$chatId, players=${players.size}" }
            val result = riddleService.surrender(chatId)
            sessionRepo.clearSurrenderVote(chatId)
            result
        } else {
            sessionRepo.saveSurrenderVote(chatId, vote)
            inProgressMessage(vote)
        }
    }

    private suspend fun newVoteResponse(
        chatId: String,
        userId: String,
        players: Set<String>,
    ): String {
        val vote =
            SurrenderVote(
                initiator = userId,
                eligiblePlayers = players,
                approvals = setOf(userId),
            )
        return if (vote.isApproved()) {
            log.infoL {
                "CONSENSUS_REACHED_IMMEDIATE chatId=$chatId, " +
                    "approvals=\${vote.approvals.size}, needed=\${vote.requiredApprovals()}"
            }
            riddleService.surrender(chatId)
        } else {
            sessionRepo.saveSurrenderVote(chatId, vote)
            messageProvider.get(
                "vote.start",
                MSG_PARAM_REQUIRED to vote.requiredApprovals(),
                MSG_PARAM_CURRENT to vote.approvals.size,
                MSG_PARAM_PREFIX to appProperties.commands.prefix,
            )
        }
    }

    suspend fun handleConsensus(
        chatId: String,
        userId: String,
    ): String {
        log.infoL { "HANDLE_SURRENDER_CONSENSUS chatId=$chatId, userId=$userId" }
        riddleService.requireSessionOrThrow(chatId)

        val players = sessionRepo.getPlayers(chatId).map { it.userId }.toSet()
        return if (sessionRepo.hasActiveSurrenderVote(chatId)) {
            activeVoteResponse(chatId, players)
        } else {
            newVoteResponse(chatId, userId, players)
        }
    }

    suspend fun handleAgree(
        chatId: String,
        userId: String,
    ): String {
        log.infoL { "HANDLE_SURRENDER_AGREE chatId=$chatId, userId=$userId" }
        riddleService.requireSessionOrThrow(chatId)

        val response =
            when (val vote = sessionRepo.getSurrenderVote(chatId)) {
                null -> messageProvider.get("vote.not_found", MSG_PARAM_PREFIX to appProperties.commands.prefix)
                else -> {
                    if (!vote.canVote(userId)) {
                        messageProvider.get("vote.cannot_vote")
                    } else if (vote.hasVoted(userId)) {
                        messageProvider.get("vote.already_voted")
                    } else {
                        val updated =
                            sessionRepo.approveSurrender(chatId, userId)
                                ?: return messageProvider.get("vote.processing_failed")
                        if (updated.isApproved()) {
                            log.infoL {
                                "CONSENSUS_REACHED chatId=$chatId, " +
                                    "approvals=\${updated.approvals.size}, needed=\${updated.requiredApprovals()}"
                            }
                            val result = riddleService.surrender(chatId)
                            sessionRepo.clearSurrenderVote(chatId)
                            result
                        } else {
                            val remain = updated.requiredApprovals() - updated.approvals.size
                            messageProvider.get(
                                "vote.agree_progress",
                                MSG_PARAM_CURRENT to updated.approvals.size,
                                MSG_PARAM_REQUIRED to updated.requiredApprovals(),
                                MSG_PARAM_REMAIN to remain,
                            )
                        }
                    }
                }
            }

        return response
    }

    suspend fun handleReject(
        chatId: String,
        userId: String,
    ): String {
        log.info("HANDLE_SURRENDER_REJECT chatId={}, userId={}", chatId, userId)
        riddleService.requireSessionOrThrow(chatId)
        return messageProvider.get("vote.reject_not_supported")
    }

    suspend fun handle(chatId: String): String {
        log.info("HANDLE_SURRENDER chatId={}", chatId)
        riddleService.requireSessionOrThrow(chatId)
        return riddleService.surrender(chatId)
    }
}
