package party.qwer.twentyq.bridge

// Lock 획득 결과
internal data class LockContext(
    val token: String,
    val requiresWrite: Boolean,
)
