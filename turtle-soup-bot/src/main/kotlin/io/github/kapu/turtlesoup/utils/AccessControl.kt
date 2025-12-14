package io.github.kapu.turtlesoup.utils

import io.github.kapu.turtlesoup.config.AccessConfig
import io.github.oshai.kotlinlogging.KotlinLogging

/** 접근 제어 검증 */
class AccessControl(
    private val config: AccessConfig,
) {
    private val allowedChatIds = config.allowedChatIds.toSet()
    private val blockedChatIds = config.blockedChatIds.toSet()
    private val blockedUserIds = config.blockedUserIds.toSet()

    /**
     * 접근 거부 사유 반환 (null = 허용)
     */
    fun getDenialReason(
        userId: String,
        chatId: String,
    ): String? =
        when {
            config.passthrough -> null
            isUserBlocked(userId) -> MessageKeys.ERROR_USER_BLOCKED
            !config.enabled -> null
            isChatBlocked(chatId) -> MessageKeys.ERROR_CHAT_BLOCKED
            else -> checkAllowedList(chatId)
        }.also { reason ->
            if (reason != null) {
                logger.debug { "access_denied user_id=$userId chat_id=$chatId reason=$reason" }
            }
        }

    /**
     * 접근 허용 여부
     */
    fun isAllowed(
        userId: String,
        chatId: String,
    ): Boolean = getDenialReason(userId, chatId) == null

    private fun isUserBlocked(userId: String): Boolean = blockedUserIds.contains(userId)

    private fun isChatBlocked(chatId: String): Boolean = blockedChatIds.contains(chatId)

    private fun checkAllowedList(chatId: String): String? {
        if (allowedChatIds.isEmpty()) return null
        if (allowedChatIds.contains(chatId)) return null
        return MessageKeys.ERROR_ACCESS_DENIED
    }

    companion object {
        private val logger = KotlinLogging.logger {}
    }
}
