package party.qwer.twentyq.service

// 복잡도 점수 누적기
internal class ComplexityAccumulator(
    var score: Int = 0,
    val reasons: MutableList<String> = mutableListOf(),
) {
    fun add(
        points: Int,
        reason: String,
    ) {
        if (points > 0) {
            score += points
            reasons.add(reason)
        }
    }
}
