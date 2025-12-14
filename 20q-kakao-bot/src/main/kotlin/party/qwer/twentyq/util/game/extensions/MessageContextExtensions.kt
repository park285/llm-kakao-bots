package party.qwer.twentyq.util.game.extensions

import party.qwer.twentyq.bridge.MessageContext
import party.qwer.twentyq.util.common.formatting.UserIdFormatter

/**
 * 사용자 표시명 반환
 */
internal fun MessageContext.displayName(anonymousName: String): String =
    UserIdFormatter.displayName(
        userId,
        sender,
        chatId,
        anonymousName,
    )
