package party.qwer.twentyq.util.game.extensions

import party.qwer.twentyq.model.RiddleSession
import party.qwer.twentyq.util.logging.LoggingConstants.PERCENT_MULTIPLIER

// RiddleSession 상태 검증 확장 함수 (10개)

fun RiddleSession.isExpired(maxQuestions: Int): Boolean = questionCount >= maxQuestions

fun RiddleSession.isActive(maxQuestions: Int): Boolean = !isExpired(maxQuestions)

fun RiddleSession.remainingQuestions(maxQuestions: Int): Int = (maxQuestions - questionCount).coerceAtLeast(0)

fun RiddleSession.canUseHint(maxHints: Int): Boolean = hintCount < maxHints

fun RiddleSession.remainingHints(maxHints: Int): Int = (maxHints - hintCount).coerceAtLeast(0)

fun RiddleSession.progress(maxQuestions: Int): Double =
    if (maxQuestions <= 0) {
        0.0
    } else {
        questionCount.toDouble() / maxQuestions.toDouble()
    }

fun RiddleSession.progressPercent(maxQuestions: Int): Int = (progress(maxQuestions) * PERCENT_MULTIPLIER).toInt()

fun RiddleSession.hasCategory(): Boolean = selectedCategory != null

fun RiddleSession.hasNoCategory(): Boolean = selectedCategory == null

fun RiddleSession.isCategory(category: String): Boolean = selectedCategory == category
