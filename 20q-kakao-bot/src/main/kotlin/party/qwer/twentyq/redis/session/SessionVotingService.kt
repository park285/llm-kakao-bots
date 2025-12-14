package party.qwer.twentyq.redis.session

import org.slf4j.LoggerFactory
import org.springframework.stereotype.Component
import party.qwer.twentyq.logging.debugL
import party.qwer.twentyq.model.SurrenderVote
import party.qwer.twentyq.redis.voting.SurrenderVoteStore

/**
 * 항복 투표 데이터 관리
 */
@Component
class SessionVotingService(
    private val surrenderVoteStore: SurrenderVoteStore,
) : VotingSessionOperations {
    companion object {
        private val log = LoggerFactory.getLogger(SessionVotingService::class.java)
    }

    override suspend fun getSurrenderVote(chatId: String): SurrenderVote? = surrenderVoteStore.getAsync(chatId)

    override suspend fun saveSurrenderVote(
        chatId: String,
        vote: SurrenderVote,
        ttlSeconds: Long,
    ) {
        surrenderVoteStore.saveAsync(chatId, vote, ttlSeconds)
        log.debugL {
            "VALKEY saveSurrenderVote chatId=$chatId vote=$vote ttl=${ttlSeconds}s"
        }
    }

    override suspend fun hasActiveSurrenderVote(chatId: String): Boolean = surrenderVoteStore.isActiveAsync(chatId)

    override suspend fun approveSurrender(
        chatId: String,
        userId: String,
        ttlSeconds: Long,
    ): SurrenderVote? = surrenderVoteStore.approveAsync(chatId, userId, ttlSeconds)

    override suspend fun clearSurrenderVote(chatId: String) {
        surrenderVoteStore.clearAsync(chatId)
        log.debugL { "VALKEY clearSurrenderVote chatId=$chatId" }
    }
}
