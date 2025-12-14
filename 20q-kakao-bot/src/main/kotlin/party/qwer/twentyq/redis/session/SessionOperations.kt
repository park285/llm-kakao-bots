package party.qwer.twentyq.redis.session

import party.qwer.twentyq.api.dto.QuestionHistory
import party.qwer.twentyq.model.PlayerInfo
import party.qwer.twentyq.model.RiddleSecret
import party.qwer.twentyq.model.SurrenderVote

interface GameSessionOperations {
    suspend fun getQuiz(chatId: String): String?

    suspend fun getSecret(chatId: String): RiddleSecret?

    suspend fun saveSecret(
        chatId: String,
        secret: RiddleSecret,
    )

    suspend fun getHistory(chatId: String): List<QuestionHistory>

    suspend fun addHistory(
        chatId: String,
        questionNumber: Int,
        question: String,
        answer: String,
        isChain: Boolean = false,
        thoughtSignature: String? = null,
        userId: String? = null,
    )

    suspend fun updateHistoryAt(
        chatId: String,
        index: Int,
        questionNumber: Int,
        question: String,
        answer: String,
    )

    suspend fun getSelectedCategory(chatId: String): String?

    suspend fun saveSelectedCategory(
        chatId: String,
        category: String?,
    )
}

interface HintSessionOperations {
    suspend fun getHintCount(chatId: String): Int

    suspend fun incrementHintCount(chatId: String)

    suspend fun saveCandidateCount(
        chatId: String,
        count: Int,
    )

    suspend fun getCandidateCount(chatId: String): Int
}

interface TrackingSessionOperations {
    suspend fun getRecentTopics(
        chatId: String,
        limit: Int = 20,
    ): List<String>

    suspend fun getRecentTopicsByCategory(
        chatId: String,
        category: String,
        limit: Int = 20,
    ): List<String>

    suspend fun getBannedTopics(
        chatId: String,
        category: String?,
        limit: Int,
    ): List<String>

    suspend fun addCompletedTopic(
        chatId: String,
        category: String,
        topic: String,
    )

    suspend fun getPlayers(chatId: String): Set<PlayerInfo>

    /**
     * 닉네임으로 플레이어 조회
     */
    suspend fun getPlayerByNickname(
        chatId: String,
        nickname: String,
    ): PlayerInfo?

    suspend fun addWrongGuess(
        chatId: String,
        guess: String,
        userId: String,
    )

    suspend fun getWrongGuesses(chatId: String): List<String>
}

interface VotingSessionOperations {
    suspend fun getSurrenderVote(chatId: String): SurrenderVote?

    suspend fun saveSurrenderVote(
        chatId: String,
        vote: SurrenderVote,
        ttlSeconds: Long = 120,
    )

    suspend fun hasActiveSurrenderVote(chatId: String): Boolean

    suspend fun approveSurrender(
        chatId: String,
        userId: String,
        ttlSeconds: Long = 120,
    ): SurrenderVote?

    suspend fun clearSurrenderVote(chatId: String)
}
