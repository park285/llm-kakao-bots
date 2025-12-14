package party.qwer.twentyq.redis.session

import org.slf4j.LoggerFactory
import org.springframework.stereotype.Component
import party.qwer.twentyq.logging.debugL
import party.qwer.twentyq.model.PlayerInfo
import party.qwer.twentyq.redis.tracking.PlayerSetStore
import party.qwer.twentyq.redis.tracking.TopicHistoryStore
import party.qwer.twentyq.redis.tracking.WrongGuessSetStore
import party.qwer.twentyq.repository.GameLogRepository
import party.qwer.twentyq.repository.UserNicknameMapRepository

/**
 * 플레이어/토픽/오답 추적 데이터 관리
 */
@Component
class SessionTrackingService(
    private val topicHistoryStore: TopicHistoryStore,
    private val playerSetStore: PlayerSetStore,
    private val wrongGuessSetStore: WrongGuessSetStore,
    private val gameLogRepository: GameLogRepository,
    private val userNicknameMapRepository: UserNicknameMapRepository,
) : TrackingSessionOperations {
    companion object {
        private val log = LoggerFactory.getLogger(SessionTrackingService::class.java)
    }

    override suspend fun getRecentTopics(
        chatId: String,
        limit: Int,
    ): List<String> {
        val topics = topicHistoryStore.getRecentAsync(chatId, limit)
        log.debugL {
            "VALKEY getRecentTopics chatId=$chatId limit=$limit found=${topics.size}"
        }
        return topics
    }

    override suspend fun getRecentTopicsByCategory(
        chatId: String,
        category: String,
        limit: Int,
    ): List<String> {
        val topics = topicHistoryStore.getRecentAsync(chatId, category, limit)
        log.debugL {
            "VALKEY getRecentTopicsByCategory chatId=$chatId category=$category limit=$limit found=${topics.size}"
        }
        return topics
    }

    override suspend fun getBannedTopics(
        chatId: String,
        category: String?,
        limit: Int,
    ): List<String> {
        val banned = topicHistoryStore.getBannedTopics(chatId, category, limit)
        log.debugL {
            "VALKEY getBannedTopics chatId=$chatId category=$category limit=$limit found=${banned.size}"
        }
        return banned
    }

    override suspend fun addCompletedTopic(
        chatId: String,
        category: String,
        topic: String,
    ) {
        topicHistoryStore.addAsync(chatId, category, topic)
        log.debugL {
            "VALKEY addCompletedTopic chatId=$chatId category='$category', topic='$topic'"
        }
    }

    override suspend fun getPlayers(chatId: String): Set<PlayerInfo> = playerSetStore.getAllAsync(chatId)

    override suspend fun getPlayerByNickname(
        chatId: String,
        nickname: String,
    ): PlayerInfo? {
        // Valkey 캐시 우선 조회
        val players = playerSetStore.getAllAsync(chatId)
        val cached = players.find { it.sender.equals(nickname, ignoreCase = true) }
        if (cached != null) {
            return cached
        }

        // 닉네임 매핑 테이블 조회 (로그 미보존 시에도 사용)
        val mappedPlayer = userNicknameMapRepository.findLatestByNickname(chatId, nickname)
        val mapped =
            mappedPlayer?.let {
                log.debugL {
                    "PLAYER_NICKNAME_MAP_HIT chatId=$chatId nickname=$nickname userId=${it.userId}"
                }
                PlayerInfo(userId = it.userId, sender = it.lastSender)
            }
        if (mapped != null) {
            return mapped
        }

        // 캐시/매핑 미스 시 DB 폴백 (기록 로그)
        val gameLog = gameLogRepository.findByChatIdAndSender(chatId, nickname)
        return gameLog?.let {
            log.debugL { "PLAYER_DB_FALLBACK chatId=$chatId nickname=$nickname userId=${it.userId}" }
            PlayerInfo(userId = it.userId, sender = it.sender)
        }
    }

    override suspend fun addWrongGuess(
        chatId: String,
        guess: String,
        userId: String,
    ) {
        wrongGuessSetStore.addAsync(chatId, guess, userId)
        log.debugL { "VALKEY addWrongGuess chatId=$chatId userId=$userId guess=$guess" }
    }

    override suspend fun getWrongGuesses(chatId: String): List<String> {
        val guesses = wrongGuessSetStore.getSessionWrongGuessesAsync(chatId)
        log.debugL { "VALKEY getWrongGuesses chatId=$chatId found=${guesses.size}" }
        return guesses
    }

    suspend fun clearTopicHistory(chatId: String) {
        topicHistoryStore.clearAllAsync(chatId)
        log.debugL { "VALKEY clearTopicHistory chatId=$chatId" }
    }
}
