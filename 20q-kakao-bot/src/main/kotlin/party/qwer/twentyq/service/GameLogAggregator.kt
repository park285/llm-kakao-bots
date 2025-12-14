package party.qwer.twentyq.service

import kotlinx.coroutines.flow.Flow
import org.springframework.stereotype.Component
import party.qwer.twentyq.model.GameLogEntity
import party.qwer.twentyq.model.ParticipantActivity
import party.qwer.twentyq.util.game.GameMessageProvider

/**
 * 게임 로그 집계 담당 클래스
 */
@Component
class GameLogAggregator(
    private val messageProvider: GameMessageProvider,
) {
    companion object {
        private const val USER_ID_DISPLAY_LENGTH = 4
        private const val PERCENTAGE_MULTIPLIER = 100
    }

    /**
     * 게임 로그 집계 (sender별 참여 횟수, 총 게임 수, 정답 수)
     */
    suspend fun aggregateGameLogs(logs: Flow<GameLogEntity>): AggregationResult {
        val senderGameCounts = mutableMapOf<String, Int>()
        var totalGames = 0
        var totalCorrect = 0

        logs.collect { log ->
            totalGames++
            if (log.result == "CORRECT") {
                totalCorrect++
            }
            val sender =
                log.sender.ifBlank {
                    messageProvider.get("user.anonymous_id", "id" to log.userId.takeLast(USER_ID_DISPLAY_LENGTH))
                }
            senderGameCounts[sender] = (senderGameCounts[sender] ?: 0) + 1
        }

        return AggregationResult(senderGameCounts, totalGames, totalCorrect)
    }

    /**
     * 참여자 활동 목록 생성
     */
    fun buildParticipantActivities(senderGameCounts: Map<String, Int>): List<ParticipantActivity> =
        senderGameCounts
            .map { (sender, count) ->
                ParticipantActivity(
                    sender = sender,
                    gamesPlayed = count,
                )
            }.sortedByDescending { it.gamesPlayed }

    /**
     * 완주율 계산
     */
    fun calculateCompletionRate(
        totalGames: Int,
        totalCorrect: Int,
    ): Int =
        if (totalGames > 0) {
            (totalCorrect.toDouble() / totalGames * PERCENTAGE_MULTIPLIER).toInt()
        } else {
            0
        }

    data class AggregationResult(
        val senderGameCounts: Map<String, Int>,
        val totalGames: Int,
        val totalCorrect: Int,
    )
}
