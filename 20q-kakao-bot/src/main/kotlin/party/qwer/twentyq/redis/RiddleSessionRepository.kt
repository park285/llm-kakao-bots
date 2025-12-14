package party.qwer.twentyq.redis

import kotlinx.coroutines.coroutineScope
import kotlinx.coroutines.launch
import org.slf4j.LoggerFactory
import org.springframework.stereotype.Repository
import party.qwer.twentyq.config.AppProperties
import party.qwer.twentyq.logging.debugL
import party.qwer.twentyq.logging.infoL
import party.qwer.twentyq.model.RiddleSecret
import party.qwer.twentyq.redis.session.CategoryStore
import party.qwer.twentyq.redis.session.GameSessionOperations
import party.qwer.twentyq.redis.session.HintSessionOperations
import party.qwer.twentyq.redis.session.HistoryStore
import party.qwer.twentyq.redis.session.SessionGameService
import party.qwer.twentyq.redis.session.SessionHintService
import party.qwer.twentyq.redis.session.SessionStore
import party.qwer.twentyq.redis.session.SessionTrackingService
import party.qwer.twentyq.redis.session.SessionVotingService
import party.qwer.twentyq.redis.session.TrackingSessionOperations
import party.qwer.twentyq.redis.session.VotingSessionOperations
import party.qwer.twentyq.redis.tracking.HintCountStore
import party.qwer.twentyq.redis.tracking.PlayerSetStore
import party.qwer.twentyq.redis.tracking.WrongGuessSetStore
import party.qwer.twentyq.util.common.extensions.minutes

/** 세션 데이터 Facade: 게임/힌트/추적/투표 서비스 조율 */
@Repository
class RiddleSessionRepository(
    private val gameService: SessionGameService,
    private val hintService: SessionHintService,
    private val trackingService: SessionTrackingService,
    private val votingService: SessionVotingService,
    private val sessionStore: SessionStore,
    private val historyStore: HistoryStore,
    private val hintCountStore: HintCountStore,
    private val categoryStore: CategoryStore,
    private val playerSetStore: PlayerSetStore,
    private val wrongGuessSetStore: WrongGuessSetStore,
    private val appProperties: AppProperties,
) : GameSessionOperations by gameService,
    HintSessionOperations by hintService,
    TrackingSessionOperations by trackingService,
    VotingSessionOperations by votingService {
    companion object {
        private val log = LoggerFactory.getLogger(RiddleSessionRepository::class.java)
    }

    override suspend fun getQuiz(chatId: String): String? {
        val quiz = gameService.getQuiz(chatId)
        if (quiz != null) {
            refreshSessionTtl(chatId)
        }
        return quiz
    }

    override suspend fun getSecret(chatId: String): RiddleSecret? {
        val secret = gameService.getSecret(chatId)
        if (secret != null) {
            refreshSessionTtl(chatId)
        }
        return secret
    }

    // ============ Coordinating Functions (3개 함수) ============

    suspend fun delete(chatId: String) {
        coroutineScope {
            launch { sessionStore.deleteAsync(chatId) }
            launch { historyStore.clearAsync(chatId) }
            launch { hintCountStore.deleteAsync(chatId) }
            launch { categoryStore.saveAsync(chatId, null) }
            launch { playerSetStore.clearAsync(chatId) }
            launch { wrongGuessSetStore.deleteAsync(chatId) }
            launch { votingService.clearSurrenderVote(chatId) }
            launch { hintService.saveCandidateCount(chatId, 0) }
        }
        log.debugL { "VALKEY delete chatId=$chatId" }
    }

    suspend fun clearAllData(chatId: String) {
        coroutineScope {
            launch { delete(chatId) }
            launch { trackingService.clearTopicHistory(chatId) }
        }
        log.infoL { "VALKEY clearAllData COMPLETE chatId=$chatId" }
    }

    suspend fun getAllPlayerIds(chatId: String): List<String> {
        val players = playerSetStore.getAllAsync(chatId)
        return players.map { it.userId }
    }

    private suspend fun refreshSessionTtl(chatId: String) {
        // 12시간 고정 TTL
        val ttl = appProperties.cache.veryLongTtlMinutes.minutes

        coroutineScope {
            launch { sessionStore.setTtlAsync(chatId, ttl) }
            launch { historyStore.setTtlAsync(chatId, ttl) }
            launch { hintCountStore.setTtl(chatId, ttl) }
            launch { categoryStore.setTtlAsync(chatId, ttl) }
            launch { playerSetStore.setTtlAsync(chatId, ttl) }
            launch { wrongGuessSetStore.setTtlAsync(chatId, ttl) }
            launch { hintService.saveCandidateCount(chatId, 0) }
        }

        log.debugL {
            "VALKEY TTL_REFRESH chatId=$chatId ttl=${ttl.toHours()}시간"
        }
    }
}
