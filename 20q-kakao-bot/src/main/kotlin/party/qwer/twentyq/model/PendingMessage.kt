package party.qwer.twentyq.model

import party.qwer.twentyq.util.common.extensions.nowMillis

data class PendingMessage(
    val userId: String,
    val content: String,
    val threadId: String?,
    val sender: String? = null,
    val timestamp: Long = nowMillis(),
    // 체인 질문 일괄 처리용 필드
    val isChainBatch: Boolean = false,
    val batchQuestions: List<String>? = null,
)
