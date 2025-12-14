package party.qwer.twentyq.service.riddle

object HintPolicy {
    private const val MAX_HINTS_TOTAL = 3
    private const val MAX_HINTS_PER_REQUEST = 1

    /** 남은 힌트 수 계산 (총 한도 - 사용량, 요청당 한도 제한) */
    fun computeMaxHints(hintCount: Int): Int {
        val remaining = (MAX_HINTS_TOTAL - hintCount).coerceAtLeast(0)
        return remaining.coerceAtMost(MAX_HINTS_PER_REQUEST)
    }
}
