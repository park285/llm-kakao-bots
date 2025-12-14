package party.qwer.twentyq.model

import party.qwer.twentyq.service.StatsPeriod

/**
 * 방 전체 통계 결과 (순수 활동량만 표시, 경쟁 지표 제외)
 */
data class RoomStatsResult(
    val period: StatsPeriod,
    val totalGames: Int,
    val totalParticipants: Int,
    val completionRate: Int = 0, // 완주율 (%)
    val participantActivities: List<ParticipantActivity>,
)

/**
 * 참여자별 활동량 (경쟁 없이 참여도만 표시)
 */
data class ParticipantActivity(
    val sender: String,
    val gamesPlayed: Int,
)
