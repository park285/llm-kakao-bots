package party.qwer.twentyq.util.common.security

import org.slf4j.Logger
import party.qwer.twentyq.config.properties.Access
import party.qwer.twentyq.service.RiddleService
import party.qwer.twentyq.service.exception.PermissionDeniedException
import party.qwer.twentyq.service.exception.SessionNotFoundException

suspend fun RiddleService.requireSessionOrThrow(chatId: String) {
    if (!this.hasSession(chatId)) throw SessionNotFoundException()
}

fun requireAdminOrThrow(
    adminUserIds: Collection<String>,
    userId: String,
    chatId: String,
    logger: Logger,
    warnMessage: String,
    warnArgs: Array<Any?> = arrayOf(chatId, userId),
    errorMessage: String? = null,
) {
    if (userId !in adminUserIds) {
        logger.warn(warnMessage, *warnArgs)
        if (errorMessage != null) {
            throw PermissionDeniedException(errorMessage)
        }
        throw PermissionDeniedException()
    }
}

fun resolveAccessDenialReason(
    access: Access,
    userId: String?,
    chatId: String?,
): String? =
    when {
        access.passthrough -> null
        isBlockedUser(access, userId) -> "error.user_blocked"
        !access.enabled -> null
        isBlockedChat(access, chatId) -> "error.chat_blocked"
        isAllowedChat(access, chatId) -> null
        else -> "error.access_denied"
    }

private fun isBlockedUser(
    access: Access,
    userId: String?,
): Boolean = !userId.isNullOrBlank() && access.blockedUserIds.contains(userId)

private fun isBlockedChat(
    access: Access,
    chatId: String?,
): Boolean = !chatId.isNullOrBlank() && access.blockedChatIds.contains(chatId)

private fun isAllowedChat(
    access: Access,
    chatId: String?,
): Boolean {
    if (access.allowedChatIds.isEmpty()) {
        return true
    }
    if (chatId.isNullOrBlank()) {
        return false
    }
    return access.allowedChatIds.contains(chatId)
}
