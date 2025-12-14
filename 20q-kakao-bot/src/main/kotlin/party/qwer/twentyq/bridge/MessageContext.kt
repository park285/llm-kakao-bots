package party.qwer.twentyq.bridge

import party.qwer.twentyq.model.Command

// 메시지 처리 컨텍스트
internal data class MessageContext(
    val chatId: String,
    val userId: String,
    val content: String,
    val threadId: String?,
    val sender: String?,
    val command: Command?,
)
