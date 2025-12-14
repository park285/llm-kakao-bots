package party.qwer.twentyq.util.game.extensions

import party.qwer.twentyq.model.PendingMessage
import party.qwer.twentyq.util.common.formatting.UserIdFormatter

/**
 * 사용자 표시명 반환
 */
fun PendingMessage.displayName(
    chatId: String,
    anonymousName: String,
): String = UserIdFormatter.displayName(userId, sender, chatId, anonymousName)
