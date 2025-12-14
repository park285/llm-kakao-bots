package party.qwer.twentyq.util.common.formatting

object UserIdFormatter {
    fun formatForDisplay(
        userId: String,
        chatId: String,
        anonymousName: String,
    ): String = if (userId.isBlank() || userId == chatId) anonymousName else userId

    /**
     * 사용자 표시명 반환 (sender 우선, 없으면 userId 포맷팅)
     */
    fun displayName(
        userId: String,
        sender: String?,
        chatId: String,
        anonymousName: String,
    ): String = sender ?: formatForDisplay(userId, chatId, anonymousName)
}
