package party.qwer.twentyq.service

// 텍스트 복잡도 통계
internal data class ComplexityStats(
    val unknownRatio: Double,
    val avgTokenLength: Double,
    val hasIncompleteHangul: Boolean,
    val hasSingleJamo: Boolean,
    val hasJamoWithSpecialChars: Boolean,
    val hangulRatio: Double,
    val contentRatio: Double,
    val tokenCount: Int,
    val textLength: Int,
)
