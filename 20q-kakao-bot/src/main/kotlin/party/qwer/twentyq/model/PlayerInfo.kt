package party.qwer.twentyq.model

import party.qwer.twentyq.util.common.formatting.UserIdFormatter

/**
 * 플레이어 정보 (방별 참여자 추적용)
 *
 * 참여 기준: 해당 방에서 게임 관련 메시지를 보낸 사용자
 * (게임 완료 여부와 무관)
 */
class PlayerInfo(
    val userId: String,
    val sender: String,
) {
    /**
     * 표시용 닉네임 반환
     */
    fun displayName(anonymousName: String = "누군가"): String =
        UserIdFormatter.displayName(
            userId = userId,
            sender = sender.ifBlank { null },
            chatId = "",
            anonymousName = anonymousName,
        )

    companion object {
        private const val USER_ID_DISPLAY_LENGTH = 4
    }

    // Set 중복 체크는 userId만 기준
    override fun equals(other: Any?): Boolean {
        if (this === other) return true
        if (other !is PlayerInfo) return false
        return userId == other.userId
    }

    override fun hashCode(): Int = userId.hashCode()
}
